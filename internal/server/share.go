/* SPDX-License-Identifier: MIT */
package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/zhaojh329/rtty-go/proto"
	"github.com/zhaojh329/rttys/v5/internal/utils"
	"golang.org/x/crypto/ssh"
)

type shareRequest struct {
	Kind        string                `json:"kind"`
	Group       string                `json:"group"`
	DeviceID    string                `json:"deviceId"`
	Port        int                   `json:"port"`
	IdleSeconds int                   `json:"idleSeconds"`
	TargetIP    string                `json:"targetIp"`
	TargetPort  int                   `json:"targetPort"`
	Serial      *proto.SerialSettings `json:"serial"`
}

type shareInfo struct {
	ID           string                `json:"id"`
	Kind         string                `json:"kind"`
	Group        string                `json:"group"`
	DeviceID     string                `json:"deviceId"`
	Host         string                `json:"host"`
	Port         int                   `json:"port"`
	IdleSeconds  int                   `json:"idleSeconds"`
	IdleDeadline *time.Time            `json:"idleDeadline,omitempty"`
	Connections  int                   `json:"connections"`
	TargetIP     string                `json:"targetIp,omitempty"`
	TargetPort   int                   `json:"targetPort,omitempty"`
	Serial       *proto.SerialSettings `json:"serial,omitempty"`
	Fingerprint  string                `json:"fingerprint,omitempty"`
}

type shareManager struct {
	mu      sync.Mutex
	srv     *RttyServer
	entries map[string]*Share
}

type Share struct {
	manager *shareManager
	info    shareInfo
	dev     *Device
	ln      net.Listener
	signer  ssh.Signer
	secret  [32]byte
	ip      net.IP
	mu      sync.Mutex
	conns   map[net.Conn]bool
	active  int
	timer   *time.Timer
	gen     uint64
	closed  bool
	done    chan struct{}
}

const maxShareConnections = 64

func (m *shareManager) create(req shareRequest, host string) (*Share, string, error) {
	if err := req.validate(); err != nil {
		return nil, "", err
	}

	dev := m.srv.GetDevice(req.Group, req.DeviceID)
	if dev == nil {
		return nil, "", errors.New("device offline")
	}

	if (req.Kind == "serial" || req.Kind == "tcp") && dev.proto < 6 {
		return nil, "", errors.New("share type unsupported by device")
	}

	var ip net.IP
	if req.Kind == "tcp" {
		ip = net.ParseIP(req.TargetIP).To4()
	}

	cfg := m.srv.cfg
	if cfg.SharePortStart < 1 || cfg.SharePortEnd > 65535 || cfg.SharePortEnd < cfg.SharePortStart {
		return nil, "", errors.New("invalid share port range")
	}
	if req.Port != 0 && (req.Port < cfg.SharePortStart || req.Port > cfg.SharePortEnd) {
		return nil, "", errors.New("port outside share range")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var signer ssh.Signer
	var err error
	if req.Kind != "tcp" {
		signer, err = loadShareHostKey(cfg.ShareHostKey)
		if err != nil {
			return nil, "", fmt.Errorf("SSH host key: %w", err)
		}
	}
	var ln net.Listener
	port := req.Port
	if port != 0 {
		ln, err = net.Listen("tcp", net.JoinHostPort(cfg.ShareBindHost, strconv.Itoa(port)))
	} else {
		for p := cfg.SharePortStart; p <= cfg.SharePortEnd; p++ {
			ln, err = net.Listen("tcp", net.JoinHostPort(cfg.ShareBindHost, strconv.Itoa(p)))
			if err == nil {
				port = p
				break
			}
		}
	}
	if ln == nil {
		return nil, "", fmt.Errorf("no share port available: %w", err)
	}
	if dev.ctx.Err() != nil {
		ln.Close()
		return nil, "", errors.New("device disconnected")
	}
	if cfg.SharePublicHost != "" {
		host = cfg.SharePublicHost
	}
	s := &Share{manager: m, dev: dev, ln: ln, signer: signer, ip: ip, conns: make(map[net.Conn]bool), done: make(chan struct{})}
	s.info = req.info()
	s.info.ID = utils.GenUniqueID()
	s.info.Host = host
	s.info.Port = port

	password := ""
	if signer != nil {
		s.info.Fingerprint = ssh.FingerprintSHA256(signer.PublicKey())
		var raw [24]byte
		if _, err := rand.Read(raw[:]); err != nil {
			ln.Close()
			return nil, "", err
		}
		password = base64.RawURLEncoding.EncodeToString(raw[:])
		s.secret = sha256.Sum256([]byte(password))
	}
	m.entries[s.info.ID] = s
	s.scheduleIdleLocked()
	go s.accept()
	return s, password, nil
}

func (req *shareRequest) validate() error {
	if req.DeviceID == "" || len(req.DeviceID) > proto.MaximumDevIDLen || len(req.Group) > proto.MaximumGroupLen {
		return errors.New("invalid share device")
	}

	if req.Kind != "terminal" && req.Kind != "serial" && req.Kind != "tcp" {
		return errors.New("invalid share type")
	}

	if req.IdleSeconds == 0 {
		req.IdleSeconds = 60
	}

	if req.IdleSeconds < 60 || req.IdleSeconds > 3600 {
		return errors.New("idle timeout must be 60 to 3600 seconds")
	}

	if req.Kind == "serial" && (req.Serial == nil || !req.Serial.Valid()) {
		return errors.New("invalid serial settings")
	}

	if req.Kind == "tcp" {
		ip := net.ParseIP(req.TargetIP).To4()
		if ip == nil || req.TargetPort < 1 || req.TargetPort > 65535 {
			return errors.New("invalid TCP target")
		}

		req.TargetIP = ip.String()
	}

	return nil
}

func (req shareRequest) info() shareInfo {
	return shareInfo{Kind: req.Kind, Group: req.Group, DeviceID: req.DeviceID,
		Port: req.Port, IdleSeconds: req.IdleSeconds,
		TargetIP: req.TargetIP, TargetPort: req.TargetPort, Serial: req.Serial}
}

func (m *shareManager) list() []shareInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]shareInfo, 0, len(m.entries))
	for _, s := range m.entries {
		s.mu.Lock()
		if !s.closed {
			info := s.info
			info.Connections = s.active
			out = append(out, info)
		}
		s.mu.Unlock()
	}
	return out
}

