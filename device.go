package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"io"
	"net"
	"os"
	"rttys/client"
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
	msgTypeAck
	msgTypeMax = msgTypeAck
)

const (
	msgTypeFileSend = iota
	msgTypeFileRecv
	msgTypeFileInfo
	msgTypeFileData
	msgTypeFileAck
	msgTypeFileAbort
)

const (
	msgRegAttrHeartbeat = iota
	msgRegAttrDevid
	msgRegAttrDescription
	msgRegAttrToken
)

const (
	msgHeartbeatAttrUptime = iota
)

const (
	devRegErrUnsupportedProto = iota + 1
	devRegErrInvalidToken
	devRegErrHookFailed
	devRegErrIdConflicting
)

var DevRegErrMsg = map[byte]string{
	0:                         "Success",
	devRegErrUnsupportedProto: "Unsupported protocol",
	devRegErrInvalidToken:     "Invalid token",
	devRegErrHookFailed:       "Hook failed",
	devRegErrIdConflicting:    "ID conflict",
}

// Minimum protocol version requirements of rtty
const rttyProtoRequired uint8 = 3

type device struct {
	br         *broker
	proto      uint8
	heartbeat  time.Duration
	id         string
	desc       string /* description of the device */
	timestamp  int64  /* Connection time */
	uptime     uint32
	token      string
	conn       net.Conn
	registered bool
	closed     uint32
	err        byte
	send       chan []byte // Buffered channel of outbound messages.
}

type termMessage struct {
	sid  string
	data []byte
}

type fileMessage struct {
	sid  string
	data []byte
}

type fileProxy struct {
	reader *io.PipeReader
	writer *io.PipeWriter
}

func (fp *fileProxy) Read(b []byte) (int, error) {
	return fp.reader.Read(b)
}

func (fp *fileProxy) Write(dev client.Client, sid string, b []byte) {
	go func() {
		_, err := fp.writer.Write(b)
		if err != nil {
			fp.Cancel(dev, sid)
			dev.(*device).br.fileProxy.Delete(sid)
			return
		}
		fp.Ack(dev, sid)
	}()
}

func (fp *fileProxy) Close() {
	fp.writer.Close()
}

func (fp *fileProxy) Cancel(dev client.Client, sid string) {
	b := make([]byte, 33)
	copy(b, sid)
	b[32] = msgTypeFileAbort
	dev.WriteMsg(msgTypeFile, b)
}

