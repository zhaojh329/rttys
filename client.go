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
    "sync"
    "errors"
    "net/http"
    "github.com/gorilla/websocket"
)

const (
    // Time allowed to read the next pong message from the peer.
    pongWait = 3 * time.Second

    // pings to peer with this period.
    pingPeriod = 5 * time.Second
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

type wsMessage struct {
    msgType int
    data []byte
}

// Representing a device or user browser
type Client struct {
    br *Broker
    isDev bool
    // device description
    description string
    devid string
    // Registration time
    timestamp int64
    sid string
    conn *websocket.Conn
    // Buffered channel of outbound messages.
    outMessage chan *wsMessage

    cmdid uint32
    cmd map[uint32]chan *wsMessage

    isJoined bool

    // Avoid repeated closes and concurrent map writes
    mutex sync.Mutex
    isClosed bool
    closeChan chan byte
}

func (c *Client) wsClose() {
    defer c.mutex.Unlock()
    c.mutex.Lock()

    if !c.isClosed {
        c.conn.Close()
        c.isClosed = true
        close(c.closeChan)
    }
}
func (c *Client) leave() {
    if c.isJoined {
        c.br.leave <- c
    }
}

func (c *Client) wsWrite(messageType int, data []byte) error {
    select {
    case c.outMessage <- &wsMessage{messageType, data}:
    case <- c.closeChan:
        return errors.New("websocket closed")
    }
    return nil
}

func (c *Client) readPump() {
    defer func() {
        c.leave()
    }()

    c.conn.SetPongHandler(func(string) error {
        /* Set not time out */
        c.conn.SetReadDeadline(time.Time{});
        return nil
    })

    for {
        msgType, data, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("error: %v", err)
            }
            break
        }

        msg := &wsMessage{msgType, data}

        inMessage := c.br.inUsrMessage
        if c.isDev {
            inMessage = c.br.inDevMessage
        }        

        select {
        case inMessage <- msg:
        case <- c.closeChan:
            return
        }
    }
}

func (c *Client) writePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.leave()
    }()

    for {
        select {
        case msg := <- c.outMessage:
            if err := c.conn.WriteMessage(msg.msgType, msg.data); err != nil {
                return
            }
        case <- ticker.C:
            c.conn.SetReadDeadline(time.Now().Add(pongWait));
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        case <- c.closeChan:
            return
        }
    }
}

/* serveWs handles websocket requests from the peer. */
func serveWs(br *Broker, w http.ResponseWriter, r *http.Request) {
    devid := r.URL.Query().Get("devid")
    if devid == "" {
        log.Println("devid required")
        return
    }

    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }

    client := &Client{
        br: br,
        devid: devid,
        conn: conn,
        timestamp: time.Now().Unix(),
        outMessage: make(chan *wsMessage, 100),
        closeChan: make(chan byte),
        isClosed: false,
    }

    isDev := r.URL.Query().Get("device")
    if isDev == "1" {
        client.isDev = true
        client.description = r.URL.Query().Get("description")
        client.cmd = make(map[uint32]chan *wsMessage)
    }

    client.br.join <- client

    go client.readPump()
    go client.writePump()
}
