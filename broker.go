package main

import (
	"crypto/x509"
	"encoding/binary"
	"time"

	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
	"github.com/zhaojh329/rttys/client"
	"github.com/zhaojh329/rttys/config"
)

type session struct {
	devid  string
	devsid byte
	u      client.Client
}

type broker struct {
	cfg            *config.Config
	devices        map[string]client.Client
	loginAck       chan *loginAckMsg
	logout         chan string
	register       chan client.Client
	unregister     chan client.Client
	waitLoginUsers map[string]client.Client
	sessions       map[string]*session
	cmdReq         chan *commandReq
	webCon         chan *webNewCon
	webReq         chan *webReq
	termMessage    chan *termMessage
	userMessage    chan *usrMessage
	cmdMessage     chan []byte
	webMessage     chan *webResp
	devCertPool    *x509.CertPool
}

func newBroker(cfg *config.Config) *broker {
	return &broker{
		cfg:            cfg,
		loginAck:       make(chan *loginAckMsg, 1000),
		logout:         make(chan string, 1000),
		register:       make(chan client.Client, 1000),
		unregister:     make(chan client.Client, 1000),
		devices:        make(map[string]client.Client),
		waitLoginUsers: make(map[string]client.Client),
		sessions:       make(map[string]*session),
		cmdReq:         make(chan *commandReq, 1000),
		webCon:         make(chan *webNewCon, 1000),
		webReq:         make(chan *webReq, 1000),
		termMessage:    make(chan *termMessage, 1000),
		userMessage:    make(chan *usrMessage, 1000),
		cmdMessage:     make(chan []byte, 1000),
		webMessage:     make(chan *webResp, 1000),
	}
}

func (br *broker) run() {
	for {
		select {
		case c := <-br.register:
			devid := c.DeviceID()

			if c.IsDevice() {
				err := byte(0)
				msg := "OK"

				if _, ok := br.devices[devid]; ok {
					log.Error().Msg("Device ID conflicting: " + devid)
					msg = "ID conflicting"
					err = 1
				} else if br.cfg.Token != "" && c.(*device).token != br.cfg.Token {
					log.Error().Msg("Invalid token from terminal device")
					msg = "Invalid token"
					err = 1
				} else {
					br.devices[devid] = c
					log.Info().Msg("New device: " + devid)
				}

				c.WriteMsg(msgTypeRegister, append([]byte{err}, msg...))
			} else {
				if dev, ok := br.devices[devid]; ok {
					if _, ok := br.waitLoginUsers[devid]; ok {
						log.Error().Msg("Another user is logining the device, wait...")
						time.AfterFunc(time.Millisecond*10, func() {
							br.register <- c
						})
					} else {
						br.waitLoginUsers[devid] = c
						dev.WriteMsg(msgTypeLogin, []byte{})
					}
				} else {
					userLoginAck(loginErrorOffline, c)
					log.Error().Msgf("Not found the device '%s'", devid)
				}
			}

		case c := <-br.unregister:
			id := c.DeviceID()

			if c.IsDevice() {
				delete(br.devices, id)

				for sid, s := range br.sessions {
					if s.devid == id {
						s.u.Close()
						delete(br.sessions, sid)
						log.Info().Msg("Delete session: " + sid)
					}
				}
			} else {
				sid := c.(*user).sid

				if s, ok := br.sessions[sid]; ok {
					delete(br.sessions, sid)
					c.Close()

					if dev, ok := br.devices[s.devid]; ok {
						dev.WriteMsg(msgTypeLogout, []byte{sid[len(sid)-1] - '0'})
					}

					log.Info().Msg("Delete session: " + sid)
				}
			}

		case msg := <-br.loginAck:
			if c, ok := br.waitLoginUsers[msg.devid]; ok {
				if msg.isBusy {
					userLoginAck(loginErrorBusy, c)
					log.Error().Msg("login fail, device busy")
				} else {
					sid := msg.devid + string(msg.sid+'0')
					br.sessions[sid] = &session{msg.devid, msg.sid, c}

					u := c.(*user)
					u.sid = sid

					userLoginAck(loginErrorNone, c)

					log.Info().Msg("New session: " + sid)
				}
				delete(br.waitLoginUsers, msg.devid)
			}

		// device active logout
		// typically, executing the exit command at the terminal will case this
		case sid := <-br.logout:
			if s, ok := br.sessions[sid]; ok {
				delete(br.sessions, sid)
				s.u.Close()

				log.Info().Msg("Delete session: " + sid)
			}

		// from device, includes terminal data and file data
		case msg := <-br.termMessage:
			if s, ok := br.sessions[msg.sid]; ok {
				s.u.WriteMsg(websocket.BinaryMessage, msg.data)
			}

		case msg := <-br.userMessage:
			if s, ok := br.sessions[msg.sid]; ok {
				if dev, ok := br.devices[s.devid]; ok {
					devsid := msg.sid[len(msg.sid)-1] - '0'
					data := msg.data

					if msg.typ == websocket.BinaryMessage {
						if data[0] == 1 {
							dev.WriteMsg(msgTypeFile, data[1:])
						} else {
							dev.WriteMsg(msgTypeTermData, append([]byte{devsid}, data[1:]...))
						}
					} else {
						typ := jsoniter.Get(data, "type").ToString()

						switch typ {
						case "winsize":
							b := [5]byte{devsid}

							cols := jsoniter.Get(data, "cols").ToUint()
							rows := jsoniter.Get(data, "rows").ToUint()

							binary.BigEndian.PutUint16(b[1:], uint16(cols))
							binary.BigEndian.PutUint16(b[3:], uint16(rows))

							dev.WriteMsg(msgTypeWinsize, b[:])
						}
					}
				}
			} else {
				log.Error().Msg("Not found sid: " + msg.sid)
			}

		case req := <-br.cmdReq:
			req.dev.WriteMsg(msgTypeCmd, req.data)

		case c := <-br.webCon:
			handleWebCon(br, c)

		case req := <-br.webReq:
			handleWebReq(req)

		case data := <-br.cmdMessage:
			handleCmdResp(br, data)

		case resp := <-br.webMessage:
			handleWebResp(resp)
		}
	}
}
