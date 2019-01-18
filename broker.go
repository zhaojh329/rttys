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
	"github.com/zhaojh329/rttys/internal/rlog"
)

const RTTY_MESSAGE_VERSION = 2
const RTTY_MAX_SESSION_ID = 1000000

type Broker struct {
	devices    map[string]*Client
	sessions   map[string]*Session
	register   chan *Client      /* Register requests from the clients. */
	unregister chan *Client      /* Unregister requests from clients. */
	inMessage  chan *wsInMessage /* Buffered channel of inbound messages. */
}

type Session struct {
	dev    *Client
	web    *Client
	devsid uint8
}

func newBroker() *Broker {
	return &Broker{
		register:   make(chan *Client, 100),
		unregister: make(chan *Client, 100),
		devices:    make(map[string]*Client),
		sessions:   make(map[string]*Session),
		inMessage:  make(chan *wsInMessage, 10000),
	}
}

func (br *Broker) newSession(web *Client) bool {
	devid := web.devid
	sid := genUniqueID("dev")

	if dev, ok := br.devices[devid]; ok {
		devsid := dev.getFreeSid()
		if devsid < 1 {
			rlog.Println("Not found  available devsid")
			return false
		}

		br.sessions[sid] = &Session{dev, web, devsid}
		dev.sessions[devsid] = sid
		web.sid = sid

		msg := fmt.Sprintf(`{"type":"login","sid":%d}`, devsid)

		// Notify the device to create a pty and associate it with a session id
		dev.wsWrite(websocket.TextMessage, []byte(msg))

		rlog.Println("New session:", sid)
		return true
	} else {
		// Notify the user that the device is offline
		msg := `{"type":"login","err":1,"msg":"offline"}`
		web.wsWrite(websocket.TextMessage, []byte(msg))
		rlog.Println("Device", devid, "offline")
		return false
	}
}

func (br *Broker) run() {
	for {
		select {
		case c := <-br.register:
			if c.isDev {
				if _, ok := br.devices[c.devid]; ok {
					rlog.Println("ID conflicting:", c.devid)
					c.wsClose()
				} else {
					br.devices[c.devid] = c
					rlog.Printf("New device:id('%s'), description('%s')", c.devid, c.desc)
				}
			} else {
				// From user
				if !br.newSession(c) {
					time.AfterFunc(500*time.Millisecond, c.wsClose)
				}
			}
		case c := <-br.unregister:
			if c.isDev {
				c.wsClose()

				if dev, ok := br.devices[c.devid]; ok {
					rlog.Printf("Dead device:id('%s'), description('%s')", dev.devid, dev.desc)
					delete(br.devices, dev.devid)
				}

				for sid, session := range br.sessions {
					if session.dev.devid == c.devid {
						session.web.wsClose()
						delete(br.sessions, sid)
						rlog.Println("Delete session: ", sid)
					}
				}
			} else {
				if session, ok := br.sessions[c.sid]; ok {
					msg := fmt.Sprintf(`{"type":"logout","sid":%d}`, session.devsid)
					session.dev.wsWrite(websocket.TextMessage, []byte(msg))
					delete(br.sessions, c.sid)
					delete(session.dev.sessions, session.devsid)
					rlog.Println("Delete session: ", c.sid)
				}
			}
		case msg := <-br.inMessage:
			msgType := msg.msgType
			data := msg.data
			var sid string
			c := msg.c

			if c.isDev {
				var devsid uint8
				if msgType == websocket.BinaryMessage {
					devsid = data[0]
					data = data[1:]
				} else {
					tp, _ := jsonparser.GetString(data, "type")
					if tp == "cmd" {
						handleCmdResp(data)
						continue
					}
					val, _ := jsonparser.GetInt(data, "sid")
					devsid = uint8(val)
				}
				sid = c.sessions[devsid]
			} else {
				sid = c.sid
			}

			if session, ok := br.sessions[sid]; ok {
				if c.isDev {
					c = session.web
				} else {
					if msgType == websocket.BinaryMessage {
						sb := make([]byte, 1)
						sb[0] = session.devsid
						data = append(sb, data...)
					}
					c = session.dev
				}
				c.wsWrite(msgType, data)
			}
		}
	}
}
