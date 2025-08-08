/* SPDX-License-Identifier: MIT */
/*
 * Author: Jianhui Zhao <zhaojh329@gmail.com>
 */

package main

import (
	"context"
	"encoding/binary"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zhaojh329/rttys/v5/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
)

type User struct {
	conn    *websocket.Conn
	sid     string
	dev     *Device
	pending chan bool
	close   sync.Once
	closed  atomic.Bool
}

type UserMsg struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
	Ack  uint16 `json:"ack"`
	Size uint32 `json:"size"`
	Name string `json:"name"`
}

const (
	LoginErrorOffline = 4000
	LoginErrorBusy    = 4001
	LoginErrorTimeout = 4002
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func handleUserConnection(srv *RttyServer, c *gin.Context) {
	defer logPanic()

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error().Err(err).Msg("upgrade to websocket failed")
		return
	}

	devid := c.Param("devid")
	if devid == "" {
		log.Error().Msg("device ID is required")
		conn.Close()
		return
	}

	user := &User{conn: conn}

	dev := srv.GetDevice(c.Query("group"), devid)
	if dev == nil {
		user.SendCloseMsg(LoginErrorOffline, "device not found")
		conn.Close()
		return
	}

	sid := utils.GenUniqueID()

	user.sid = sid
	user.dev = dev
	user.pending = make(chan bool, 1)

	dev.pending.Store(sid, user)

	defer user.Close()

	if err := dev.WriteMsg(msgTypeLogin, sid, nil); err != nil {
		log.Error().Msgf("send login msg to device %s fail: %v", dev.id, err)
		return
	}

	ctx, cancel := context.WithCancel(dev.ctx)

	go func() {
		<-ctx.Done()
		user.Close()
	}()

	defer cancel()

	if !user.waitForLogin(dev, ctx, sid) {
		return
	}

	user.handleMsg()
}

func (user *User) SendCloseMsg(code int, text string) {
	user.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, text), time.Now().Add(time.Second))
}

func (user *User) Close() {
	user.close.Do(func() {
		dev := user.dev
		sid := user.sid

		user.closed.Store(true)

		if _, loaded := dev.users.LoadAndDelete(sid); loaded {
			dev.WriteMsg(msgTypeLogout, sid, nil)
		}

		dev.pending.Delete(sid)
		user.conn.Close()

		log.Debug().Msgf("user with session '%s' closed", sid)
	})
}

func (user *User) WriteMsg(typ int, data []byte) error {
	return user.conn.WriteMessage(typ, data)
}

func (user *User) waitForLogin(dev *Device, ctx context.Context, sid string) bool {
	for {
		select {
		case <-ctx.Done():
			return false

		case ok := <-user.pending:
			return ok

		case <-time.After(TermLoginTimeout):
			if _, loaded := dev.pending.LoadAndDelete(sid); loaded {
				log.Error().Msgf("login timeout for session %s of device %s", sid, dev.id)
				user.SendCloseMsg(LoginErrorTimeout, "login timeout")
				return false
			}
		}
	}
}

func (user *User) handleMsg() {
	dev := user.dev
	sid := user.sid

	for {
		msgType, data, err := user.conn.ReadMessage()
		if err != nil {
			if !user.closed.Load() {
				closeError, ok := err.(*websocket.CloseError)
				if !ok || ignoredWsCloseError(closeError.Code) {
					log.Error().Msgf("user read fail: %v", err)
				}
			}
			return
		}

		if msgType == websocket.BinaryMessage {
			if len(data) < 1 {
				log.Error().Msgf("invalid msg from user")
				return
			}

			typ := msgTypeTermData
			if data[0] == 1 {
				typ = msgTypeFile
			}

			err = dev.WriteMsg(typ, sid, data[1:])
		} else {
			msg := &UserMsg{}

			err = jsoniter.Unmarshal(data, msg)
			if err != nil {
				log.Error().Msgf("invalid msg from user")
				return
			}

			switch msg.Type {
			case "winsize":
				b := make([]byte, 4)

				binary.BigEndian.PutUint16(b, msg.Cols)
				binary.BigEndian.PutUint16(b[2:], msg.Rows)

				err = dev.WriteMsg(msgTypeWinsize, sid, b)

			case "ack":
				b := make([]byte, 2)
				binary.BigEndian.PutUint16(b, msg.Ack)
				err = dev.WriteMsg(msgTypeAck, sid, b)

			case "fileInfo":
				b := make([]byte, 4+len(msg.Name))
				binary.BigEndian.PutUint32(b, msg.Size)
				copy(b[4:], []byte(msg.Name))

				err = dev.WriteFileMsg(msgTypeFile, sid, msgTypeFileInfo, b)

			case "fileCanceled":
				err = dev.WriteFileMsg(msgTypeFile, sid, msgTypeFileAbort, nil)

			case "fileAck":
				err = dev.WriteFileMsg(msgTypeFile, sid, msgTypeFileAck, nil)
			}
		}

		if err != nil {
			log.Error().Msgf("write msg to device '%s' fail: %v", dev.id, err)
			return
		}
	}
}

func ignoredWsCloseError(code int) bool {
	return code != websocket.CloseGoingAway &&
		code != websocket.CloseAbnormalClosure &&
		code != websocket.CloseNormalClosure
}
