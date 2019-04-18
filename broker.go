/*
 * Copyright (C) 2017 Jianhui Zhao <jianhuizhao329@gmail.com>
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU Lesser General Public
 * License as published by the Free Software Foundation; either
 * version 2.1 of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public
 * License along with this library; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301
 * USA
 */

package main

import (
	"fmt"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
)

const (
	RTTY_MESSAGE_VERSION = 2
	RTTY_MAX_SESSION_ID  = 1000000
)

type Broker struct {
	devices       map[string]*Device
	sessions      map[string]*Session
	connecting    chan *Device     /* Connecting requests from Device. */
	disconnecting chan *Device     /* Disconnecting requests from Device. */
	logining      chan *User       /* Login requests from the User. */
	logouting     chan *User       /* Logout requests from the User. */
	inDevMessage  chan *DevMessage /* Buffered channel of inbound messages from device. */
	inUsrMessage  chan *UsrMessage /* Buffered channel of inbound messages from user. */
}

type Session struct {
	dev    *Device
	user   *User
	devsid uint8
}

func newBroker() *Broker {
	return &Broker{
		connecting:    make(chan *Device, 100),
		disconnecting: make(chan *Device, 100),
		logining:      make(chan *User, 100),
		logouting:     make(chan *User, 100),
		devices:       make(map[string]*Device),
		sessions:      make(map[string]*Session),
		inDevMessage:  make(chan *DevMessage, 1000),
		inUsrMessage:  make(chan *UsrMessage, 1000),
	}
}

func (br *Broker) newSession(user *User) bool {
	devid := user.devid
	sid := genUniqueID("tty")

	if dev, ok := br.devices[devid]; ok {
		devsid := dev.getFreeSid()
		if devsid < 1 {
			log.Println("Not found  available devsid")
			return false
		}

		br.sessions[sid] = &Session{dev, user, devsid}
		dev.sessions[devsid] = sid
		user.sid = sid

		msg := fmt.Sprintf(`{"type":"login","sid":%d}`, devsid)

		// Notify the device to create a pty and associate it with a session id
		dev.wsWrite(websocket.TextMessage, []byte(msg))

		log.Println("New session:", sid)
		return true
	} else {
		// Notify the user that the device is offline
		msg := `{"type":"login","err":1,"msg":"offline"}`
		user.wsWrite(websocket.TextMessage, []byte(msg))
		log.Println("Device", devid, "offline")
		return false
	}
}

func (br *Broker) run() {
	for {
		select {
		case dev := <-br.connecting:
			if _, ok := br.devices[dev.devid]; ok {
				log.Println("ID conflicting:", dev.devid)
				dev.Close()
			} else {
				br.devices[dev.devid] = dev
				log.Println("New device:", dev.devid)
			}

		case dev := <-br.disconnecting:
			if dev, ok := br.devices[dev.devid]; ok {
				delete(br.devices, dev.devid)

				log.Println("Died device:", dev.devid)

				for sid, session := range br.sessions {
					if session.dev.devid == dev.devid {
						session.user.Close()
						delete(br.sessions, sid)
						log.Println("Delete session: ", sid)
					}
				}
			}

		case user := <-br.logining:
			if !br.newSession(user) {
				time.AfterFunc(500*time.Millisecond, user.Close)
			}

		case user := <-br.logouting:
			if session, ok := br.sessions[user.sid]; ok {
				devsid := session.devsid
				dev := session.dev
				sid := user.sid

				msg := fmt.Sprintf(`{"type":"logout","sid":%d}`, devsid)
				dev.wsWrite(websocket.TextMessage, []byte(msg))

				delete(br.sessions, sid)
				delete(session.dev.sessions, devsid)

				log.Println("Delete session: ", sid)
			}

		case msg := <-br.inDevMessage:
			msgType := msg.msgType
			data := msg.data
			devsid := uint8(0)

			if msgType == websocket.BinaryMessage {
				devsid = data[0]
				data = data[1:]
			} else {
				typ, _ := jsonparser.GetString(data, "type")
				if typ == "cmd" {
					handleCmdResp(data)
					continue
				}
				val, _ := jsonparser.GetInt(data, "sid")
				devsid = uint8(val)
			}

			sid := msg.dev.sessions[devsid]

			if session, ok := br.sessions[sid]; ok {
				session.user.wsWrite(msgType, data)
			}

		case msg := <-br.inUsrMessage:
			msgType := msg.msgType
			data := msg.data

			if session, ok := br.sessions[msg.user.sid]; ok {
				if msgType == websocket.BinaryMessage {
					data = append([]byte{session.devsid}, data...)
				}
				session.dev.wsWrite(msgType, data)
			}
		}
	}
}
