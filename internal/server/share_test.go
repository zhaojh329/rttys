package server

import (
	"context"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/zhaojh329/rtty-go/proto"
	"golang.org/x/crypto/ssh"
)

func testShareServer(t *testing.T) (*RttyServer, *Device, net.Conn, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	srv := New(Config{ShareBindHost: "127.0.0.1", SharePortStart: port,
		SharePortEnd: port, ShareHostKey: t.TempDir() + "/host_key"})
	serverConn, clientConn := net.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	dev := &Device{id: "device", proto: 6, ctx: ctx, cancel: cancel, conn: serverConn,
		msg: proto.NewMsgReaderWriter(proto.RoleRttys, serverConn)}
	if !srv.AddDevice(dev) {
		t.Fatal("add device")
	}
	t.Cleanup(func() {
		cancel()
		srv.shares.closeDevice(dev)
		serverConn.Close()
		clientConn.Close()
	})
	return srv, dev, clientConn, port
}

func TestSSHShareInteractiveAndExpiry(t *testing.T) {
	srv, dev, clientConn, port := testShareServer(t)
	reader := proto.NewMsgReaderWriter(proto.RoleRtty, clientConn)
	go func() {
		for {
			typ, data, err := reader.Read()
			if err != nil {
				return
			}
			switch typ {
			case proto.MsgTypeLogin:
				handleLoginMsg(dev, append(append([]byte(nil), data...), 0))
			case proto.MsgTypeTermData:
				handleTermDataMsg(dev, data)
			}
		}
	}()
	s, password, err := srv.shares.create(shareRequest{Kind: "terminal", DeviceID: "device"}, "localhost")
	if err != nil {
		t.Fatal(err)
	}
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	bad := &ssh.ClientConfig{User: "share", Auth: []ssh.AuthMethod{ssh.Password("wrong")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: 2 * time.Second}
	if c, err := ssh.Dial("tcp", addr, bad); err == nil {
		c.Close()
		t.Fatal("wrong password accepted")
	}
	config := &ssh.ClientConfig{User: "share", Auth: []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: 2 * time.Second}
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		t.Fatal(err)
	}
	session, err := client.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := session.RequestPty("xterm", 80, 24, ssh.TerminalModes{}); err != nil {
		t.Fatal(err)
	}
	if err := session.Shell(); err != nil {
		t.Fatal(err)
	}
	if _, err := stdin.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		buf := make([]byte, 5)
		_, err := io.ReadFull(stdout, buf)
		if err == nil && string(buf) != "hello" {
			err = io.ErrUnexpectedEOF
		}
		result <- err
	}()
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("SSH terminal data timed out")
	}
	s.mu.Lock()
	s.info.IdleSeconds = 1
	s.mu.Unlock()
	session.Close()
	client.Close()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && len(srv.shares.list()) != 0 {
		time.Sleep(20 * time.Millisecond)
	}
	if len(srv.shares.list()) != 0 {
		t.Fatal("idle SSH share remained open")
	}
}

func TestTCPShareForwardsAndClosesWithDevice(t *testing.T) {
	srv, dev, clientConn, port := testShareServer(t)
	reader := proto.NewMsgReaderWriter(proto.RoleRtty, clientConn)
	go func() {
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
				handleDeviceTCPMsg(dev, append(append([]byte(nil), data[:32]...), append([]byte{proto.TCPTypeData}, data[33:]...)...))
			case proto.TCPTypeCloseWrite:
				handleDeviceTCPMsg(dev, append(append([]byte(nil), data[:32]...), proto.TCPTypeCloseWrite))
			}
		}
	}()
	s, _, err := srv.shares.create(shareRequest{Kind: "tcp", DeviceID: "device",
		TargetIP: "127.0.0.1", TargetPort: 22}, "localhost")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := conn.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 5)
	if _, err := io.ReadFull(conn, buf); err != nil || string(buf) != "hello" {
		t.Fatalf("TCP echo: %q, %v", buf, err)
	}
	if s.snapshot().Connections != 1 {
		t.Fatal("active TCP connection was not counted")
	}
	srv.shares.closeDevice(dev)
	if len(srv.shares.list()) != 0 {
		t.Fatal("share remained after device disconnected")
	}
	if _, err := conn.Read(buf); err == nil {
		t.Fatal("active TCP connection remained after device disconnect")
	}
}

