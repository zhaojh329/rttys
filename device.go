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

	"github.com/gorilla/websocket"
)

const (
	/* Max session id for each device */
	RTTY_MAX_SESSION_ID_DEV = 5
)

type Device struct {
	*Client
	desc      string           /* description of the device */
	timestamp int64            /* Connection time */
	sessions  map[uint8]string /* sessions of each device */
}

type DeviceInfo struct {
	ID          string `json:"id"`
	Uptime      int64  `json:"uptime"`
	Description string `json:"description"`
}

type DevMessage struct {
	msgType int
	data    []byte
	dev     *Device
}

func (dev *Device) Close() {
	dev.Client.Close()
	dev.br.disconnecting <- dev
}

func (dev *Device) getFreeSid() uint8 {
	for sid := uint8(1); sid <= RTTY_MAX_SESSION_ID_DEV; sid++ {
		if _, ok := dev.sessions[sid]; !ok {
			return sid
		}
	}
	return uint8(0)
}

/*
 * If the Server does not receive a PING Packet from the Client within one and
 * a half times the Keep Alive time period, the server will disconnect the
 * Connection
 */
func (dev *Device) keepAlive(keepalive int64) {
	defer dev.Close()

	ticker := time.NewTicker(time.Second * time.Duration(keepalive))
	last := time.Now().Unix()
	keepalive = keepalive * 3

	/* Get the current ping handler */
	pingHandler := dev.ws.PingHandler()

	dev.ws.SetPingHandler(func(appData string) error {
		last = time.Now().Unix()
		return pingHandler(appData)
	})

	for {
		select {
		case <-dev.closeChan:
			return
		case <-ticker.C:
			now := time.Now().Unix()
			if now-last > keepalive {
				log.Printf("Inactive device in long time, now kill it: %s\n", dev.devid)
				return
			}
		}
	}
}

func (dev *Device) readAlway() {
	defer dev.Close()

	for {
		msgType, data, err := dev.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		msg := &DevMessage{msgType, data, dev}

		select {
		case dev.br.inDevMessage <- msg:
		case <-dev.closeChan:
			log.Println("closeChan from readAlway")
			return
		}
	}
}
