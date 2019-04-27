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
	jsoniter "github.com/json-iterator/go"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

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
	token string
	resp  string
	t     *time.Timer
}

type CommandInfo struct {
	Devid string `json:"devid"`
	Cmd   string `json:"cmd"`
}

var commands sync.Map

func handleCmdResp(data []byte) {
	token := jsoniter.Get(data, "token").ToString()

	if cmd, ok := commands.Load(token); ok {
		cmd := cmd.(*commandStatus)
		cmd.resp = jsoniter.Get(data, "attrs").ToString()
	}
}

func cmdErrReply(err int, w http.ResponseWriter) {
	fmt.Fprintf(w, `{"err": %d, "msg":"%s"}`, err, cmdErrMsg[err])
}

func serveCmd(br *Broker, w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token != "" {
		cmd, ok := commands.Load(token)
		if ok {
			cmd := cmd.(*commandStatus)
			if len(cmd.resp) == 0 {
				cmdErrReply(RTTY_CMD_ERR_PENDING, w)
			} else {
				commands.Delete(token)
				io.WriteString(w, cmd.resp)
				cmd.t.Stop()
			}
		} else {
			cmdErrReply(RTTY_CMD_ERR_INVALID_TOKEN, w)
		}
		return
	}

	body, _ := ioutil.ReadAll(r.Body)

	cmdInfo := CommandInfo{}
	err := jsoniter.Unmarshal(body, &cmdInfo)
	if err != nil || cmdInfo.Cmd == "" || cmdInfo.Devid == "" {
		cmdErrReply(RTTY_CMD_ERR_INVALID, w)
		return
	}

	dev, ok := br.devices[cmdInfo.Devid]
	if !ok {
		cmdErrReply(RTTY_CMD_ERR_OFFLINE, w)
		return
	}

	token = genUniqueID("cmd")

	cmd := &commandStatus{
		token: token,
		t: time.AfterFunc(30*time.Second, func() {
			commands.Delete(token)
		}),
	}

	commands.Store(token, cmd)

	msg := fmt.Sprintf(`{"type":"cmd","token":"%s","attrs":%s}`, token, body)
	dev.wsWrite(websocket.TextMessage, []byte(msg))

	fmt.Fprintf(w, `{"token":"%s"}`, token)
}
