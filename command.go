package main

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"rttys/utils"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
)

const commandTimeout = 30 // second

const (
	rttyCmdErrInvalid = 1001
	rttyCmdErrOffline = 1002
	rttyCmdErrTimeout = 1003
)

var cmdErrMsg = map[int]string{
	rttyCmdErrInvalid: "invalid format",
	rttyCmdErrOffline: "device offline",
	rttyCmdErrTimeout: "timeout",
}

type CommandRespAttrs struct {
	Code   int    `json:"code"`
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}

type CommandRespInfo struct {
	Token string           `json:"token"`
	Attrs CommandRespAttrs `json:"attrs"`
}

type commandReqInfo struct {
	Cmd      string   `json:"cmd"`
	Username string   `json:"username"`
	Params   []string `json:"params"`
}

type commandReq struct {
	cancel context.CancelFunc
	c      *gin.Context
	devid  string
	data   []byte
}

var commands sync.Map

func handleCmdResp(data []byte) {
	info := &CommandRespInfo{}

	jsoniter.Unmarshal(data, info)

	if req, ok := commands.Load(info.Token); ok {
		req := req.(*commandReq)
		req.c.JSON(http.StatusOK, info.Attrs)
		req.cancel()
	}
}

func cmdErrReply(err int, req *commandReq) {
	req.c.JSON(http.StatusOK, gin.H{
		"err": err,
		"msg": cmdErrMsg[err],
	})
	req.cancel()
}

func handleCmdReq(br *broker, c *gin.Context) {
	devid := c.Param("devid")

	ctx, cancel := context.WithCancel(context.Background())

	req := &commandReq{
		cancel: cancel,
		c:      c,
		devid:  devid,
	}

	if _, ok := br.getDevice(devid); !ok {
		cmdErrReply(rttyCmdErrOffline, req)
		return
	}

	cmdInfo := commandReqInfo{}

	err := c.BindJSON(&cmdInfo)
	if err != nil || cmdInfo.Username == "" || cmdInfo.Cmd == "" {
		cmdErrReply(rttyCmdErrInvalid, req)
		return
	}

	token := utils.GenUniqueID()

	data := make([]string, 4)

	data[0] = cmdInfo.Username
	data[1] = cmdInfo.Cmd
	data[2] = token
	data[3] = string(byte(len(cmdInfo.Params)))

	msg := []byte(strings.Join(data, string(byte(0))))

	for _, param := range cmdInfo.Params {
		msg = append(msg, param...)
		msg = append(msg, 0)
	}

	req.data = msg
	br.cmdReq <- req

	waitTime := commandTimeout

	wait := c.Query("wait")
	if wait != "" {
		waitTime, _ = strconv.Atoi(wait)
	}

	if waitTime == 0 {
		c.Status(http.StatusOK)
		return
	}

	commands.Store(token, req)

	if waitTime < 0 || waitTime > commandTimeout {
		waitTime = commandTimeout
	}

	tmr := time.NewTimer(time.Second * time.Duration(waitTime))

	select {
	case <-tmr.C:
		cmdErrReply(rttyCmdErrTimeout, req)
		commands.Delete(token)
	case <-ctx.Done():
	}
}
