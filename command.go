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
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
)

const RTTY_MAX_CMD_ID = 1000000

const (
	RTTY_CMD_ERR_INVALID = 1001
	RTTY_CMD_ERR_OFFLINE = 1002
	RTTY_CMD_ERR_BUSY    = 1003
	RTTY_CMD_ERR_TIMEOUT = 1004
)

var cmdErrMsg = map[int]string{
	RTTY_CMD_ERR_INVALID: "invalid format",
	RTTY_CMD_ERR_OFFLINE: "device offline",
	RTTY_CMD_ERR_BUSY:    "server is busy",
	RTTY_CMD_ERR_TIMEOUT: "timeout",
}

var commands = struct {
	sync.Mutex
	m map[uint32]chan []byte
}{m: make(map[uint32]chan []byte)}

// return nil for failed
func getFreeCmdChan() (chan []byte, uint32) {
	defer func() {
		commands.Unlock()
	}()

	commands.Lock()
	for id := uint32(1); id <= RTTY_MAX_CMD_ID; id++ {
		_, ok := commands.m[id]
		if !ok {
			ch := make(chan []byte)
			commands.m[id] = ch
			return ch, id
		}
	}

	return nil, 0
}

func freeCmd(id uint32) {
	commands.Lock()
	delete(commands.m, id)
	commands.Unlock()
}

func handleCmdResp(data []byte) {
	id, _ := jsonparser.GetInt(data, "id")
	if ch, ok := commands.m[uint32(id)]; ok {
		ch <- data
	}
}

func cmdErrReply(err int, w http.ResponseWriter) {
	msg := fmt.Sprintf(`{"err": %d, "msg":"%s"}`, err, cmdErrMsg[err])
	w.Write([]byte(msg))
}

func serveCmd(br *Broker, w http.ResponseWriter, r *http.Request) {
	timer := time.NewTimer(time.Second * 5)
	defer func() {
		timer.Stop()
	}()

	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()

	devid, err := jsonparser.GetString(body, "devid")
	if err != nil {
		cmdErrReply(RTTY_CMD_ERR_INVALID, w)
		return
	}

	_, err = jsonparser.GetString(body, "cmd")
	if err != nil {
		cmdErrReply(RTTY_CMD_ERR_INVALID, w)
		return
	}

	dev, ok := br.devices[devid]
	if !ok {
		cmdErrReply(RTTY_CMD_ERR_OFFLINE, w)
		return
	}

	cmdChan, id := getFreeCmdChan()
	if cmdChan == nil {
		cmdErrReply(RTTY_CMD_ERR_BUSY, w)
		return
	}

	msg := fmt.Sprintf(`{"type":"cmd","id":%d,"attrs":%s}`, id, string(body))
	dev.wsWrite(websocket.TextMessage, []byte(msg))

	select {
	case data := <-cmdChan:
		attrs, _, _, _ := jsonparser.Get(data, "attrs")
		w.Write(attrs)
	case <-timer.C:
		freeCmd(id)
		cmdErrReply(RTTY_CMD_ERR_TIMEOUT, w)
	}
}