func TestSerialSSHShareUsesSelectedSettings(t *testing.T) {
	srv, dev, clientConn, port := testShareServer(t)
	reader := proto.NewMsgReaderWriter(proto.RoleRtty, clientConn)
	settings := proto.SerialSettings{Port: "ttyUSB0", BaudRate: 115200, DataBits: 8,
		StopBits: 1, Parity: proto.SerialParityNone}
	opened := make(chan proto.SerialSettings, 1)
	go func() {
		for {
			typ, data, err := reader.Read()
			if err != nil {
				return
			}
			switch typ {
			case proto.MsgTypeSerialOpen:
				got, err := proto.ParseSerialSettings(data[32:])
				if err == nil {
					opened <- got
				}
				handleSerialOpenMsg(dev, append(append([]byte(nil), data[:32]...), proto.SerialOK))
			case proto.MsgTypeTermData:
				handleTermDataMsg(dev, data)
			}
		}
	}()
	_, password, err := srv.shares.create(shareRequest{Kind: "serial", DeviceID: "device", Serial: &settings}, "localhost")
	if err != nil {
		t.Fatal(err)
	}
	config := &ssh.ClientConfig{User: "share", Auth: []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), Timeout: 2 * time.Second}
	client, err := ssh.Dial("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), config)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	stdin, err := session.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Shell(); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-opened:
		if got != settings {
			t.Fatalf("serial settings: got %+v, want %+v", got, settings)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("serial open timed out")
	}
	if _, err := stdin.Write([]byte("serial-data")); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		buf := make([]byte, len("serial-data"))
		_, err := io.ReadFull(stdout, buf)
		if err == nil && string(buf) != "serial-data" {
			err = io.ErrUnexpectedEOF
		}
		result <- err
	}()
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("serial data timed out")
	}
}

func TestShareHostKeyPersists(t *testing.T) {
	path := t.TempDir() + "/host_key"
	first, err := loadShareHostKey(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := loadShareHostKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if ssh.FingerprintSHA256(first.PublicKey()) != ssh.FingerprintSHA256(second.PublicKey()) {
		t.Fatal("SSH host key changed between loads")
	}
}

func TestLateTerminalOpenClosesClientSession(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	dev := &Device{msg: proto.NewMsgReaderWriter(proto.RoleRttys, serverConn)}
	result := make(chan error, 1)
	id := "0123456789abcdef0123456789abcdef"
	go func() {
		result <- handleLoginMsg(dev, append([]byte(id), 0))
	}()
	clientConn.SetReadDeadline(time.Now().Add(time.Second))
	typ, data, err := proto.NewMsgReaderWriter(proto.RoleRtty, clientConn).Read()
	if err != nil || typ != proto.MsgTypeLogout || string(data) != id {
		t.Fatalf("late terminal response: type=%d data=%q error=%v", typ, data, err)
	}
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}

func TestShareConnectionLimit(t *testing.T) {
	s := &Share{conns: make(map[net.Conn]bool)}
	var peers []net.Conn
	for range maxShareConnections {
		server, client := net.Pipe()
		peers = append(peers, server, client)
		if !s.track(server) {
			t.Fatal("rejected connection below limit")
		}
	}
	defer func() {
		for _, conn := range peers {
			conn.Close()
		}
	}()
	extra, peer := net.Pipe()
	defer extra.Close()
	defer peer.Close()
	if s.track(extra) {
		t.Fatal("accepted connection above limit")
	}
	s.release(peers[0])
	if !s.track(extra) {
		t.Fatal("did not accept connection after slot released")
	}
}
