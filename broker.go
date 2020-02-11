package main

import (
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
)

type Session struct {
	dev    *Device
	user   *User
	devsid byte
}

type Broker struct {
	token       string
	login       chan *User
	logout      chan string
	register    chan *Device
	unregister  chan *Device
	devices     map[string]*Device
	sessions    map[string]*Session
	commands    map[string]*CommandStatus
	newSession  chan *Session
	cmdReq      chan *CommandReq
	devMessage  chan *DevMessage
	userMessage chan *UsrMessage
	cmdMessage  chan []byte
	clearCmd    chan string
}

func newBroker(token string) *Broker {
	return &Broker{
		token:       token,
		login:       make(chan *User, 10),
		logout:      make(chan string, 10),
		register:    make(chan *Device, 1000),
		unregister:  make(chan *Device, 1000),
		devices:     make(map[string]*Device),
		sessions:    make(map[string]*Session),
		newSession:  make(chan *Session, 10),
		commands:    make(map[string]*CommandStatus),
		cmdReq:      make(chan *CommandReq, 1000),
		devMessage:  make(chan *DevMessage, 1000),
		userMessage: make(chan *UsrMessage, 1000),
		cmdMessage:  make(chan []byte, 1000),
		clearCmd:    make(chan string, 1000),
	}
}

func (br *Broker) run() {
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

			dev.writeMsg(MsgTypeRegister, append([]byte{err}, msg...))
			if err == 1 {
				dev.close()
			}

		case dev := <-br.unregister:
			if _, ok := br.devices[dev.id]; ok {
				delete(br.devices, dev.id)
			}

			for sid, session := range br.sessions {
				if session.dev == dev {
					session.user.close()
					delete(br.sessions, sid)
					log.Info().Msg("Delete session: " + sid)
				}
			}

		case user := <-br.login:
			if dev, ok := br.devices[user.devid]; ok {
				if !dev.login(user) {
					user.loginAck(LoginErrorBusy)
					log.Error().Msgf("Device '%s' is busy", dev.id)
				}
			} else {
				user.loginAck(LoginErrorOffline)
				log.Error().Msgf("Not found the device '%s'", user.devid)
			}

		case sid := <-br.logout:
			if session, ok := br.sessions[sid]; ok {
				delete(br.sessions, sid)
				session.user.close()
				session.dev.logout(sid[len(sid)-1] - '0')
				log.Info().Msg("Delete session: " + sid)
			}

		case session := <-br.newSession:
			sid := session.dev.id + string(session.devsid+'0')
			session.user.sid = sid
			session.user.loginAck(LoginErrorNone)
			br.sessions[sid] = session
			log.Info().Msg("New session: " + sid)

		case msg := <-br.devMessage:
			sid := msg.devid + string(msg.sid+'0')
			if session, ok := br.sessions[sid]; ok {
				data := []byte{0}
				if msg.isFileMsg {
					data[0] = 1
				}
				session.user.writeMessage(websocket.BinaryMessage, append(data, msg.data...))
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
						session.dev.writeMsg(MsgTypeFile, data)
					} else {
						session.dev.writeMsg(MsgTypeTermData, append([]byte{devsid}, data...))
					}
				} else {
					typ := jsoniter.Get(msg.data, "type").ToString()
					switch typ {
					case "winsize":
						cols := jsoniter.Get(msg.data, "cols").ToInt()
						rows := jsoniter.Get(msg.data, "rows").ToInt()
						data = append([]byte{devsid}, intToBytes(cols, 2)...)
						data = append(data, intToBytes(rows, 2)...)
						session.dev.writeMsg(MsgTypeWinsize, data)
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
