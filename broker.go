package main

import (
	"crypto/x509"
	"encoding/binary"
	"sync/atomic"
	"time"

	"rttys/client"
	"rttys/config"
	"rttys/utils"

	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
)

type session struct {
	dev       client.Client
	user      client.Client
	confirmed uint32
}

type broker struct {
	cfg         *config.Config
	devices     map[string]client.Client
	loginAck    chan *loginAckMsg
	logout      chan string
	register    chan client.Client
	unregister  chan client.Client
	sessions    map[string]*session
	termMessage chan *termMessage
	userMessage chan *usrMessage
	cmdMessage  chan []byte
	httpMessage chan *httpResp
	devCertPool *x509.CertPool
}

func newBroker(cfg *config.Config) *broker {
	return &broker{
		cfg:         cfg,
		loginAck:    make(chan *loginAckMsg, 1000),
		logout:      make(chan string, 1000),
		register:    make(chan client.Client, 1000),
		unregister:  make(chan client.Client, 1000),
		devices:     make(map[string]client.Client),
		sessions:    make(map[string]*session),
		termMessage: make(chan *termMessage, 1000),
		userMessage: make(chan *usrMessage, 1000),
		cmdMessage:  make(chan []byte, 1000),
		httpMessage: make(chan *httpResp, 1000),
	}
}

func (br *broker) run() {
	for {
		select {
		case c := <-br.register:
			devid := c.DeviceID()

			if c.IsDevice() {
				dev := c.(*device)
				err := byte(0)
				msg := "OK"

				if _, ok := br.devices[devid]; ok {
					log.Error().Msg("Device ID conflicting: " + devid)
					msg = "ID conflicting"
					err = 1
				} else if br.cfg.Token != "" && dev.token != br.cfg.Token {
					log.Error().Msg("Invalid token from terminal device")
					msg = "Invalid token"
					err = 1
				} else if dev.proto < rttyProto {
					if dev.proto < rttyProto {
						log.Error().Msgf("%s: unsupported protocol version: %d, need %d", dev.id, dev.proto, rttyProto)
						msg = "unsupported protocol"
						err = 1
					}
				} else {
					dev.registered = true
					br.devices[devid] = c
					dev.UpdateDb()
					log.Info().Msgf("Device '%s' registered, proto %d", devid, dev.proto)
				}

				c.WriteMsg(msgTypeRegister, append([]byte{err}, msg...))
			} else {
				if dev, ok := br.devices[devid]; ok {
					sid := utils.GenUniqueID("sid")

					s := &session{
						dev:  dev,
						user: c,
					}

					time.AfterFunc(time.Second*3, func() {
						if atomic.LoadUint32(&s.confirmed) == 0 {
							c.Close()
						}
					})

					br.sessions[sid] = s

					dev.WriteMsg(msgTypeLogin, []byte(sid))
					log.Info().Msg("New session: " + sid)
				} else {
					userLoginAck(loginErrorOffline, c)
					log.Error().Msgf("Not found the device '%s'", devid)
				}
			}

		case c := <-br.unregister:
			devid := c.DeviceID()

			if c.IsDevice() {
				delete(br.devices, devid)

				for sid, s := range br.sessions {
					if s.dev == c {
						s.user.Close()
						delete(br.sessions, sid)
						log.Info().Msg("Delete session: " + sid)
					}
				}

				log.Info().Msgf("Device '%s' unregistered", devid)
			} else {
				sid := c.(*user).sid

				if _, ok := br.sessions[sid]; ok {
					delete(br.sessions, sid)
					c.Close()

					if dev, ok := br.devices[devid]; ok {
						dev.WriteMsg(msgTypeLogout, []byte(sid))
					}

					log.Info().Msg("Delete session: " + sid)
				}
			}

		case msg := <-br.loginAck:
			if s, ok := br.sessions[msg.sid]; ok {
				if msg.isBusy {
					userLoginAck(loginErrorBusy, s.user)
					log.Error().Msg("login fail, device busy")
				} else {
					atomic.StoreUint32(&s.confirmed, 1)

					u := s.user.(*user)
					u.sid = msg.sid

					userLoginAck(loginErrorNone, s.user)
				}
			}

		// device active logout
		// typically, executing the exit command at the terminal will case this
		case sid := <-br.logout:
			if s, ok := br.sessions[sid]; ok {
				delete(br.sessions, sid)
				s.user.Close()

				log.Info().Msg("Delete session: " + sid)
			}

		// from device, includes terminal data and file data
		case msg := <-br.termMessage:
			if s, ok := br.sessions[msg.sid]; ok {
				s.user.WriteMsg(websocket.BinaryMessage, msg.data)
			}

		case msg := <-br.userMessage:
			if s, ok := br.sessions[msg.sid]; ok {
				if dev, ok := br.devices[s.dev.DeviceID()]; ok {
					data := msg.data

					if msg.typ == websocket.BinaryMessage {
						typ := msgTypeTermData
						if data[0] == 1 {
							typ = msgTypeFile
						}
						dev.WriteMsg(typ, append([]byte(msg.sid), data[1:]...))
					} else {
						typ := jsoniter.Get(data, "type").ToString()

						switch typ {
						case "winsize":
							b := [32 + 4]byte{}

							copy(b[:], msg.sid)

							cols := jsoniter.Get(data, "cols").ToUint()
							rows := jsoniter.Get(data, "rows").ToUint()

							binary.BigEndian.PutUint16(b[32:], uint16(cols))
							binary.BigEndian.PutUint16(b[34:], uint16(rows))

							dev.WriteMsg(msgTypeWinsize, b[:])

						case "ack":
							b := [32 + 2]byte{}
							copy(b[:], msg.sid)

							ack := jsoniter.Get(data, "ack").ToUint()
							binary.BigEndian.PutUint16(b[32:], uint16(ack))
							dev.WriteMsg(msgTypeAck, b[:])
						}
					}
				}
			} else {
				log.Error().Msg("Not found sid: " + msg.sid)
			}

		case data := <-br.cmdMessage:
			handleCmdResp(data)

		case resp := <-br.httpMessage:
			handleHttpProxyResp(resp)
		}
	}
}