func (fp *fileProxy) Ack(dev client.Client, sid string) {
	b := make([]byte, 33)
	copy(b, sid)
	b[32] = msgTypeFileAck
	dev.WriteMsg(msgTypeFile, b)
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

func (dev *device) Closed() bool {
	return atomic.LoadUint32(&dev.closed) == 1
}

func (dev *device) CloseConn() {
	dev.conn.Close()
}

func (dev *device) Close() {
	if dev.Closed() {
		return
	}

	atomic.StoreUint32(&dev.closed, 1)

	log.Debug().Msgf("Device '%s' disconnected", dev.conn.RemoteAddr())

	dev.CloseConn()

	close(dev.send)
}

func parseDeviceInfo(dev *device, b []byte) bool {
	if len(b) < 1 {
		return false
	}

	dev.proto = b[0]

	if dev.proto > 4 {
		attrs := parseTLV(b[1:])
		if attrs == nil {
			return false
		}

		for typ, val := range attrs {
			switch typ {
			case msgRegAttrHeartbeat:
				dev.heartbeat = time.Duration(val[0]) * time.Second
			case msgRegAttrDevid:
				dev.id = string(val)
			case msgRegAttrDescription:
				dev.desc = string(val)
			case msgRegAttrToken:
				dev.token = string(val)
			}
		}

		return true
	}

	b = b[1:]

	fields := bytes.Split(b, []byte{0})

	if len(fields) < 3 {
		return false
	}

	dev.id = string(fields[0])
	dev.desc = string(fields[1])
	dev.token = string(fields[2])

	return true
}

func parseHeartbeat(dev *device, b []byte) bool {
	if dev.proto > 4 {
		attrs := parseTLV(b)
		if attrs == nil {
			return false
		}

		for typ, val := range attrs {
			switch typ {
			case msgHeartbeatAttrUptime:
				dev.uptime = binary.BigEndian.Uint32(val)
			}
		}
	} else {
		if len(b) < 4 {
			return false
		}
		dev.uptime = binary.BigEndian.Uint32(b[:4])
	}

	return true
}

func msgTypeName(typ byte) string {
	switch typ {
	case msgTypeRegister:
		return "register"
	case msgTypeLogin:
		return "login"
	case msgTypeLogout:
		return "logout"
	case msgTypeTermData:
		return "termdata"
	case msgTypeWinsize:
		return "winsize"
	case msgTypeCmd:
		return "cmd"
	case msgTypeHeartbeat:
		return "heartbeat"
	case msgTypeFile:
		return "file"
	case msgTypeHttp:
		return "http"
	case msgTypeAck:
		return "ack"
	default:
		return "unknown"
	}
}

func (dev *device) readLoop() {
	logPrefix := dev.conn.RemoteAddr().String()

	tmr := time.AfterFunc(time.Second*5, func() {
		log.Error().Msgf("%s: timeout", logPrefix)
		dev.Close()
	})

	defer func() {
		dev.br.unregister <- dev
		tmr.Stop()
	}()

	br := bufio.NewReader(dev.conn)

	for {
		b, err := br.Peek(3)
		if err != nil {
			if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
				log.Error().Msgf("%s: %s", logPrefix, err.Error())
			}
			return
		}

		br.Discard(3)

		typ := b[0]

		if typ > msgTypeMax {
			log.Error().Msgf("%s: invalid msg type: %d", logPrefix, typ)
			return
		}

		log.Debug().Msgf("%s: recv msg: %s", logPrefix, msgTypeName(typ))

		msgLen := binary.BigEndian.Uint16(b[1:])

		b = make([]byte, msgLen)
		_, err = io.ReadFull(br, b)
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}

		switch typ {
		case msgTypeRegister:
			if !parseDeviceInfo(dev, b) {
				log.Error().Msgf("%s: msgTypeRegister: invalid", logPrefix)
				return
			}

			if dev.id == "" {
				log.Error().Msgf("%s: msgTypeRegister: devid is empty", logPrefix)
				return
			}

			logPrefix = dev.id

			dev.br.devRegister(dev)

		case msgTypeLogin:
			if msgLen < 33 {
				log.Error().Msgf("%s: msgTypeLogin: invalid", logPrefix)
				return
			}

			sid := string(b[:32])
			code := b[32]

			dev.br.loginAck <- &loginAckMsg{dev.id, sid, code == 1}

		case msgTypeLogout:
			if msgLen < 32 {
				log.Error().Msgf("%s: msgTypeLogout: invalid", logPrefix)
				return
			}

			dev.br.logout <- string(b[:32])

		case msgTypeTermData:
			fallthrough
		case msgTypeFile:
			if msgLen < 32 {
				log.Error().Msgf("%s: msgTypeTermData|msgTypeFile: invalid", logPrefix)
				return
			}

			sid := string(b[:32])

			if typ == msgTypeFile {
				dev.br.fileMessage <- &fileMessage{sid, b[32:]}
			} else {
				dev.br.termMessage <- &termMessage{sid, b[32:]}
			}

		case msgTypeCmd:
			if msgLen < 1 {
				log.Error().Msgf("%s: msgTypeCmd: invalid", logPrefix)
				return
			}

			dev.br.cmdResp <- b

		case msgTypeHttp:
			if msgLen < 18 {
				log.Error().Msgf("%s: msgTypeHttp: invalid", logPrefix)
				return
			}

			dev.br.httpResp <- &httpResp{b, dev}

		case msgTypeHeartbeat:
			if !parseHeartbeat(dev, b) {
				log.Error().Msgf("%s: msgTypeHeartbeat: invalid", logPrefix)
				return
			}
			dev.br.heartbeat <- dev.id
		default:
			log.Error().Msgf("%s: invalid msg type: %d", logPrefix, typ)
		}

		tmr.Reset(dev.heartbeat * 3 / 2)
	}
}

func (dev *device) writeLoop() {
	defer func() {
		dev.br.unregister <- dev
	}()

	for msg := range dev.send {
		_, err := dev.conn.Write(msg)
		if err != nil {
			log.Error().Msg(err.Error())
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
		tlsConfig.MinVersion = tls.VersionTLS12

		if cfg.SslCacert == "" {
			log.Warn().Msgf("mTLS not enabled")
		} else {
			caCert, err := os.ReadFile(cfg.SslCacert)
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

			dev := &device{
				br:        br,
				conn:      conn,
				heartbeat: time.Second * 5,
				timestamp: time.Now().Unix(),
				send:      make(chan []byte, 256),
			}

			go dev.readLoop()
			go dev.writeLoop()
		}
	}()
}
