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
    "time"
    "strconv"
    "math/rand"
    "crypto/md5"
    "encoding/hex"
    "github.com/gorilla/websocket"
    "github.com/golang/protobuf/proto"
    "github.com/zhaojh329/rttys/rtty"
)

const RTTY_MESSAGE_VERSION = 2

type Broker struct {
    devices map[string]*Client
    sessions map[string]*Session

    // Join requests from the clients.
    join chan *Client

    // Leave requests from clients.
    leave chan *Client

    // Buffered channel of inbound messages from device.
    inDevMessage chan *wsMessage

    // Buffered channel of inbound messages from user.
    inUsrMessage chan *wsMessage
}

type Session struct {
    dev *Client
    user *Client
}

func newBroker() *Broker {
    return &Broker{
        join: make(chan *Client, 100),
        leave: make(chan *Client, 100),
        devices: make(map[string]*Client),
        sessions: make(map[string]*Session),
        inDevMessage: make(chan *wsMessage, 100),
        inUsrMessage: make(chan *wsMessage, 100),
    }
}

func RttyMessageInit(msg *rtty.RttyMessage) []byte {
    data, _ := proto.Marshal(msg)
    return data
}

func generateSID(devid string) string {
    md5Ctx := md5.New()
    md5Ctx.Write([]byte(devid + strconv.FormatFloat(rand.Float64(), 'e', 6, 32)))
    cipherStr := md5Ctx.Sum(nil)
    return hex.EncodeToString(cipherStr)
}

func (br *Broker) newSession(user *Client) bool {
    devid := user.devid
    sid := generateSID(devid)

    if dev, ok := br.devices[devid]; ok {
        br.sessions[sid] = &Session{dev, user}
        user.sid = sid

        // Write to user
        msg := RttyMessageInit(&rtty.RttyMessage{
            Version: RTTY_MESSAGE_VERSION,
            Type: rtty.RttyMessage_LOGINACK,
            Sid: sid,
            Code: rtty.RttyMessage_LoginCode_value["OK"],
        })
        user.wsWrite(websocket.BinaryMessage, msg)

        
        // Write to device
        msg = RttyMessageInit(&rtty.RttyMessage{
            Version: RTTY_MESSAGE_VERSION,
            Type: rtty.RttyMessage_LOGIN,
            Sid: sid,
        })
        dev.wsWrite(websocket.BinaryMessage, msg)
        
        rlog.Println("New session:", sid)
        return true
    } else {
        // Write to user
        msg := RttyMessageInit(&rtty.RttyMessage{
            Version: RTTY_MESSAGE_VERSION,
            Type: rtty.RttyMessage_LOGINACK,
            Sid: sid,
            Code: rtty.RttyMessage_LoginCode_value["OFFLINE"],
        })
        user.wsWrite(websocket.BinaryMessage, msg)

        rlog.Println("Device", devid, "offline")
        return false
    }
}

func delSession(sessions map[string]*Session, sid string) {
    if session, ok := sessions[sid]; ok {
        delete(sessions, sid)
        session.user.wsClose()
        rlog.Println("Delete session: ", sid)

        if session.dev != nil {
            msg := RttyMessageInit(&rtty.RttyMessage{
                Version: RTTY_MESSAGE_VERSION,
                Type: rtty.RttyMessage_LOGOUT,
                Sid: sid,
            })
            session.dev.wsWrite(websocket.BinaryMessage, msg)
        }
    }
}

func dispatchMsg(data []byte, isDev bool, br *Broker) {
    msg := &rtty.RttyMessage{};
    proto.Unmarshal(data, msg);

    if msg.Type == rtty.RttyMessage_COMMAND {
        cmdMutex.Lock()
        if cmd, ok := command[msg.Id]; ok {
            cmd <- msg
        }
        cmdMutex.Unlock()
        return;
    }

    if session, ok := br.sessions[msg.Sid]; ok {
        if msg.Type == rtty.RttyMessage_LOGOUT {
            session.dev = nil
            delSession(br.sessions, msg.Sid)
            return
        }

        if isDev {
            session.user.wsWrite(websocket.BinaryMessage, data)
        } else {
            session.dev.wsWrite(websocket.BinaryMessage, data)
        }
    }
}

func (br *Broker) run() {
    for {
        select {
        case client := <- br.join:
            if client.isDev {
                if _, ok := br.devices[client.devid]; ok {
                    rlog.Println("ID conflicting:", client.devid)
                    client.wsClose();
                } else {
                    client.isJoined = true
                    br.devices[client.devid] = client
                    rlog.Printf("New device:id('%s'), description('%s')", client.devid, client.description)
                }
            } else {
                // From user browse
                if !br.newSession(client) {
                    time.AfterFunc(500 * time.Millisecond, client.wsClose)
                }
            }
        case client := <- br.leave:
            if client.isDev {
                client.wsClose()

                if dev, ok := br.devices[client.devid]; ok {
                    rlog.Printf("Dead device:id('%s'), description('%s')", dev.devid, dev.description)
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
        case msg := <- br.inDevMessage:
            dispatchMsg(msg.data, true, br)
        case msg := <- br.inUsrMessage:
            dispatchMsg(msg.data, false, br)
        }
    }
}
