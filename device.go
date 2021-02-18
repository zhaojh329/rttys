package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"io"
	"io/ioutil"
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
	msgTypeWeb       = 0x08
)

const heartbeatInterval = time.Second * 5

type device struct {
	br         *broker
	id         string
	desc       string /* description of the device */
	timestamp  int64  /* Connection time */
	uptime     uint32
	token      string
	conn       net.Conn
	active     time.Time
	closeMutex sync.Mutex
	closed     bool
	cancel     context.CancelFunc
}

type termMessage struct {
	sid  string
	data []byte
}

type loginAckMsg struct {
	devid  string
	sid    byte
	isBusy bool
}

func (dev *device) IsDevice() bool {
	return true
}

func (dev *device) DeviceID() string {
	return dev.id
}

func (dev *device) WriteMsg(typ int, data []byte) {
	b := []byte{byte(typ), 0, 0}

	binary.BigEndian.PutUint16(b[1:], uint16(len(data)))

	dev.conn.Write(append(b, data...))
}

func (dev *device) Close() {
	defer dev.closeMutex.Unlock()

	dev.closeMutex.Lock()

	if !dev.closed {
		dev.closed = true

		dev.conn.Close()

		dev.cancel()

		dev.br.unregister <- dev

		log.Info().Msgf("Device '%s' closed", dev.id)
	}
}

func parseDeviceInfo(b []byte) (string, string, string, bool) {
	fields := bytes.Split(b, []byte{0})

	if len(fields) < 3 {
		return "", "", "", false
	}

	id := string(fields[0])
	desc := string(fields[1])
	token := string(fields[2])

	return id, desc, token, true
}

func parseHeartbeat(dev *device, b []byte) {
	// Old rtty not support this
	if len(b) < 4 {
		return
	}
	dev.uptime = binary.BigEndian.Uint32(b[:4])
}

func (dev *device) keepAlive(ctx context.Context) {
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
					dev.Close()
					return
				}
				ninactive = ninactive + 1
			}

			if now.Sub(lastHeartbeat) > heartbeatInterval-1 {
				lastHeartbeat = now
				dev.WriteMsg(msgTypeHeartbeat, []byte{})
			}
		case <-ctx.Done():
			return
		}
	}
}

func (dev *device) readLoop() {
	defer dev.Close()

	br := bufio.NewReader(dev.conn)

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
		msgLen := binary.BigEndian.Uint16(b[1:])

		b = make([]byte, msgLen)
		_, err = io.ReadFull(br, b)
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}

		dev.active = time.Now()

		switch typ {
		case msgTypeRegister:
			id, desc, token, ok := parseDeviceInfo(b)
			if !ok {
				log.Error().Msg("msgTypeRegister: invalid")
				return
			}

			dev.id = id
			dev.token = token
			dev.desc = desc
			dev.br.register <- dev

		case msgTypeLogin:
			if msgLen < 1 {
				log.Error().Msg("msgTypeLogin: invalid")
				return
			}

			code := b[0]
			sid := byte(0)

			if code == 0 {
				if msgLen < 2 {
					log.Error().Msg("msgTypeLogin: invalid")
					return
				}

				sid = b[1]
			}
			dev.br.loginAck <- &loginAckMsg{dev.id, sid, code == 1}

		case msgTypeLogout:
			if msgLen < 1 {
				log.Error().Msg("msgTypeLogout: invalid")
				return
			}

			dev.br.logout <- dev.id + string(b[0]+'0')

		case msgTypeTermData:
			fallthrough
		case msgTypeFile:
			if msgLen < 1 {
				log.Error().Msg("msgTypeTermData|msgTypeFile: invalid")
				return
			}

			sid := dev.id + string(b[0]+'0')

			if typ == msgTypeFile {
				b[0] = 1
			} else {
				b[0] = 0
			}

			dev.br.termMessage <- &termMessage{sid, b}

		case msgTypeCmd:
			if msgLen < 1 {
				log.Error().Msg("msgTypeCmd: invalid")
				return
			}

			dev.br.cmdMessage <- b

		case msgTypeWeb:
			if msgLen < 18 {
				log.Error().Msg("msgTypeWeb: invalid")
				return
			}

			dev.br.webMessage <- &webResp{b, dev}

		case msgTypeHeartbeat:
			parseHeartbeat(dev, b)

		default:
			log.Error().Msgf("invalid msg type: %d", typ)
		}
	}
}

func listenDevice(br *broker) {
	cfg := br.cfg

	ln, err := net.Listen("tcp", cfg.AddrDev)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	if cfg.SslCert != "" && cfg.SslKey != "" {
		crt, err := tls.LoadX509KeyPair(cfg.SslCert, cfg.SslKey)
		if err != nil {
			log.Fatal().Msg(err.Error())
		}

		tlsConfig := &tls.Config{}
		tlsConfig.Certificates = []tls.Certificate{crt}
		tlsConfig.Time = time.Now
		tlsConfig.Rand = rand.Reader

		caCert, err := ioutil.ReadFile(cfg.SslDevices)
		if err != nil {
			log.Warn().Msgf("mTLS not used: %s", err.Error())
		} else {
			br.devCertPool = x509.NewCertPool()
			br.devCertPool.AppendCertsFromPEM(caCert)

			// Create the TLS Config with the CA pool and enable Client certificate validation
			tlsConfig.ClientCAs = br.devCertPool
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}

		ln = tls.NewListener(ln, tlsConfig)
		log.Info().Msgf("Listen device on: %s SSL on", cfg.AddrDev)
	} else {
		log.Info().Msgf("Listen device on: %s SSL off", cfg.AddrDev)
	}

	go func() {
		defer ln.Close()

		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Error().Msg(err.Error())
				continue
			}

			ctx, cancel := context.WithCancel(context.Background())

			dev := &device{
				br:        br,
				conn:      conn,
				cancel:    cancel,
				active:    time.Now(),
				timestamp: time.Now().Unix(),
			}

			go dev.readLoop()
			go dev.keepAlive(ctx)
		}
	}()
}
