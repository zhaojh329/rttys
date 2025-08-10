/* SPDX-License-Identifier: MIT */
/*
 * Author: Jianhui Zhao <zhaojh329@gmail.com>
 */

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
	"github.com/zhaojh329/rtty-go/proto"
	"github.com/zhaojh329/rttys/v5/utils"
)

type DeviceInfo struct {
	Group     string `json:"group"`
	ID        string `json:"id"`
	Connected uint32 `json:"connected"`
	Uptime    uint32 `json:"uptime"`
	Desc      string `json:"description"`
	Proto     uint8  `json:"proto"`
	IPaddr    string `json:"ipaddr"`
}

type Device struct {
	group     string
	id        string
	proto     uint8
	desc      string
	timestamp int64
	uptime    uint32
	token     string
	heartbeat time.Duration

	users    sync.Map
	pending  sync.Map
	commands sync.Map
	https    sync.Map

	conn   net.Conn
	close  sync.Once
	ctx    context.Context
	cancel context.CancelFunc

	msg *proto.MsgReaderWriter
}

const (
	devRegErrUnsupportedProto = iota + 1
	devRegErrInvalidToken
	devRegErrHookFailed
	devRegErrIdConflicting
)

const (
	RttyProtoRequired uint8 = 3
	WaitRegistTimeout       = 5 * time.Second
	DefaultHeartbeat        = 5 * time.Second
	TermLoginTimeout        = 5 * time.Second
	CommandTimeout          = 30
)

var DevRegErrMsg = map[byte]string{
	0:                         "Success",
	devRegErrUnsupportedProto: "Unsupported protocol",
	devRegErrInvalidToken:     "Invalid token",
	devRegErrHookFailed:       "Hook failed",
	devRegErrIdConflicting:    "ID conflict",
}

var DeviceMsgHandlers = map[byte]func(*Device, []byte) error{
	proto.MsgTypeHeartbeat: handleHeartbeatMsg,
	proto.MsgTypeLogin:     handleLoginMsg,
	proto.MsgTypeLogout:    handleLogoutMsg,
	proto.MsgTypeTermData:  handleTermDataMsg,
	proto.MsgTypeFile:      handleFileMsg,
	proto.MsgTypeCmd:       handleCmdMsg,
	proto.MsgTypeHttp:      handleHttpMsg,
}

func (srv *RttyServer) ListenDevices() {
	cfg := &srv.cfg

	ln, err := net.Listen("tcp", cfg.AddrDev)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer ln.Close()

	if cfg.SslCert != "" && cfg.SslKey != "" {
		cert, err := tls.LoadX509KeyPair(cfg.SslCert, cfg.SslKey)
		if err != nil {
			log.Fatal().Msg(err.Error())
		}

		config := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}

		if cfg.CaCert != "" {
			caCert, err := os.ReadFile(cfg.CaCert)
			if err != nil {
				log.Fatal().Msg(err.Error())
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			config.ClientCAs = caCertPool
			config.ClientAuth = tls.RequireAndVerifyClientCert
		}

		ln = tls.NewListener(ln, config)

		log.Info().Msgf("Listen devices on: %s SSL on", ln.Addr().(*net.TCPAddr))
	} else {
		log.Info().Msgf("Listen devices on: %s SSL off", ln.Addr().(*net.TCPAddr))
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Error().Msg(err.Error())
			continue
		}

		go handleDeviceConnection(srv, conn)
	}
}

