package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	jsoniter "github.com/json-iterator/go"
)

const CommandTimeout = time.Second * 30

const (
	RttyCmdErrInvalid      = 1001
	RttyCmdErrOffline      = 1002
	RttyCmdErrBusy         = 1003
	RttyCmdErrTimeout      = 1004
	RttyCmdErrPending      = 1005
	RttyCmdErrInvalidToken = 1006
)

var cmdErrMsg = map[int]string{
	RttyCmdErrInvalid:      "invalid format",
	RttyCmdErrOffline:      "device offline",
	RttyCmdErrBusy:         "server is busy",
	RttyCmdErrTimeout:      "timeout",
	RttyCmdErrPending:      "pending",
	RttyCmdErrInvalidToken: "invalid token",
}

type CommandStatus struct {
	ts    time.Time
	token string
	resp  string
	tmr   *time.Timer
}

type CommandInfo struct {
	Cmd      string `json:"cmd"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type CommandReq struct {
	done    chan struct{}
	token   string
	content []byte
	devid   string
	w       http.ResponseWriter
}

func handleCmdResp(br *Broker, data []byte) {
	token := jsoniter.Get(data, "token").ToString()

	if cmd, ok := br.commands[token]; ok {
		cmd.resp = jsoniter.Get(data, "attrs").ToString()
	}
}

func cmdErrReply(err int, req *CommandReq) {
	fmt.Fprintf(req.w, `{"err": %d, "msg":"%s"}`, err, cmdErrMsg[err])
	close(req.done)
}

func handleCmdReq(br *Broker, req *CommandReq) {
	token := req.token

	if token != "" {
		if cmd, ok := br.commands[token]; ok {
			if len(cmd.resp) == 0 {
				if time.Now().Sub(cmd.ts) > CommandTimeout {
					cmdErrReply(RttyCmdErrTimeout, req)
				} else {
					cmdErrReply(RttyCmdErrPending, req)
				}
			} else {
				io.WriteString(req.w, cmd.resp)
				close(req.done)
				br.clearCmd <- token
			}
		} else {
			cmdErrReply(RttyCmdErrInvalidToken, req)
		}
		return
	}

	cmdInfo := CommandInfo{}
	err := jsoniter.Unmarshal(req.content, &cmdInfo)
	if err != nil || cmdInfo.Cmd == "" {
		cmdErrReply(RttyCmdErrInvalid, req)
		return
	}

	dev, ok := br.devices[req.devid]
	if !ok {
		cmdErrReply(RttyCmdErrOffline, req)
		return
	}

	token = genUniqueID("cmd")

	cmd := &CommandStatus{
		ts:    time.Now(),
		token: token,
		tmr: time.AfterFunc(CommandTimeout+time.Second*2, func() {
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

	dev.writeMsg(MsgTypeCmd, data)

	fmt.Fprintf(req.w, `{"token":"%s"}`, token)
	close(req.done)
}
