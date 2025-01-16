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

// Minimum protocol version requirements of rtty
const rttyProtoRequired uint8 = 3
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

func (dev *device) Close() {
	if dev.Closed() {
		return
	}

	atomic.StoreUint32(&dev.closed, 1)

	log.Debug().Msgf("Device '%s' disconnected", dev.conn.RemoteAddr())

	dev.conn.Close()

	close(dev.send)
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
		log.Error().Msgf("%s: msgTypeRegister: invalid", dev.conn.RemoteAddr())
		return false
	}

	dev.id = string(fields[0])
	dev.desc = string(fields[1])
	dev.token = string(fields[2])

	return true
}

func parseHeartbeat(dev *device, b []byte) {
	dev.uptime = binary.BigEndian.Uint32(b[:4])
}

func (dev *device) readLoop() {
	defer func() {
		dev.br.unregister <- dev
	}()

	logPrefix := dev.conn.RemoteAddr().String()

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
			if msgLen < 2 {
				log.Error().Msgf("%s: msgTypeRegister: invalid", logPrefix)
				return
			}

			if !parseDeviceInfo(dev, b) {
				return
			}

			if dev.id == "" {
				log.Error().Msgf("%s: msgTypeRegister: devid is empty", logPrefix)
				return
			}

			logPrefix = dev.id

			dev.br.register <- dev

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
			parseHeartbeat(dev, b)

		default:
			log.Error().Msgf("%s: invalid msg type: %d", logPrefix, typ)
		}
	}
}

func (dev *device) writeLoop() {
	ticker := time.NewTicker(time.Second)

	defer func() {
		ticker.Stop()
		dev.br.unregister <- dev
	}()

	ninactive := 0
	lastHeartbeat := time.Now()

	for {
		select {
		case msg, ok := <-dev.send:
			if !ok {
				return
			}

			_, err := dev.conn.Write(msg)
			if err != nil {
				log.Error().Msg(err.Error())
				return
			}

		case <-ticker.C:
			now := time.Now()
			if now.Sub(dev.active) > heartbeatInterval*3/2 {
				if dev.id == "" {
					return
				}

				log.Error().Msgf("Inactive device in long time: %s", dev.id)
				if ninactive > 1 {
					log.Error().Msgf("Inactive 3 times, now kill it: %s", dev.id)
					return
				}
				ninactive = ninactive + 1
			}

			if now.Sub(lastHeartbeat) > heartbeatInterval-1 {
				lastHeartbeat = now
				if len(dev.send) < 1 {
					dev.WriteMsg(msgTypeHeartbeat, []byte{})
				}
			}
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
				active:    time.Now(),
				timestamp: time.Now().Unix(),
				send:      make(chan []byte, 256),
			}

			go dev.readLoop()
			go dev.writeLoop()
		}
	}()
}