func handleDeviceConnection(srv *RttyServer, conn net.Conn) {
	defer logPanic()

	dev := &Device{
		conn:      conn,
		heartbeat: DefaultHeartbeat,
		timestamp: time.Now().Unix(),

		msg: proto.NewMsgReaderWriter(proto.RoleRttys, conn),
	}
	defer dev.Close(srv)

	dev.ctx, dev.cancel = context.WithCancel(context.Background())

	log.Debug().Msgf("new device '%s' connected", conn.RemoteAddr())

	conn.SetReadDeadline(time.Now().Add(WaitRegistTimeout))

	typ, data, err := dev.ReadMsg()
	if err != nil {
		log.Error().Msgf("read register msg fail: %v", err)
		return
	}

	if typ != proto.MsgTypeRegister {
		log.Error().Msg("register msg expected first")
		return
	}

	err = dev.ParseRegister(data)
	if err != nil {
		log.Error().Err(err).Msg("invalid device info")
		return
	}

	code := dev.Register(srv)

	err = dev.WriteMsg(proto.MsgTypeRegister, code, DevRegErrMsg[code])
	if err != nil {
		log.Error().Err(err).Msgf("send register to device '%s' fail", dev.id)
		return
	}

	if code != 0 {
		return
	}

	log.Info().Msgf("device '%s' registered, group '%s' proto %d, heartbeat %v",
		dev.id, dev.group, dev.proto, dev.heartbeat)

	for {
		conn.SetReadDeadline(time.Now().Add(dev.heartbeat * 3 / 2))

		typ, data, err = dev.ReadMsg()
		if err != nil {
			if err != io.EOF {
				log.Error().Msgf("read msg from device '%s' fail: %v", dev.id, err)
			}
			return
		}

		log.Debug().Msgf("device msg %s from device %s", proto.MsgTypeName(typ), dev.id)

		handler, ok := DeviceMsgHandlers[typ]
		if !ok {
			log.Error().Msgf("unexpected message '%s' from device '%s'", proto.MsgTypeName(typ), dev.id)
			return
		}

		err = handler(dev, data)
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}
	}
}

func (dev *Device) ReadMsg() (byte, []byte, error) {
	return dev.msg.Read()
}

func (dev *Device) WriteMsg(typ byte, data ...any) error {
	return dev.msg.Write(typ, data...)
}

func (dev *Device) Close(srv *RttyServer) {
	dev.close.Do(func() {
		log.Error().Msgf("device '%s' disconnected", dev.id)
		srv.DelDevice(dev)
		dev.cancel()
		dev.conn.Close()
	})
}

func (dev *Device) ParseRegister(b []byte) error {
	dev.proto = b[0]

	if dev.proto > 4 {
		attrs := utils.ParseTLV(b[1:])
		if attrs == nil {
			return fmt.Errorf("invalid tlv")
		}

		for typ, val := range attrs {
			switch typ {
			case proto.MsgRegAttrHeartbeat:
				dev.heartbeat = time.Duration(val[0]) * time.Second
			case proto.MsgRegAttrDevid:
				dev.id = string(val)
			case proto.MsgRegAttrDescription:
				dev.desc = string(val)
			case proto.MsgRegAttrToken:
				dev.token = string(val)
			case proto.MsgRegAttrGroup:
				dev.group = string(val)
			}
		}
	} else {
		b = b[1:]

		fields := bytes.Split(b, []byte{0})

		if len(fields) < 3 {
			return fmt.Errorf("invalid format")
		}

		dev.id = string(fields[0])
		dev.desc = string(fields[1])
		dev.token = string(fields[2])
	}

	if dev.id == "" {
		return fmt.Errorf("not found device id")
	}

	if len(dev.id) > proto.MaximumDevIDLen {
		return fmt.Errorf("device id too long")
	}

	if len(dev.desc) > proto.MaximumDescLen {
		return fmt.Errorf("device desc too long")
	}

	if len(dev.group) > proto.MaximumGroupLen {
		return fmt.Errorf("device group too long")
	}

	return nil
}

func (dev *Device) Register(srv *RttyServer) byte {
	cfg := &srv.cfg

	if dev.proto < RttyProtoRequired {
		log.Error().Msgf("minimum proto required %d, found %d for device '%s'", RttyProtoRequired, dev.proto, dev.id)
		return devRegErrHookFailed
	}

	if cfg.Token != "" && dev.token != cfg.Token {
		log.Error().Msgf("invalid token for device '%s'", dev.id)
		return devRegErrInvalidToken
	}

	devHookUrl := cfg.DevHookUrl
	if devHookUrl != "" {
		cli := &http.Client{
			Timeout: 3 * time.Second,
		}

		data := fmt.Sprintf(`{"group":"%s", "devid":"%s", "token":"%s"}`, dev.group, dev.id, dev.token)

		resp, err := cli.Post(devHookUrl, "application/json", strings.NewReader(data))
		if err != nil {
			log.Error().Msgf("call device hook url fail for device %s: %v", dev.id, err)
			return devRegErrHookFailed
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Error().Msgf("call device hook url for device '%s', StatusCode: %d", dev.id, resp.StatusCode)
			return devRegErrHookFailed
		}
	}

	if !srv.AddDevice(dev) {
		return devRegErrIdConflicting
	}

	return 0
}

