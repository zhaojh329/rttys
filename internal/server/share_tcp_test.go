package server

import (
	"bytes"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/zhaojh329/rtty-go/proto"
)

func TestTCPShareCloseWakesBlockedForward(t *testing.T) {
	srv, dev, clientConn, port := testShareServer(t)
	reader := proto.NewMsgReaderWriter(proto.RoleRtty, clientConn)
	full := make(chan struct{})
	go func() {
		total := 0
		for {
			typ, data, err := reader.Read()
			if err != nil {
				return
			}
			if typ != proto.MsgTypeTCP {
				continue
			}

			switch data[32] {
			case proto.TCPTypeOpen:
				handleDeviceTCPMsg(dev, append(append([]byte(nil), data[:32]...), proto.TCPTypeOpenResult, proto.TCPOpenOK))
			case proto.TCPTypeData:
				total += len(data) - 33
				if total == proto.TCPWindowSize {
					close(full)
				}
			}
		}
	}()

	s, _, err := srv.shares.create(shareRequest{Kind: "tcp", DeviceID: "device", TargetIP: "127.0.0.1", TargetPort: 22}, "localhost")
	if err != nil {
		t.Fatal(err)
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	go conn.Write(bytes.Repeat([]byte("x"), proto.TCPWindowSize+4096))
	select {
	case <-full:
	case <-time.After(3 * time.Second):
		t.Fatal("forward did not reach its send window")
	}

	s.close()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		count := 0
		dev.tcpForwards.Range(func(_, _ any) bool { count++; return true })
		if count == 0 && s.snapshot().Connections == 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("closing a share left a forward waiting for ACK")
}

func TestTCPShareOrderlyCloseDoesNotAbortPeer(t *testing.T) {
	srv, dev, clientConn, port := testShareServer(t)
	reader := proto.NewMsgReaderWriter(proto.RoleRtty, clientConn)
	aborted := make(chan struct{}, 1)
	received := make(chan []byte, 1)
	go func() {
		var payload []byte
		for {
			typ, data, err := reader.Read()
			if err != nil {
				return
			}
			if typ != proto.MsgTypeTCP {
				continue
			}

			switch data[32] {
			case proto.TCPTypeOpen:
				id := append([]byte(nil), data[:32]...)
				handleDeviceTCPMsg(dev, append(append([]byte(nil), id...), proto.TCPTypeOpenResult, proto.TCPOpenOK))
				handleDeviceTCPMsg(dev, append(id, proto.TCPTypeCloseWrite))
			case proto.TCPTypeData:
				payload = append(payload, data[33:]...)
			case proto.TCPTypeCloseWrite:
				received <- payload
			case proto.TCPTypeClose:
				aborted <- struct{}{}
			}
		}
	}()

	s, _, err := srv.shares.create(shareRequest{Kind: "tcp", DeviceID: "device", TargetIP: "127.0.0.1", TargetPort: 22}, "localhost")
	if err != nil {
		t.Fatal(err)
	}

	raw, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
	if err != nil {
		t.Fatal(err)
	}

	conn := raw.(*net.TCPConn)
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))

	var byteBuf [1]byte
	if _, err := conn.Read(byteBuf[:]); err != io.EOF {
		t.Fatalf("target half close: %v", err)
	}

	payload := bytes.Repeat([]byte("tail"), 4096)
	if _, err := conn.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := conn.CloseWrite(); err != nil {
		t.Fatal(err)
	}

	select {
	case data := <-received:
		if !bytes.Equal(data, payload) {
			t.Fatal("tail data not delivered before half close")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("half close not forwarded")
	}

	deadline := time.Now().Add(time.Second)
	for s.snapshot().Connections != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if s.snapshot().Connections != 0 {
		t.Fatal("completed connection still active")
	}

	select {
	case <-aborted:
		t.Fatal("orderly half closes followed by forced close before peer drained its input")
	default:
	}
}
