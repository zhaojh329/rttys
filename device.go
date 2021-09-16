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
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	msgTypeRegister = iota
	msgTypeLogin
	msgTypeLogout
	msgTypeTermData
	msgTypeWinsize
	msgTypeCmd
	msgTypeHeartbeat
	msgTypeFile
	msgTypeHttp
	msgTypeMax = msgTypeHttp
)

const rttyProto uint8 = 3
const heartbeatInterval = time.Second * 5

type device struct {
	br         *broker
	proto      uint8
	id         string
	desc       string /* description of the device */
	timestamp  int64  /* Connection time */
	uptime     uint32
	token      string
	conn       net.Conn
	active     time.Time
	registered bool
	closed     uint32
	cancel     context.CancelFunc
	send       chan []byte // Buffered channel of outbound messages.
}

type termMessage struct {
	sid  string
	data []byte
}

type loginAckMsg struct {
	devid  string
	sid    string
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

	dev.send <- append(b, data...)
}

func (dev *device) Close() {
	if atomic.LoadUint32(&dev.closed) == 1 {
		return
	}
	atomic.StoreUint32(&dev.closed, 1)

	log.Debug().Msgf("Device '%s' disconnected", dev.conn.RemoteAddr())

	dev.conn.Close()

	dev.cancel()

	if dev.registered {
		dev.br.unregister <- dev
	}
}

func (dev *device) UpdateDb() {
	db, err := instanceDB(dev.br.cfg.DB)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}
	defer db.Close()

	cnt := 0

	db.QueryRow("SELECT COUNT(*) FROM device WHERE id = ?", dev.id).Scan(&cnt)
	if cnt == 0 {
		_, err = db.Exec("INSERT INTO device values(?,?,?,?)", dev.id, dev.desc, time.Now(), "")
	} else {
		_, err = db.Exec("UPDATE device SET description = ?, online = ? WHERE id = ?", dev.desc, time.Now(), dev.id)
	}

	if err != nil {
		log.Error().Msg(err.Error())
	}
}

func parseDeviceInfo(dev *device, b []byte) bool {
	dev.proto = b[0]

	b = b[1:]

	fields := bytes.Split(b, []byte{0})

	if len(fields) < 3 {
		log.Error().Msg("msgTypeRegister: invalid")
		return false
	}

	dev.id = string(fields[0])
	dev.desc = string(fields[1])
	dev.token = string(fields[2])

	return true
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
				if !dev.registered {
					dev.Close()
					return
				}

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

		if typ > msgTypeMax {
			log.Error().Msgf("invalid msg type: %d", typ)
			return
		}

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
			if !parseDeviceInfo(dev, b) {
				return
			}

			dev.br.register <- dev

		case msgTypeLogin:
			if msgLen < 33 {
				log.Error().Msg("msgTypeLogin: invalid")
				return
			}

			sid := string(b[:32])
			code := b[32]

			dev.br.loginAck <- &loginAckMsg{dev.id, sid, code == 1}

		case msgTypeLogout:
			if msgLen < 32 {
				log.Error().Msg("msgTypeLogout: invalid")
				return
			}

			dev.br.logout <- string(b[:32])

		case msgTypeTermData:
			fallthrough
		case msgTypeFile:
			if msgLen < 32 {
				log.Error().Msg("msgTypeTermData|msgTypeFile: invalid")
				return
			}

			sid := string(b[:32])

			b = b[31:]

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

		case msgTypeHttp:
			if msgLen < 18 {
				log.Error().Msg("msgTypeHttp: invalid")
				return
			}

			dev.br.httpMessage <- &httpResp{b, dev}

		case msgTypeHeartbeat:
			parseHeartbeat(dev, b)

		default:
			log.Error().Msgf("invalid msg type: %d", typ)
		}
	}
}

func (dev *device) writeLoop(ctx context.Context) {
	defer dev.Close()

	for {
		select {
		case msg := <-dev.send:
			_, err := dev.conn.Write(msg)
			if err != nil {
				log.Error().Msg(err.Error())
				return
			}
		case <-ctx.Done():
			return
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

		if cfg.SslCacert == "" {
			log.Warn().Msgf("mTLS not enabled")
		} else {
			caCert, err := ioutil.ReadFile(cfg.SslCacert)
			if err != nil {
				log.Error().Msgf("mTLS not enabled: %s", err.Error())
			} else {
				br.devCertPool = x509.NewCertPool()
				br.devCertPool.AppendCertsFromPEM(caCert)

				// Create the TLS Config with the CA pool and enable Client certificate validation
				tlsConfig.ClientCAs = br.devCertPool
				tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
			}
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

			log.Debug().Msgf("Device '%s' connected", conn.RemoteAddr())

			ctx, cancel := context.WithCancel(context.Background())

			dev := &device{
				br:        br,
				conn:      conn,
				cancel:    cancel,
				active:    time.Now(),
				timestamp: time.Now().Unix(),
				send:      make(chan []byte, 256),
			}

			go dev.readLoop()
			go dev.writeLoop(ctx)
			go dev.keepAlive(ctx)
		}
	}()
}
