package main

import (
	"context"
	"io"
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

type commandInfo struct {
	Cmd      string `json:"cmd"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type commandReq struct {
	cancel context.CancelFunc
	c      *gin.Context
	devid  string
	data   []byte
}

var commands sync.Map

func handleCmdResp(data []byte) {
	token := jsoniter.Get(data, "token").ToString()

	if req, ok := commands.Load(token); ok {
		req := req.(*commandReq)
		req.c.String(http.StatusOK, jsoniter.Get(data, "attrs").ToString())
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

	content, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	cmdInfo := commandInfo{}
	err = jsoniter.Unmarshal(content, &cmdInfo)
	if err != nil || cmdInfo.Cmd == "" {
		cmdErrReply(rttyCmdErrInvalid, req)
		return
	}

	token := utils.GenUniqueID()

	params := jsoniter.Get(content, "params")

	data := make([]string, 4)

	data[0] = jsoniter.Get(content, "username").ToString()
	data[1] = jsoniter.Get(content, "cmd").ToString()
	data[2] = token
	data[3] = string(byte(params.Size()))

	msg := []byte(strings.Join(data, string(byte(0))))

	for i := range params.Size() {
		msg = append(msg, params.Get(i).ToString()...)
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
