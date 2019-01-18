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
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	br    *Broker
	ws    *websocket.Conn
	devid string

	mutex     sync.Mutex /* Avoid repeated closes */
	closed    bool
	closeChan chan byte

	outMessage chan *wsOutMessage /* Buffered channel of outbound messages */
}

type wsOutMessage struct {
	msgType int
	data    []byte
}

func (c *Client) Close() {
	defer c.mutex.Unlock()
	c.mutex.Lock()

	if !c.closed {
		c.ws.Close()
		c.closed = true
		close(c.closeChan)
	}
}

func (c *Client) wsWrite(msgType int, data []byte) {
	c.outMessage <- &wsOutMessage{msgType, data}
}

func (c *Client) writePump() {
	defer c.Close()

	for {
		select {
		case msg := <-c.outMessage:
			if err := c.ws.WriteMessage(msg.msgType, msg.data); err != nil {
				return
			}
		case <-c.closeChan:
			return
		}
	}
}

/* serveWs handles websocket requests from the device or user. */
func serveWs(br *Broker, w http.ResponseWriter, r *http.Request) {
	keepalive, _ := strconv.Atoi(r.URL.Query().Get("keepalive"))
	isDev := r.URL.Query().Get("device") != ""
	devid := r.URL.Query().Get("devid")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	if devid == "" {
		conn.Close()
		log.Println("devid required")
		return
	}

	client := &Client{
		br:         br,
		devid:      devid,
		ws:         conn,
		closeChan:  make(chan byte),
		outMessage: make(chan *wsOutMessage, 1000),
	}

	if isDev {
		desc := r.URL.Query().Get("description")
		sessions := make(map[uint8]string)

		dev := &Device{client, desc, time.Now().Unix(), sessions}

		if keepalive > 0 {
			go dev.keepAlive(int64(keepalive))
		}

		go dev.readAlway()

		br.connecting <- dev
	} else {
		user := &User{client, ""}

		go user.readAlway()

		br.logining <- user
	}

	go client.writePump()
}
