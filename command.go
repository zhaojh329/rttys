package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	jsoniter "github.com/json-iterator/go"
)

const commandTimeout = time.Second * 30

const (
	rttyCmdErrInvalid      = 1001
	rttyCmdErrOffline      = 1002
	rttyCmdErrBusy         = 1003
	rttyCmdErrTimeout      = 1004
	rttyCmdErrPending      = 1005
	rttyCmdErrInvalidToken = 1006
)

var cmdErrMsg = map[int]string{
	rttyCmdErrInvalid:      "invalid format",
	rttyCmdErrOffline:      "device offline",
	rttyCmdErrBusy:         "server is busy",
	rttyCmdErrTimeout:      "timeout",
	rttyCmdErrPending:      "pending",
	rttyCmdErrInvalidToken: "invalid token",
}

type commandStatus struct {
	ts    time.Time
	token string
	resp  string
	tmr   *time.Timer
}

type commandInfo struct {
	Cmd      string `json:"cmd"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type commandReq struct {
	done    chan struct{}
	token   string
	content []byte
	devid   string
	w       http.ResponseWriter
}

func handleCmdResp(br *broker, data []byte) {
	token := jsoniter.Get(data, "token").ToString()

	if cmd, ok := br.commands[token]; ok {
		cmd.resp = jsoniter.Get(data, "attrs").ToString()
	}
}

func cmdErrReply(err int, req *commandReq) {
	fmt.Fprintf(req.w, `{"err": %d, "msg":"%s"}`, err, cmdErrMsg[err])
	close(req.done)
}

func handleCmdReq(br *broker, req *commandReq) {
	token := req.token

	if token != "" {
		if cmd, ok := br.commands[token]; ok {
			if len(cmd.resp) == 0 {
				if time.Now().Sub(cmd.ts) > commandTimeout {
					cmdErrReply(rttyCmdErrTimeout, req)
				} else {
					cmdErrReply(rttyCmdErrPending, req)
				}
			} else {
				io.WriteString(req.w, cmd.resp)
				close(req.done)
				br.clearCmd <- token
			}
		} else {
			cmdErrReply(rttyCmdErrInvalidToken, req)
		}
		return
	}

	cmdInfo := commandInfo{}
	err := jsoniter.Unmarshal(req.content, &cmdInfo)
	if err != nil || cmdInfo.Cmd == "" {
		cmdErrReply(rttyCmdErrInvalid, req)
		return
	}

	dev, ok := br.devices[req.devid]
	if !ok {
		cmdErrReply(rttyCmdErrOffline, req)
		return
	}

	token = genUniqueID("cmd")

	cmd := &commandStatus{
		ts:    time.Now(),
		token: token,
		tmr: time.AfterFunc(commandTimeout+time.Second*2, func() {
			br.clearCmd <- token
		}),
	}

	br.commands[token] = cmd

	username := jsoniter.Get(req.content, "username").ToString()
	password := jsoniter.Get(req.content, "password").ToString()
	cmdName := jsoniter.Get(req.content, "cmd").ToString()
	params := jsoniter.Get(req.content, "params")

	var data []byte
	data = append(data, username...)
	data = append(data, 0)

	data = append(data, password...)
	data = append(data, 0)

	data = append(data, cmdName...)
	data = append(data, 0)

	data = append(data, token...)
	data = append(data, 0)

	data = append(data, byte(params.Size()))
	for i := 0; i < params.Size(); i++ {
		data = append(data, params.Get(i).ToString()...)
		data = append(data, 0)
	}

	dev.writeMsg(msgTypeCmd, data)

	fmt.Fprintf(req.w, `{"token":"%s"}`, token)
	close(req.done)
}
