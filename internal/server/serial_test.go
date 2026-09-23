package server

import (
	"bytes"
	"context"
	"net"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhaojh329/rtty-go/proto"
)

func TestParseSerialSettingsParity(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  proto.SerialParity
		valid bool
	}{
		{"none", proto.SerialParityNone, true},
		{"odd", proto.SerialParityOdd, true},
		{"even", proto.SerialParityEven, true},
		{"mark", 0, false},
		{"", 0, false},
	} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/?port=COM3&baudRate=115200&dataBits=8&stopBits=1&parity="+tc.input, nil)

		settings, err := parseSerialSettings(c)
		if (err == nil) != tc.valid || (tc.valid && settings.Parity != tc.want) {
			t.Fatalf("parity %q: settings=%+v error=%v", tc.input, settings, err)
		}
	}
}

func TestListSerialPortsMatchesRequest(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	dev := &Device{ctx: context.Background(), msg: proto.NewMsgReaderWriter(proto.RoleRttys, serverConn)}
	client := proto.NewMsgReaderWriter(proto.RoleRtty, clientConn)
	type result struct {
		ports []string
		err   error
	}
	done := make(chan result, 1)
	go func() {
		ports, err := dev.ListSerialPorts(context.Background())
		done <- result{ports, err}
	}()

	typ, id, err := client.Read()
	if err != nil || typ != proto.MsgTypeSerialPorts || len(id) != 32 {
		t.Fatalf("serial query: type=%d id=%q error=%v", typ, id, err)
	}
	response := append(append([]byte{}, id...), proto.SerialOK)
	response = append(response, []byte{0, 0, 4, 'C', 'O', 'M', '3', 0, 0, 4, 'C', 'O', 'M', '4'}...)
	if err := handleSerialPortsMsg(dev, response); err != nil {
		t.Fatal(err)
	}

	res := <-done
	if res.err != nil || !reflect.DeepEqual(res.ports, []string{"COM3", "COM4"}) {
		t.Fatalf("ports=%v error=%v", res.ports, res.err)
	}
}

func TestSerialPortsTLVResponse(t *testing.T) {
	for _, tc := range []struct {
		name    string
		code    byte
		payload []byte
		want    []string
		wantErr bool
	}{
		{name: "empty"},
		{name: "unknown attribute", payload: []byte{99, 0, 1, 'x', 0, 0, 4, 'C', 'O', 'M', '3'}, want: []string{"COM3"}},
		{name: "big endian length", payload: append([]byte{0, 1, 0}, bytes.Repeat([]byte{'x'}, 256)...), want: []string{strings.Repeat("x", 256)}},
		{name: "short header", payload: []byte{0, 0}, wantErr: true},
		{name: "short value", payload: []byte{0, 0, 4, 'C'}, wantErr: true},
		{name: "trailing byte", payload: []byte{0, 0, 1, 'x', 0}, wantErr: true},
		{name: "json rejected", payload: []byte(`[]`), wantErr: true},
		{name: "query failure", code: proto.SerialFailed, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dev := &Device{}
			id := "0123456789abcdef0123456789abcdef"
			result := make(chan serialPortsResult, 1)
			dev.serialRequests.Store(id, result)
			response := append([]byte(id), tc.code)
			response = append(response, tc.payload...)

			if err := handleSerialPortsMsg(dev, response); err != nil {
				t.Fatal(err)
			}

			res := <-result
			if (res.err != nil) != tc.wantErr {
				t.Fatalf("error=%v, want error=%v", res.err, tc.wantErr)
			}

			if !tc.wantErr && !reflect.DeepEqual(res.ports, tc.want) {
				t.Fatalf("ports=%v, want %v", res.ports, tc.want)
			}
		})
	}
}

func TestLateSerialOpenClosesClientSession(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	dev := &Device{msg: proto.NewMsgReaderWriter(proto.RoleRttys, serverConn)}
	client := proto.NewMsgReaderWriter(proto.RoleRtty, clientConn)
	id := "0123456789abcdef0123456789abcdef"
	done := make(chan error, 1)
	go func() { done <- handleSerialOpenMsg(dev, append([]byte(id), proto.SerialOK)) }()

	typ, payload, err := client.Read()
	if err != nil || typ != proto.MsgTypeLogout || string(payload) != id {
		t.Fatalf("late open cleanup: type=%d data=%q error=%v", typ, payload, err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
