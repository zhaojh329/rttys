/* SPDX-License-Identifier: MIT */
package server

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/zhaojh329/rtty-go/proto"
	"github.com/zhaojh329/rtty-go/tcpforward"
	"github.com/zhaojh329/rttys/v5/internal/utils"
)

type tcpForward struct {
	share   *Share
	id      string
	ready   chan byte
	forward *tcpforward.Forwarder
	done    chan struct{}
	once    sync.Once
}

func (dev *Device) sendTCP(id string, op byte, body ...any) error {
	args := append([]any{id, op}, body...)
	return dev.WriteMsg(proto.MsgTypeTCP, args...)
}

func handleDeviceTCPMsg(dev *Device, data []byte) error {
	if len(data) < 33 {
		return fmt.Errorf("short TCP message")
	}

	id, op := string(data[:32]), data[32]
	v, ok := dev.tcpForwards.Load(id)
	if !ok {
		if op == proto.TCPTypeOpenResult && len(data) == 34 && data[33] == proto.TCPOpenOK {
			dev.sendTCP(id, proto.TCPTypeClose)
		}
		return nil
	}

	f := v.(*tcpForward)
	switch op {
	case proto.TCPTypeOpenResult:
		if len(data) != 34 || data[33] > proto.TCPOpenFailed {
			return fmt.Errorf("invalid TCP open result")
		}
		select {
		case f.ready <- data[33]:
		default:
		}
	case proto.TCPTypeClose:
		if len(data) != 33 {
			return fmt.Errorf("invalid TCP close")
		}
		f.close(false)
	default:
		if err := f.forward.Handle(op, data[33:]); err != nil {
			f.close(true)
		}
	}

	return nil
}

func (f *tcpForward) close(notify bool) {
	f.once.Do(func() {
		close(f.done)
		f.forward.Close()
		f.share.dev.tcpForwards.CompareAndDelete(f.id, f)
		if notify {
			f.share.dev.sendTCP(f.id, proto.TCPTypeClose)
		}
	})
}

func (s *Share) serveTCP(conn net.Conn) {
	defer s.release(conn)

	f := &tcpForward{share: s, id: utils.GenUniqueID(), ready: make(chan byte, 1),
		done: make(chan struct{})}
	f.forward = tcpforward.New(conn, func(op byte, data []byte) error {
		return s.dev.sendTCP(f.id, op, data)
	})

	notify := true
	defer func() { f.close(notify) }()

	s.dev.tcpForwards.Store(f.id, f)
	var target [6]byte
	copy(target[:4], s.ip.To4())
	binary.BigEndian.PutUint16(target[4:], uint16(s.info.TargetPort))
	if err := s.dev.sendTCP(f.id, proto.TCPTypeOpen, target[:]); err != nil {
		return
	}

	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case code := <-f.ready:
		if code != proto.TCPOpenOK {
			return
		}
	case <-timer.C:
		return
	case <-f.done:
		return
	case <-s.done:
		return
	case <-s.dev.ctx.Done():
		return
	}

	if !s.activate(conn) {
		return
	}

	go func() {
		select {
		case <-s.done:
			f.close(true)
		case <-s.dev.ctx.Done():
			f.close(false)
		case <-f.done:
		}
	}()

	notify = f.forward.Run() != nil
}
