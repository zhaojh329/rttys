/* SPDX-License-Identifier: MIT */
package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/zhaojh329/rtty-go/proto"
	"github.com/zhaojh329/rttys/v5/internal/utils"
	"golang.org/x/crypto/ssh"
)

func (s *Share) serveSSH(raw net.Conn) {
	defer s.release(raw)
	raw.SetDeadline(time.Now().Add(15 * time.Second))
	config := &ssh.ServerConfig{
		MaxAuthTries: 3,
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			hash := sha256.Sum256(password)
			if conn.User() != "share" || subtle.ConstantTimeCompare(hash[:], s.secret[:]) != 1 {
				return nil, fmt.Errorf("invalid credentials")
			}
			return nil, nil
		},
	}
	config.AddHostKey(s.signer)
	serverConn, channels, requests, err := ssh.NewServerConn(raw, config)
	if err != nil {
		return
	}
	defer serverConn.Close()
	raw.SetDeadline(time.Time{})
	go ssh.DiscardRequests(requests)
	var once sync.Once
	for ch := range channels {
		if ch.ChannelType() != "session" {
			ch.Reject(ssh.Prohibited, "only interactive sessions are supported")
			continue
		}
		accepted := false
		once.Do(func() { accepted = true })
		if !accepted {
			ch.Reject(ssh.Prohibited, "only one session is supported")
			continue
		}
		channel, reqs, err := ch.Accept()
		if err != nil {
			return
		}
		go s.serveSSHChannel(raw, channel, reqs)
	}
}

func (s *Share) serveSSHChannel(raw net.Conn, ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer ch.Close()
	defer s.deactivate(raw)
	var peer *sshPeer
	var cols, rows uint32
	for req := range reqs {
		switch req.Type {
		case "pty-req":
			var pty struct {
				Term                      string
				Cols, Rows, Width, Height uint32
				Modes                     string
			}
			if ssh.Unmarshal(req.Payload, &pty) == nil {
				cols, rows = pty.Cols, pty.Rows
				req.Reply(true, nil)
			} else {
				req.Reply(false, nil)
			}
		case "window-change":
			var size struct{ Cols, Rows, Width, Height uint32 }
			if ssh.Unmarshal(req.Payload, &size) == nil && peer != nil {
				peer.winsize(size.Cols, size.Rows)
			}
		case "shell":
			if peer != nil || len(req.Payload) != 0 {
				req.Reply(false, nil)
				continue
			}
			if !s.activate(raw) {
				req.Reply(false, nil)
				return
			}
			peer = &sshPeer{share: s, ch: ch, dev: s.dev, sid: utils.GenUniqueID(),
				pending: make(chan byte, 1), output: make(chan []byte, 64), done: make(chan struct{})}
			req.Reply(true, nil)
			go peer.run(cols, rows)
		default:
			req.Reply(false, nil)
		}
	}
	if peer != nil {
		peer.Close()
	}
}

type sshPeer struct {
	share   *Share
	ch      ssh.Channel
	dev     *Device
	sid     string
	pending chan byte
	output  chan []byte
	once    sync.Once
	done    chan struct{}
	mu      sync.Mutex
	closed  bool
}

func (p *sshPeer) OnOpen(code byte) {
	select {
	case <-p.done:
		if _, loaded := p.dev.users.LoadAndDelete(p.sid); loaded {
			p.dev.WriteMsg(proto.MsgTypeLogout, p.sid)
		}
		return
	default:
	}

	select {
	case p.pending <- code:
	default:
	}
}

func (p *sshPeer) OnData(data []byte) {
	select {
	case p.output <- append([]byte(nil), data...):
	case <-p.share.done:
	case <-p.done:
	default:
		p.Close()
	}
}

func (p *sshPeer) OnFile(_ []byte) {
	p.dev.WriteMsg(proto.MsgTypeFile, p.sid, proto.MsgTypeFileAbort)
}

func (p *sshPeer) Close() {
	p.once.Do(func() {
		p.mu.Lock()
		p.closed = true
		close(p.done)
		_, loaded := p.dev.users.LoadAndDelete(p.sid)
		p.dev.pending.Delete(p.sid)
		p.mu.Unlock()
		if loaded {
			p.dev.WriteMsg(proto.MsgTypeLogout, p.sid)
		}
		p.ch.Close()
	})
}

func (p *sshPeer) winsize(cols, rows uint32) {
	if p.share.info.Kind == "terminal" && cols > 0 && rows > 0 {
		p.dev.WriteMsg(proto.MsgTypeWinsize, p.sid, uint16(min(cols, 65535)), uint16(min(rows, 65535)))
	}
}

func (p *sshPeer) run(cols, rows uint32) {
	defer p.Close()
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.dev.pending.Store(p.sid, p)
	p.mu.Unlock()
	var err error
	if p.share.info.Kind == "serial" {
		var payload []byte
		payload, err = p.share.info.Serial.MarshalBinary()
		if err == nil {
			err = p.dev.WriteMsg(proto.MsgTypeSerialOpen, p.sid, payload)
		}
	} else {
		err = p.dev.WriteMsg(proto.MsgTypeLogin, p.sid)
	}
	if err != nil {
		return
	}
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	select {
	case code := <-p.pending:
		if code != 0 {
			io.WriteString(p.ch.Stderr(), "Device session unavailable\r\n")
			return
		}
	case <-timer.C:
		io.WriteString(p.ch.Stderr(), "Device session timed out\r\n")
		return
	case <-p.dev.ctx.Done():
		return
	case <-p.share.done:
		return
	case <-p.done:
		return
	}
	p.winsize(cols, rows)
	go p.writeOutput()
	buf := make([]byte, 32*1024)
	for {
		n, err := p.ch.Read(buf)
		if n > 0 && p.dev.WriteMsg(proto.MsgTypeTermData, p.sid, buf[:n]) != nil {
			return
		}
		if err != nil {
			return
		}
	}
}

func (p *sshPeer) writeOutput() {
	for {
		select {
		case <-p.share.done:
			p.Close()
			return
		case <-p.done:
			return
		case data := <-p.output:
			for len(data) > 0 {
				n, err := p.ch.Write(data)
				if n > 0 {
					p.dev.WriteMsg(proto.MsgTypeAck, p.sid, uint16(n))
					data = data[n:]
				}
				if err != nil || n == 0 {
					p.Close()
					return
				}
			}
		}
	}
}
