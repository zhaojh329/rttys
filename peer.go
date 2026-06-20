/* SPDX-License-Identifier: MIT */
/*
 * Author: Jianhui Zhao <zhaojh329@gmail.com>
 */

package main

import (
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/rs/zerolog/log"
)

const (
	peerCapabilitySignal = uint32(1 << iota)
)

type PeerSession struct {
	id           string
	dev          *Device
	user         *User
	createdAt    time.Time
	lastDeviceTx atomic.Int64
}

func newPeerSession(id string, dev *Device, user *User) *PeerSession {
	return &PeerSession{
		id:        id,
		dev:       dev,
		user:      user,
		createdAt: time.Now(),
	}
}

func (srv *RttyServer) StorePeerSession(s *PeerSession) {
	srv.peerSessions.Store(s.id, s)
}

func (srv *RttyServer) GetPeerSession(id string) *PeerSession {
	if v, ok := srv.peerSessions.Load(id); ok {
		return v.(*PeerSession)
	}

	return nil
}

func (srv *RttyServer) DeletePeerSession(id string) {
	srv.peerSessions.Delete(id)
}

func (s *PeerSession) SendToBrowser(signalType byte, payload []byte) error {
	msg := struct {
		Type       string          `json:"type"`
		SignalType byte            `json:"signalType"`
		Payload    json.RawMessage `json:"payload"`
	}{
		Type:       "peerSignal",
		SignalType: signalType,
		Payload:    append(json.RawMessage(nil), payload...),
	}

	data, err := jsoniter.Marshal(msg)
	if err != nil {
		return err
	}

	return s.user.WriteMsg(websocket.TextMessage, data)
}

func handlePeerSignalMsg(dev *Device, data []byte) error {
	sid := string(data[:32])
	signalType := data[32]

	session := dev.srv.GetPeerSession(sid)
	if session == nil {
		log.Warn().Msgf("peer session '%s' for device '%s' not found", sid, dev.id)
		return nil
	}

	session.lastDeviceTx.Store(time.Now().Unix())
	if err := session.SendToBrowser(signalType, data[33:]); err != nil {
		return err
	}
	log.Debug().Msgf("peer signal from device '%s': sid=%s type=%d payload=%dB", dev.id, sid, signalType, len(data)-33)

	return nil
}
