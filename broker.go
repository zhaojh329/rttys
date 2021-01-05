package main

import (
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
)

type session struct {
	dev    *device
	u      *user
	devsid byte
}

type broker struct {
	token       string
	login       chan *user
	logout      chan string
	register    chan *device
	unregister  chan *device
	devices     map[string]*device
	sessions    map[string]*session
	commands    map[string]*commandStatus
	newSession  chan *session
	cmdReq      chan *commandReq
	devMessage  chan *devMessage
	userMessage chan *usrMessage
	cmdMessage  chan []byte
	clearCmd    chan string
}

func newBroker(token string) *broker {
	return &broker{
		token:       token,
		login:       make(chan *user, 10),
		logout:      make(chan string, 10),
		register:    make(chan *device, 1000),
		unregister:  make(chan *device, 1000),
		devices:     make(map[string]*device),
		sessions:    make(map[string]*session),
		newSession:  make(chan *session, 10),
		commands:    make(map[string]*commandStatus),
		cmdReq:      make(chan *commandReq, 1000),
		devMessage:  make(chan *devMessage, 1000),
		userMessage: make(chan *usrMessage, 1000),
		cmdMessage:  make(chan []byte, 1000),
		clearCmd:    make(chan string, 1000),
	}
}

func (br *broker) run() {
	for {
		select {
		case dev := <-br.register:
			err := byte(0)
			msg := "OK"

			if _, ok := br.devices[dev.id]; ok {
				log.Error().Msg("Device ID conflicting: " + dev.id)
				msg = "ID conflicting"
				err = 1
			} else if dev.token != br.token {
				log.Error().Msg("Invalid token from terminal device")
				msg = "Invalid token"
				err = 1
			} else {
				br.devices[dev.id] = dev
				log.Info().Msg("New device: " + dev.id)
			}

			dev.writeMsg(msgTypeRegister, append([]byte{err}, msg...))
			if err == 1 {
				dev.close()
			}

		case dev := <-br.unregister:
			if _, ok := br.devices[dev.id]; ok {
				delete(br.devices, dev.id)
			}

			for sid, session := range br.sessions {
				if session.dev == dev {
					session.u.close()
					delete(br.sessions, sid)
					log.Info().Msg("Delete session: " + sid)
				}
			}

		case u := <-br.login:
			if dev, ok := br.devices[u.devid]; ok {
				if !dev.login(u) {
					u.loginAck(loginErrorBusy)
					log.Error().Msgf("Device '%s' is busy", dev.id)
				}
			} else {
				u.loginAck(loginErrorOffline)
				log.Error().Msgf("Not found the device '%s'", u.devid)
			}

		case sid := <-br.logout:
			if session, ok := br.sessions[sid]; ok {
				delete(br.sessions, sid)
				session.u.close()
				session.dev.logout(sid[len(sid)-1] - '0')
				log.Info().Msg("Delete session: " + sid)
			}

		case session := <-br.newSession:
			sid := session.dev.id + string(session.devsid+'0')
			session.u.sid = sid
			session.u.loginAck(loginErrorNone)
			br.sessions[sid] = session
			log.Info().Msg("New session: " + sid)

		case msg := <-br.devMessage:
			sid := msg.devid + string(msg.sid+'0')
			if session, ok := br.sessions[sid]; ok {
				data := []byte{0}
				if msg.isFileMsg {
					data[0] = 1
				}
				session.u.writeMessage(websocket.BinaryMessage, append(data, msg.data...))
			}

		case msg := <-br.userMessage:
			msgType := msg.msgType
			data := msg.data
			if session, ok := br.sessions[msg.sid]; ok {
				devsid := msg.sid[len(msg.sid)-1] - '0'
				if msgType == websocket.BinaryMessage {
					isFileMsg := data[0] == 1
					data = data[1:]
					if isFileMsg {
						session.dev.writeMsg(msgTypeFile, data)
					} else {
						session.dev.writeMsg(msgTypeTermData, append([]byte{devsid}, data...))
					}
				} else {
					typ := jsoniter.Get(msg.data, "type").ToString()
					switch typ {
					case "winsize":
						cols := jsoniter.Get(msg.data, "cols").ToInt()
						rows := jsoniter.Get(msg.data, "rows").ToInt()
						data = append([]byte{devsid}, intToBytes(cols, 2)...)
						data = append(data, intToBytes(rows, 2)...)
						session.dev.writeMsg(msgTypeWinsize, data)
					}
				}
			} else {
				log.Error().Msg("Not found sid: " + msg.sid)
			}

		case cmdReq := <-br.cmdReq:
			handleCmdReq(br, cmdReq)

		case data := <-br.cmdMessage:
			handleCmdResp(br, data)

		case token := <-br.clearCmd:
			if cmd, ok := br.commands[token]; ok {
				delete(br.commands, token)
				cmd.tmr.Stop()
			}
		}
	}
}
