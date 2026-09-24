/* SPDX-License-Identifier: MIT */
/*
 * Author: Jianhui Zhao <zhaojh329@gmail.com>
 */

package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zhaojh329/rtty-go/proto"
	xlog "github.com/zhaojh329/rttys/v5/internal/log"
	"github.com/zhaojh329/rttys/v5/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
)

type User struct {
	conn    *websocket.Conn
	sid     string
	dev     *Device
	pending chan byte
	serial  bool
	close   sync.Once
	closed  atomic.Bool
}

type devicePeer interface {
	OnOpen(byte)
	OnData([]byte)
	OnFile([]byte)
	Close()
}

func (user *User) OnOpen(code byte) {
	if user.closed.Load() {
		if _, loaded := user.dev.users.LoadAndDelete(user.sid); loaded {
			user.dev.WriteMsg(proto.MsgTypeLogout, user.sid)
		}
		return
	}

	if code == 0 {
		user.WriteMsg(websocket.TextMessage, []byte(`{"type":"login"}`))
	} else if user.serial {
		user.SendCloseMsg(4100+int(code), "serial open failed")
	} else {
		user.SendCloseMsg(LoginErrorBusy, "device busy")
	}
	user.pending <- code
}

func (user *User) OnData(data []byte) {
	msg := make([]byte, len(data)+1)
	copy(msg[1:], data)
	if err := user.WriteMsg(websocket.BinaryMessage, msg); err != nil {
		user.Close()
	}
}

func (user *User) OnFile(data []byte) {
	if len(data) == 0 {
		return
	}
	switch data[0] {
	case proto.MsgTypeFileSend:
		user.WriteMsg(websocket.TextMessage,
			fmt.Appendf(nil, `{"type":"sendfile", "name": "%s"}`, string(data[1:])))
	case proto.MsgTypeFileRecv:
		user.WriteMsg(websocket.TextMessage, []byte(`{"type":"recvfile"}`))
	case proto.MsgTypeFileData:
		user.WriteMsg(websocket.BinaryMessage, append([]byte{1}, data[1:]...))
	case proto.MsgTypeFileAck:
		user.WriteMsg(websocket.TextMessage, []byte(`{"type":"fileAck"}`))
	case proto.MsgTypeFileAbort:
		user.WriteMsg(websocket.BinaryMessage, []byte{1})
	}
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
	LoginErrorOffline     = 4000
	LoginErrorBusy        = 4001
	LoginErrorTimeout     = 4002
	LoginErrorUnsupported = 4003
	LoginErrorInvalid     = 4004
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func handleUserConnection(srv *RttyServer, c *gin.Context) {
	defer xlog.RecoverPanic()

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

	user := &User{conn: conn, serial: c.Query("mode") == "serial"}

	dev := srv.GetDevice(c.Query("group"), devid)
	if dev == nil {
		user.SendCloseMsg(LoginErrorOffline, "device not found")
		conn.Close()
		return
	}
	if user.serial && dev.proto < 6 {
		user.SendCloseMsg(LoginErrorUnsupported, "serial unsupported")
		conn.Close()
		return
	}
	if mode := c.Query("mode"); mode != "" && mode != "serial" {
		user.SendCloseMsg(LoginErrorInvalid, "invalid mode")
		conn.Close()
		return
	}

	sid := utils.GenUniqueID()

	user.sid = sid
	user.dev = dev
	user.pending = make(chan byte, 1)

	dev.pending.Store(sid, user)

	defer user.Close()

	var requestErr error
	if user.serial {
		settings, err := parseSerialSettings(c)
		if err != nil {
			user.SendCloseMsg(LoginErrorInvalid, "invalid serial settings")
			return
		}

		payload, err := settings.MarshalBinary()
		if err != nil {
			user.SendCloseMsg(LoginErrorInvalid, "invalid serial settings")
			return
		}
		requestErr = dev.WriteMsg(proto.MsgTypeSerialOpen, sid, payload)
	} else {
		requestErr = dev.WriteMsg(proto.MsgTypeLogin, sid)
	}
	if requestErr != nil {
		log.Error().Msgf("send session request to device %s fail: %v", dev.id, requestErr)
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

func parseSerialSettings(c *gin.Context) (proto.SerialSettings, error) {
	baud, err := strconv.Atoi(c.Query("baudRate"))
	if err != nil {
		return proto.SerialSettings{}, err
	}
	dataBits, err := strconv.Atoi(c.Query("dataBits"))
	if err != nil {
		return proto.SerialSettings{}, err
	}
	stopBits, err := strconv.Atoi(c.Query("stopBits"))
	if err != nil {
		return proto.SerialSettings{}, err
	}

	var parity proto.SerialParity
	switch c.Query("parity") {
	case "none":
		parity = proto.SerialParityNone
	case "odd":
		parity = proto.SerialParityOdd
	case "even":
		parity = proto.SerialParityEven
	default:
		return proto.SerialSettings{}, fmt.Errorf("invalid serial parity")
	}

	settings := proto.SerialSettings{
		Port: c.Query("port"), BaudRate: baud, DataBits: dataBits,
		StopBits: stopBits, Parity: parity,
	}
	if !settings.Valid() {
		return proto.SerialSettings{}, fmt.Errorf("invalid serial settings")
	}

	return settings, nil
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
			dev.WriteMsg(proto.MsgTypeLogout, sid)
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
	timeout := TermLoginTimeout
	if user.serial {
		timeout = 10 * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return false

		case code := <-user.pending:
			return code == 0

		case <-timer.C:
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

			typ := proto.MsgTypeTermData
			if data[0] == 1 && !user.serial {
				typ = proto.MsgTypeFile
			} else if data[0] != 0 {
				return
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
				if !user.serial {
					err = dev.WriteMsg(proto.MsgTypeWinsize, sid, msg.Cols, msg.Rows)
				}

			case "ack":
				err = dev.WriteMsg(proto.MsgTypeAck, sid, msg.Ack)

			case "fileInfo":
				if !user.serial {
					err = dev.WriteMsg(proto.MsgTypeFile, sid, proto.MsgTypeFileInfo, msg.Size, msg.Name)
				}

			case "fileCanceled":
				if !user.serial {
					err = dev.WriteMsg(proto.MsgTypeFile, sid, proto.MsgTypeFileAbort)
				}

			case "fileAck":
				if !user.serial {
					err = dev.WriteMsg(proto.MsgTypeFile, sid, proto.MsgTypeFileAck)
				}
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
