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
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	/* Minimal version required of the device: 6.2.0 */
	RTTY_REQUIRED_VERSION = (6 << 16) | (3 << 8) | 0

	/* Max session id for each device */
	RTTY_MAX_SESSION_ID_DEV = 5
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	conn       *websocket.Conn
	br         *Broker
	devid      string
	desc       string /* description for device */
	isDev      bool
	timestamp  int64      /* Registration time */
	mutex      sync.Mutex /* Avoid repeated closes and concurrent map writes */
	closed     bool
	closeChan  chan byte
	sessions   map[uint8]uint32
	sid        uint32
	outMessage chan *wsOutMessage /* Buffered channel of outbound messages */
}

type wsInMessage struct {
	msgType int
	data    []byte
	c       *Client
}

type wsOutMessage struct {
	msgType int
	data    []byte
}

func (c *Client) getFreeSid() uint8 {
	for sid := uint8(1); sid <= RTTY_MAX_SESSION_ID_DEV; sid++ {
		if _, ok := c.sessions[sid]; !ok {
			return sid
		}
	}
	return uint8(0)
}

func (c *Client) wsClose() {
	defer c.mutex.Unlock()
	c.mutex.Lock()

	if !c.closed {
		c.conn.Close()
		c.closed = true
		close(c.closeChan)
	}
}

func (c *Client) unregister() {
	c.br.unregister <- c
}

func (c *Client) wsWrite(msgType int, data []byte) error {
	select {
	case c.outMessage <- &wsOutMessage{msgType, data}:
	case <-c.closeChan:
		return errors.New("websocket closed")
	}
	return nil
}

func (c *Client) readPump() {
	defer func() {
		c.unregister()
	}()

	for {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				rlog.Printf("error: %v", err)
			}
			break
		}

		msg := &wsInMessage{msgType, data, c}

		select {
		case c.br.inMessage <- msg:
		case <-c.closeChan:
			return
		}
	}
}

func (c *Client) writePump() {
	defer func() {
		c.unregister()
	}()

	for {
		select {
		case msg := <-c.outMessage:
			if err := c.conn.WriteMessage(msg.msgType, msg.data); err != nil {
				return
			}
		case <-c.closeChan:
			return
		}
	}
}

/*
 * If the Server does not receive a PING Packet from the Client within one and
 * a half times the Keep Alive time period, the server will disconnect the
 * Connection
 */
func (c *Client) keepAlive(keepalive int64) {
	ticker := time.NewTicker(time.Second * time.Duration(keepalive))
	last := time.Now().Unix()
	keepalive = keepalive*3/2 + 1

	defer func() {
		c.unregister()
	}()

	/* Get the current ping handler */
	pingHandler := c.conn.PingHandler()

	c.conn.SetPingHandler(func(appData string) error {
		last = time.Now().Unix()
		return pingHandler(appData)
	})

	for {
		select {
		case <-c.closeChan:
			return
		case <-ticker.C:
			now := time.Now().Unix()
			if now-last > keepalive {
				rlog.Printf("Inactive device in long time, now kill it(%s)\n", c.devid)
				return
			}
		}
	}
}

/* serveWs handles websocket requests from the peer. */
func serveWs(br *Broker, w http.ResponseWriter, r *http.Request) {
	keepalive, _ := strconv.Atoi(r.URL.Query().Get("keepalive"))
	ver, _ := strconv.Atoi(r.URL.Query().Get("ver"))
	isDev := r.URL.Query().Get("device") != ""
	devid := r.URL.Query().Get("devid")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		rlog.Println(err)
		return
	}

	if devid == "" {
		msg := fmt.Sprintf(`{"type":"register","err":1,"msg":"devid required"}`)
		conn.WriteMessage(websocket.TextMessage, []byte(msg))
		rlog.Println("devid required")
		time.AfterFunc(100*time.Millisecond, func() {
			conn.Close()
		})
		return
	}

	if isDev {
		if ver < RTTY_REQUIRED_VERSION {
			msg := fmt.Sprintf(`{"type":"register","err":1,"msg":"version is not matched"}`)
			conn.WriteMessage(websocket.TextMessage, []byte(msg))
			rlog.Printf("version is not matched for device '%s', you need to update your server(rttys) or client(rtty) or both them", devid)
			time.AfterFunc(100*time.Millisecond, func() {
				conn.Close()
			})
			return
		}
	}

	client := &Client{
		br:         br,
		conn:       conn,
		devid:      devid,
		timestamp:  time.Now().Unix(),
		outMessage: make(chan *wsOutMessage, 10000),
		closeChan:  make(chan byte),
	}

	if isDev {
		client.isDev = true
		client.sessions = make(map[uint8]uint32)
		client.desc = r.URL.Query().Get("description")

		if keepalive > 0 {
			go client.keepAlive(int64(keepalive))
		}
	}

	go client.readPump()
	go client.writePump()

	client.br.register <- client
}
