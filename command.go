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
    "net/http"
    "io/ioutil"
    "encoding/json"
    "github.com/gorilla/websocket"
)

const (
    COMMAND_ERR_NONE = iota
    COMMAND_ERR_TIMEOUT
    COMMAND_ERR_NOTFOUND
    COMMAND_ERR_READ
    COMMAND_ERR_LOGIN
    COMMAND_ERR_SYS
    COMMAND_ERR_PARAMETER
    COMMAND_ERR_DEVOFFLINE
)

type CommandReq struct {
    ID uint32 `json:"id"`
    Username string `json:"username"`
    Password string `json:"password"`
    Devid string `json:"devid"`
    Cmd string `json:"cmd"`
    Params []string `json:"params"`
    Env []string `json:"env"`
}

type CommandResult struct {
    ID uint32 `json:"id,omitempty"`
    Err int `json:err`
    Msg string `json:"msg"`
    Code int `json:"code"`
    Stdout string `json:"stdout"`
    Stderr string `json:"stderr"`
}

var errStr = map[int]string {
    COMMAND_ERR_NONE: "",
    COMMAND_ERR_TIMEOUT: "timeout",
    COMMAND_ERR_NOTFOUND: "not found",
    COMMAND_ERR_READ: "read error",
    COMMAND_ERR_LOGIN: "login failed",
    COMMAND_ERR_SYS: "system error",
    COMMAND_ERR_PARAMETER: "devid and cmd required",
    COMMAND_ERR_DEVOFFLINE: "device offline",
}

func serveCmd(br *Bridge, w http.ResponseWriter, r *http.Request) {
    ticker := time.NewTicker(time.Second * 5)
    defer func() {
        ticker.Stop()
    }()

    err := COMMAND_ERR_NONE

    body, _ := ioutil.ReadAll(r.Body)
    r.Body.Close()

    req := CommandReq{}
    json.Unmarshal(body, &req)

    if req.Devid == "" || req.Cmd == "" {
        err = COMMAND_ERR_PARAMETER
    } else if dev, ok := br.devices[req.Devid]; !ok {
        err = COMMAND_ERR_DEVOFFLINE
    } else {
        req.ID = dev.cmdid
        dev.cmd[req.ID] = make(chan *wsMessage)
        dev.cmdid = dev.cmdid + 1
        if dev.cmdid == 1024 {
            dev.cmdid = 0
        }
        js, _ := json.Marshal(req)
        dev.wsWrite(websocket.TextMessage, js)

        select {
        case wsMsg := <- dev.cmd[req.ID]:
            res := CommandResult{}
            json.Unmarshal(wsMsg.data, &res)

            delete(dev.cmd, res.ID)
            res.ID = 0
            res.Msg = errStr[res.Err]
            js, _ = json.Marshal(res)

            w.Write(js)
            return
        case <- ticker.C:
            err = COMMAND_ERR_TIMEOUT
            goto Err
        }
    }

Err:
    res := CommandResult{Err: err, Msg: errStr[err]}
    js, _ := json.Marshal(res)
    w.Write(js)
}
