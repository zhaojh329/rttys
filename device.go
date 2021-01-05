package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	msgTypeRegister  = 0x00
	msgTypeLogin     = 0x01
	msgTypeLogout    = 0x02
	msgTypeTermData  = 0x03
	msgTypeWinsize   = 0x04
	msgTypeCmd       = 0x05
	msgTypeHeartbeat = 0x06
	msgTypeFile      = 0x07
)

const heartbeatInterval = time.Second * 5

type device struct {
	br         *broker
	id         string
	desc       string /* description of the device */
	timestamp  int64  /* Connection time */
	token      string
	conn       net.Conn
	loginMutex sync.Mutex
	u          *user /* User who is wait login */
	active     time.Time
	closeMutex sync.Mutex
	closed     bool
	closeCh    chan struct{}
}

type devMessage struct {
	devid     string
	sid       uint8
	data      []byte
	isFileMsg bool
}

func (dev *device) login(u *user) bool {
	defer dev.loginMutex.Unlock()

	dev.loginMutex.Lock()

	if dev.u != nil {
		return false
	}

	dev.u = u
	dev.writeMsg(msgTypeLogin, []byte{})
	return true
}

func (dev *device) handleLogin(code byte, sid byte) {
	defer dev.loginMutex.Unlock()

	dev.loginMutex.Lock()

	if dev.u == nil {
		return
	}

	u := dev.u
	dev.u = nil

	if code == 1 {
		log.Error().Msg("login fail, device busy")
		u.loginAck(loginErrorBusy)
		return
	}

	dev.br.newSession <- &session{dev, u, sid}
}

func (dev *device) logout(sid byte) {
	dev.writeMsg(msgTypeLogout, []byte{sid})
}

func (dev *device) keepAlive() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	ninactive := 0

	lastHeartbeat := time.Now()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			if now.Sub(dev.active) > heartbeatInterval*3/2 {
				log.Error().Msgf("Inactive device in long time: %s", dev.id)
				if ninactive > 1 {
					log.Error().Msgf("Inactive 3 times, now kill it: %s", dev.id)
					dev.close()
					return
				}
				ninactive = ninactive + 1
			}

			if now.Sub(lastHeartbeat) > heartbeatInterval-1 {
				lastHeartbeat = now
				dev.writeMsg(msgTypeHeartbeat, []byte{})
			}
		case <-dev.closeCh:
			return
		}
	}
}

func (dev *device) close() {
	defer dev.closeMutex.Unlock()

	dev.closeMutex.Lock()

	if !dev.closed {
		dev.closed = true
		time.AfterFunc(time.Second, func() {
			dev.br.unregister <- dev
			close(dev.closeCh)
			dev.conn.Close()
			log.Info().Msgf("Device '%s' closed", dev.id)
		})
	}
}

func (dev *device) writeMsg(typ byte, data []byte) {
	b := []byte{typ}
	b = append(b, intToBytes(len(data), 2)...)
	b = append(b, data...)
	dev.conn.Write(b)
}

func parseDeviceInfo(b []byte) (string, string, string) {
	fields := bytes.Split(b, []byte{0})

	id := string(fields[0])
	desc := string(fields[1])
	token := string(fields[2])

	return id, desc, token
}

func (dev *device) readLoop() {
	defer dev.close()

	br := bufio.NewReaderSize(dev.conn, 4096+100)

	for {
		b, err := br.Peek(3)
		if err != nil {
			if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
				log.Error().Msg(err.Error())
			}
			return
		}

		br.Discard(3)

		typ := b[0]
		msgLen := bytesToIntU(b[1:])

		b, err = br.Peek(msgLen)
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}

		br.Discard(msgLen)

		dev.active = time.Now()

		switch typ {
		case msgTypeRegister:
			id, desc, token := parseDeviceInfo(b)
			dev.id = id
			dev.token = token
			dev.desc = desc
			dev.br.register <- dev

		case msgTypeLogin:
			code := b[0]
			sid := byte(0)
			if code == 0 {
				sid = b[1]
			}
			dev.handleLogin(code, sid)

		case msgTypeLogout:
			sid := b[0]
			dev.br.logout <- dev.id + string(sid+'0')

		case msgTypeTermData:
			fallthrough
		case msgTypeFile:
			sid := b[0]
			data := make([]byte, len(b[1:]))
			copy(data, b[1:])
			dev.br.devMessage <- &devMessage{dev.id, sid, data, typ == msgTypeFile}

		case msgTypeCmd:
			data := make([]byte, len(b))
			copy(data, b)
			dev.br.cmdMessage <- b

		case msgTypeHeartbeat:

		default:
			log.Error().Msg("invalid msg type")
		}
	}
}

func listenDevice(br *broker, cfg *rttysConfig) {
	ln, err := net.Listen("tcp", cfg.addrDev)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer ln.Close()

	if cfg.sslCert != "" && cfg.sslKey != "" {
		crt, err := tls.LoadX509KeyPair(cfg.sslCert, cfg.sslKey)
		if err != nil {
			log.Fatal().Msg(err.Error())
		}

		tlsConfig := &tls.Config{}
		tlsConfig.Certificates = []tls.Certificate{crt}
		tlsConfig.Time = time.Now
		tlsConfig.Rand = rand.Reader

		ln = tls.NewListener(ln, tlsConfig)
		log.Info().Msgf("Listen device on: %s SSL on", cfg.addrDev)
	} else {
		log.Info().Msgf("Listen device on: %s SSL off", cfg.addrDev)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Error().Msg(err.Error())
			continue
		}

		dev := &device{
			br:        br,
			conn:      conn,
			closeCh:   make(chan struct{}),
			active:    time.Now(),
			timestamp: time.Now().Unix(),
		}

		go dev.readLoop()
		go dev.keepAlive()
	}
}