func (s *Share) snapshot() shareInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	info := s.info
	info.Connections = s.active
	return info
}

func (m *shareManager) get(id string) *Share {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.entries[id]
}

func (m *shareManager) closeDevice(dev *Device) {
	m.mu.Lock()
	var shares []*Share
	for _, s := range m.entries {
		if s.dev == dev {
			shares = append(shares, s)
		}
	}
	m.mu.Unlock()
	for _, s := range shares {
		s.close()
	}
}

func (s *Share) scheduleIdleLocked() {
	s.gen++
	gen := s.gen
	deadline := time.Now().Add(time.Duration(s.info.IdleSeconds) * time.Second)
	s.info.IdleDeadline = &deadline
	if s.timer != nil {
		s.timer.Stop()
	}
	s.timer = time.AfterFunc(time.Until(deadline), func() {
		s.expire(gen)
	})
}

func (s *Share) track(conn net.Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || len(s.conns) >= maxShareConnections {
		return false
	}
	s.conns[conn] = false
	return true
}

func (s *Share) activate(conn net.Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	if !s.conns[conn] {
		s.conns[conn] = true
		s.active++
		s.gen++
		if s.timer != nil {
			s.timer.Stop()
		}
		s.info.IdleDeadline = nil
	}
	return true
}

func (s *Share) release(conn net.Conn) {
	conn.Close()
	s.mu.Lock()
	if active, ok := s.conns[conn]; ok {
		delete(s.conns, conn)
		if active {
			s.active--
			if s.active == 0 && !s.closed {
				s.scheduleIdleLocked()
			}
		}
	}
	s.mu.Unlock()
}

func (s *Share) deactivate(conn net.Conn) {
	s.mu.Lock()
	if s.conns[conn] {
		s.conns[conn] = false
		s.active--
		if s.active == 0 && !s.closed {
			s.scheduleIdleLocked()
		}
	}
	s.mu.Unlock()
}

func (s *Share) markClosedLocked() ([]net.Conn, bool) {
	if s.closed {
		return nil, false
	}
	s.closed = true
	close(s.done)
	s.gen++
	if s.timer != nil {
		s.timer.Stop()
	}
	conns := make([]net.Conn, 0, len(s.conns))
	for conn := range s.conns {
		conns = append(conns, conn)
	}
	return conns, true
}

func (s *Share) finishClosed(conns []net.Conn) {
	s.ln.Close()
	for _, conn := range conns {
		conn.Close()
	}
	s.manager.mu.Lock()
	if s.manager.entries[s.info.ID] == s {
		delete(s.manager.entries, s.info.ID)
	}
	s.manager.mu.Unlock()
}

func (s *Share) expire(gen uint64) {
	s.mu.Lock()
	if s.closed || s.active != 0 || s.gen != gen {
		s.mu.Unlock()
		return
	}
	conns, ok := s.markClosedLocked()
	s.mu.Unlock()
	if ok {
		s.finishClosed(conns)
	}
}

func (s *Share) close() {
	s.mu.Lock()
	conns, ok := s.markClosedLocked()
	s.mu.Unlock()
	if ok {
		s.finishClosed(conns)
	}
}

func (s *Share) accept() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		if !s.track(conn) {
			conn.Close()
			continue
		}
		if s.info.Kind == "tcp" {
			go s.serveTCP(conn)
		} else {
			go s.serveSSH(conn)
		}
	}
}
