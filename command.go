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
	"time"

	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
)

const (
	RTTY_CMD_ERR_INVALID       = 1001
	RTTY_CMD_ERR_OFFLINE       = 1002
	RTTY_CMD_ERR_BUSY          = 1003
	RTTY_CMD_ERR_TIMEOUT       = 1004
	RTTY_CMD_ERR_PENDING       = 1005
	RTTY_CMD_ERR_INVALID_TOKEN = 1006
)

var cmdErrMsg = map[int]string{
	RTTY_CMD_ERR_INVALID:       "invalid format",
	RTTY_CMD_ERR_OFFLINE:       "device offline",
	RTTY_CMD_ERR_BUSY:          "server is busy",
	RTTY_CMD_ERR_TIMEOUT:       "timeout",
	RTTY_CMD_ERR_PENDING:       "pending",
	RTTY_CMD_ERR_INVALID_TOKEN: "invalid token",
}

type commandStatus struct {
	resp []byte
	t    *time.Timer
}

var commands = make(map[string]*commandStatus)

func handleCmdResp(data []byte) {
	token, _ := jsonparser.GetString(data, "token")
	if cmd, ok := commands[token]; ok {
		attrs, _, _, _ := jsonparser.Get(data, "attrs")
		cmd.resp = attrs
	}
}

func cmdErrReply(err int, w http.ResponseWriter) {
	msg := fmt.Sprintf(`{"err": %d, "msg":"%s"}`, err, cmdErrMsg[err])
	w.Write([]byte(msg))
}

func serveCmd(br *Broker, w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token != "" {
		if cmd, ok := commands[token]; ok {
			if len(cmd.resp) == 0 {
				cmdErrReply(RTTY_CMD_ERR_PENDING, w)
			} else {
				w.Write(cmd.resp)
				cmd.t.Stop()
				delete(commands, token)
			}
		} else {
			cmdErrReply(RTTY_CMD_ERR_INVALID_TOKEN, w)
		}
		return
	}

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

	token = UniqueId("cmd")

	commands[token] = &commandStatus{
		t: time.AfterFunc(30*time.Second, func() {
			delete(commands, token)
		}),
	}

	msg := fmt.Sprintf(`{"type":"cmd","token":"%s","attrs":%s}`, token, string(body))
	dev.wsWrite(websocket.TextMessage, []byte(msg))

	resp := fmt.Sprintf(`{"token":"%s"}`, token)
	w.Write([]byte(resp))
}
