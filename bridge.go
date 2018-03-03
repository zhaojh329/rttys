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
    "log"
    "time"
    "github.com/gorilla/websocket"
)

type Bridge struct {
    // Registered devices.
    devices map[string]*Client

    // Registered users.
    sessions map[string]*Session

    // Register requests from the clients.
    register chan *Client

    // Unregister requests from clients.
    unregister chan *Client

    // Buffered channel of inbound messages.
    inbound chan *wsMessage
}

type Session struct {
    dev *Client
    user *Client
}

func newBridge() *Bridge {
    return &Bridge{
        register: make(chan *Client),
        unregister: make(chan *Client),
        devices: make(map[string]*Client),
        sessions: make(map[string]*Session),
        inbound: make(chan *wsMessage, 100),
    }
}

func (br *Bridge) newSession(user *Client) bool {
    devid := user.devid
    sid := generateSID(devid)

    if dev, ok := br.devices[devid]; ok {
        br.sessions[sid] = &Session{dev, user}
        user.sid = sid

        // Write to user
        pkt := rttyPacketNew(RTTY_PACKET_LOGINACK)
        pkt.PutString(RTTY_ATTR_SID, sid)
        pkt.PutU8(RTTY_ATTR_CODE, 0)
        user.wsWrite(websocket.BinaryMessage, pkt.Bytes())
        
        // Write to device
        pkt = rttyPacketNew(RTTY_PACKET_LOGIN)
        pkt.PutString(RTTY_ATTR_SID, sid)
        dev.wsWrite(websocket.BinaryMessage, pkt.Bytes())
        
        log.Println("New session:", sid)
        return true
    } else {
        // Write to user
        pkt := rttyPacketNew(RTTY_PACKET_LOGINACK)
        pkt.PutString(RTTY_ATTR_SID, sid)
        pkt.PutU8(RTTY_ATTR_CODE, 1)
        user.wsWrite(websocket.BinaryMessage, pkt.Bytes())

        log.Println("Device", devid, "offline")
        return false
    }
}

func delSession(sessions map[string]*Session, sid string) {
    if session, ok := sessions[sid]; ok {
        delete(sessions, sid)
        session.user.wsClose()
        log.Println("Delete session: ", sid)

        if session.dev != nil {
            pkt := rttyPacketNew(RTTY_PACKET_LOGOUT)
            pkt.PutString(RTTY_ATTR_SID, sid)
            session.dev.wsWrite(websocket.BinaryMessage, pkt.Bytes())
        }
    }
}

func (br *Bridge) run() {
    for {
        select {
        case client := <- br.register:
            if client.isDev {
                if dev, ok := br.devices[client.devid]; ok {
                    pkt := rttyPacketNew(RTTY_PACKET_ANNOUNCE)
                    pkt.PutU8(RTTY_ATTR_CODE, 1)
                    dev.wsWrite(websocket.BinaryMessage, pkt.Bytes())
                    log.Println("ID conflicting:", dev.devid)
                } else {
                    br.devices[client.devid] = client
                    log.Printf("New device:id('%s'), description('%s')", client.devid, client.description)
                }
            } else {
                // From user browse
                if !br.newSession(client) {
                    time.AfterFunc(500 * time.Millisecond, client.wsClose)
                }
            }
        case client := <- br.unregister:
            if client.isDev {
                client.wsClose()

                if dev, ok := br.devices[client.devid]; ok {
                    log.Printf("Dead device:id('%s'), description('%s')", dev.devid, dev.description)
                    delete(br.devices, dev.devid)
                }

                for sid, session := range br.sessions {
                    if session.dev.devid == client.devid {
                        session.dev = nil
                        delSession(br.sessions, sid)
                    }
                }
            } else {
                delSession(br.sessions, client.sid)
            }
        case msg := <- br.inbound:
            pkt := rttyPacketParse(msg.data)
            if session, ok := br.sessions[pkt.sid]; ok {
                if (msg.isDev) {
                    if pkt.typ == RTTY_PACKET_LOGOUT {
                        session.dev = nil
                        delSession(br.sessions, pkt.sid)
                    } else {
                        session.user.wsWrite(websocket.BinaryMessage, msg.data)
                    }
                } else {
                    session.dev.wsWrite(websocket.BinaryMessage, msg.data)
                }
            }
        }
    }
}