func handleHeartbeatMsg(dev *Device, data []byte) error {
	if !parseHeartbeat(dev, data) {
		return fmt.Errorf("invalid heartbeat msg from device '%s'", dev.id)
	}
	return dev.WriteMsg(proto.MsgTypeHeartbeat)
}

func parseHeartbeat(dev *Device, data []byte) bool {
	if dev.proto > 4 {
		attrs := utils.ParseTLV(data)
		if attrs == nil {
			return false
		}

		for typ, val := range attrs {
			switch typ {
			case proto.MsgHeartbeatAttrUptime:
				dev.uptime = binary.BigEndian.Uint32(val)
			}
		}
	} else {
		if len(data) < 4 {
			return false
		}
		dev.uptime = binary.BigEndian.Uint32(data[:4])
	}

	return true
}

func handleLogoutMsg(dev *Device, data []byte) error {
	sid := string(data[:32])

	if val, loaded := dev.users.LoadAndDelete(sid); loaded {
		user := val.(*User)
		user.Close()
	}

	return nil
}

func handleLoginMsg(dev *Device, data []byte) error {
	sid := string(data[:32])
	code := data[32]

	if val, loaded := dev.pending.LoadAndDelete(sid); loaded {
		user := val.(*User)

		ok := code == 0
		errCode := 0

		if ok {
			log.Debug().Msgf("login session '%s' for device '%s' success", sid, dev.id)
			dev.users.Store(sid, user)
		} else {
			errCode = LoginErrorBusy
			log.Error().Msgf("login session '%s' for device '%s' fail, due to device busy", sid, dev.id)
		}

		if errCode == 0 {
			user.WriteMsg(websocket.TextMessage, []byte(fmt.Appendf(nil, `{"type":"login"}`)))
		} else {
			user.SendCloseMsg(LoginErrorBusy, "device busy")
		}

		user.pending <- ok
	}

	return nil
}

func handleTermDataMsg(dev *Device, data []byte) error {
	sid := string(data[:32])

	if val, ok := dev.users.Load(sid); ok {
		user := val.(*User)
		data[31] = 0
		user.WriteMsg(websocket.BinaryMessage, data[31:])
	}

	return nil
}

func handleFileMsg(dev *Device, data []byte) error {
	sid := string(data[:32])
	typ := data[32]

	if val, ok := dev.users.Load(sid); ok {
		user := val.(*User)

		switch typ {
		case proto.MsgTypeFileSend:
			user.WriteMsg(websocket.TextMessage,
				fmt.Appendf(nil, `{"type":"sendfile", "name": "%s"}`, string(data[33:])))

		case proto.MsgTypeFileRecv:
			user.WriteMsg(websocket.TextMessage, []byte(`{"type":"recvfile"}`))

		case proto.MsgTypeFileData:
			data[32] = 1
			user.WriteMsg(websocket.BinaryMessage, data[32:])

		case proto.MsgTypeFileAck:
			user.WriteMsg(websocket.TextMessage, []byte(`{"type":"fileAck"}`))

		case proto.MsgTypeFileAbort:
			user.WriteMsg(websocket.BinaryMessage, []byte{1})
		}
	}

	return nil
}

func handleHttpMsg(dev *Device, data []byte) error {
	var saddr [18]byte

	copy(saddr[:], data[:18])

	data = data[18:]

	if c, ok := dev.https.Load(saddr); ok {
		c := c.(net.Conn)
		if len(data) == 0 {
			c.Close()
		} else {
			c.Write(data)
		}
	}

	return nil
}

func handleCmdMsg(dev *Device, data []byte) error {
	info := &CommandRespInfo{}

	err := jsoniter.Unmarshal(data, info)
	if err != nil {
		return fmt.Errorf("parse command resp info error: %v", err)
	}

	var attrs map[string]any
	err = jsoniter.Unmarshal(info.Attrs, &attrs)
	if err != nil {
		return fmt.Errorf("parse command resp attrs error: %v", err)
	}

	attrs["devid"] = dev.id

	if val, ok := dev.commands.Load(info.Token); ok {
		req := val.(*CommandReq)
		req.acked = true
		req.c.JSON(http.StatusOK, attrs)
		req.cancel()
	}

	return nil
}
