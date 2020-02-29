package main

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	MsgTypeRegister  = 0x00
	MsgTypeLogin     = 0x01
	MsgTypeLogout    = 0x02
	MsgTypeTermData  = 0x03
	MsgTypeWinsize   = 0x04
	MsgTypeCmd       = 0x05
	MsgTypeHeartbeat = 0x06
	MsgTypeFile      = 0x07
)

const HeartbeatInterval = time.Second * 5

type Device struct {
	br         *Broker
	id         string
	desc       string /* description of the device */
	timestamp  int64  /* Connection time */
	token      string
	conn       net.Conn
	loginMutex sync.Mutex
	user       *User /* User who is wait login */
	active     time.Time
	closeMutex sync.Mutex
	closed     bool
	closeCh    chan struct{}
}

type DevMessage struct {
	devid     string
	sid       uint8
	data      []byte
	isFileMsg bool
}

func (dev *Device) login(user *User) bool {
	defer dev.loginMutex.Unlock()

	dev.loginMutex.Lock()

	if dev.user != nil {
		return false
	}

	dev.user = user
	dev.writeMsg(MsgTypeLogin, []byte{})
	return true
}

func (dev *Device) handleLogin(code byte, sid byte) {
	defer dev.loginMutex.Unlock()

	dev.loginMutex.Lock()

	if dev.user == nil {
		return
	}

	user := dev.user
	dev.user = nil

	if code == 1 {
		log.Error().Msg("login fail, device busy")
		user.loginAck(LoginErrorBusy)
		return
	}

	dev.br.newSession <- &Session{dev, user, sid}
}

func (dev *Device) logout(sid byte) {
	dev.writeMsg(MsgTypeLogout, []byte{sid})
}

func (dev *Device) keepAlive() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if time.Now().Sub(dev.active) > HeartbeatInterval*3/2 {
				log.Error().Msgf("Inactive device in long time, now kill it: %s", dev.id)
				dev.close()
				return
			}
		case <-dev.closeCh:
			return
		}
	}
}

func (dev *Device) close() {
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

func (dev *Device) writeMsg(typ byte, data []byte) {
	b := []byte{typ}
	b = append(b, intToBytes(len(data), 2)...)
	b = append(b, data...)
	dev.conn.Write(b)
}

func parseDeviceInfo(br *bufio.Reader) (string, string, string) {
	id, _ := br.ReadString(0)
	desc, _ := br.ReadString(0)
	token, _ := br.ReadString(0)

	id = strings.Trim(id, "\000")
	desc = strings.Trim(desc, "\000")
	token = strings.Trim(token, "\000")

	return id, desc, token
}

func (dev *Device) readLoop() {
	defer dev.close()

	br := bufio.NewReaderSize(dev.conn, 4096+100)

	msgLen := -1

	for {
		if msgLen < 0 {
			b, err := br.Peek(3)
			if err != nil {
				if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
					log.Error().Msg(err.Error())
				}
				return
			}
			msgLen = bytesToIntU(b[1:])
		}

		_, err := br.Peek(msgLen + 3)
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}

		typ, _ := br.ReadByte()
		br.Discard(2)

		dev.active = time.Now()

		switch typ {
		case MsgTypeRegister:
			id, desc, token := parseDeviceInfo(br)
			dev.id = id
			dev.token = token
			dev.desc = desc
			dev.br.register <- dev

		case MsgTypeLogin:
			code, _ := br.ReadByte()
			sid := byte(0)
			if code == 0 {
				sid, _ = br.ReadByte()
			}
			dev.handleLogin(code, sid)

		case MsgTypeLogout:
			sid, _ := br.ReadByte()
			dev.br.logout <- dev.id + string(sid+'0')

		case MsgTypeTermData:
			fallthrough
		case MsgTypeFile:
			sid, _ := br.ReadByte()
			data := make([]byte, msgLen-1)
			br.Read(data)
			dev.br.devMessage <- &DevMessage{dev.id, sid, data, typ == MsgTypeFile}

		case MsgTypeCmd:
			data := make([]byte, msgLen)
			br.Read(data)
			dev.br.cmdMessage <- data

		case MsgTypeHeartbeat:
			dev.writeMsg(MsgTypeHeartbeat, []byte{})

		default:
			log.Error().Msg("invalid msg type")
			br.Discard(msgLen)
		}

		msgLen = -1
	}
}

func listenDevice(br *Broker, cfg *RttysConfig) {
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

		dev := &Device{
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
