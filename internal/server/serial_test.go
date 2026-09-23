package server

import (
	"context"
	"net"
	"net/http/httptest"
	"reflect"
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
	response = append(response, []byte(`["COM3","COM4"]`)...)
	if err := handleSerialPortsMsg(dev, response); err != nil {
		t.Fatal(err)
	}

	res := <-done
	if res.err != nil || !reflect.DeepEqual(res.ports, []string{"COM3", "COM4"}) {
		t.Fatalf("ports=%v error=%v", res.ports, res.err)
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
