package server

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhaojh329/rtty-go/proto"
)

func shareAPIRouter(srv *RttyServer) *gin.Engine {
	a := &APIServer{srv: srv}

	r := gin.New()
	r.POST("/api/shares", a.handleCreateShare)
	r.GET("/api/shares", a.handleListShares)
	r.DELETE("/api/shares/:id", a.handleDeleteShare)

	return r
}

func shareAPIRequest(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-user")
	req.Header.Set("X-Rttys-Device-ID", "forged")
	req.Header.Add("X-Rttys-Device-ID", "another-device")
	req.Header.Set("X-Rttys-Group", "forged")
	req.Header.Add("X-Rttys-Group", "another-group")
	req.Header.Set("X-Original-URL", "/forged")

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	return recorder
}

func TestCreateShareHookReceivesTarget(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   string
		target shareInfo
	}{
		{"tcp", `{"kind":"tcp","deviceId":"device","targetIp":"::ffff:192.0.2.10","targetPort":2222}`,
			shareInfo{Kind: "tcp", DeviceID: "device", TargetIP: "192.0.2.10", TargetPort: 2222, IdleSeconds: 60}},
		{"serial", `{"kind":"serial","deviceId":"device","serial":{"port":"/dev/ttyUSB0","baudRate":115200,"dataBits":8,"stopBits":1,"parity":1}}`,
			shareInfo{Kind: "serial", DeviceID: "device", IdleSeconds: 60, Serial: &proto.SerialSettings{Port: "/dev/ttyUSB0", BaudRate: 115200, DataBits: 8, StopBits: 1, Parity: proto.SerialParityOdd}}},
		{"terminal", `{"kind":"terminal","deviceId":"device","idleSeconds":300}`,
			shareInfo{Kind: "terminal", DeviceID: "device", IdleSeconds: 300}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, _, _, _ := testShareServer(t)
			var calls atomic.Int32
			hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "GET" || r.Header.Get("Authorization") != "Bearer test-user" ||
					r.Header.Get("X-Rttys-Hook") != "true" || r.Header.Get("X-Original-Method") != "POST" ||
					r.Header.Get("X-Original-URL") != "/api/shares?group=forged&deviceId=forged" ||
					len(r.Header.Values("X-Rttys-Device-ID")) != 1 || len(r.Header.Values("X-Rttys-Group")) != 1 || r.Header.Get("X-Rttys-Share") != "" {
					t.Error("hook request lost identity/method or retained spoofed context")
				}

				if r.Header.Get("X-Rttys-Device-ID") != tc.target.DeviceID || r.Header.Get("X-Rttys-Group") != tc.target.Group {
					t.Errorf("incorrect authorization device/group: %v", r.Header)
					w.WriteHeader(http.StatusForbidden)
				}
			}))
			defer hook.Close()
			srv.cfg.UserHookUrl = hook.URL

			response := shareAPIRequest(shareAPIRouter(srv), "POST", "/api/shares?group=forged&deviceId=forged", tc.body)
			if response.Code != http.StatusCreated || calls.Load() != 1 {
				t.Fatalf("create: status=%d calls=%d body=%s", response.Code, calls.Load(), response.Body)
			}

			shares := srv.shares.list()
			if len(shares) != 1 {
				t.Fatalf("created shares: %d", len(shares))
			}

			got := shares[0]
			got.ID, got.Host, got.Fingerprint = "", "", ""
			got.Port, got.IdleDeadline = 0, nil
			if !reflect.DeepEqual(got, tc.target) {
				t.Fatalf("unexpected created target: %+v", got)
			}
		})
	}
}

func TestCreateShareDeniedBeforeAllocation(t *testing.T) {
	srv, _, _, port := testShareServer(t)
	var calls atomic.Int32
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("X-Original-Method") != "POST" ||
			r.Header.Get("X-Rttys-Device-ID") != "secret" || r.Header.Get("X-Rttys-Group") != "private" {
			t.Errorf("missing actual device/group: %v", r.Header)
		}
		w.WriteHeader(http.StatusForbidden)
	}))
	defer hook.Close()
	srv.cfg.UserHookUrl = hook.URL

	response := shareAPIRequest(shareAPIRouter(srv), "POST", "/api/shares?group=public&deviceId=device",
		`{"kind":"tcp","group":"private","deviceId":"secret","targetIp":"10.0.0.1","targetPort":22}`)
	if response.Code != http.StatusForbidden || calls.Load() != 1 || len(srv.shares.list()) != 0 {
		t.Fatalf("unauthorized create: status=%d calls=%d", response.Code, calls.Load())
	}

	ln, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port})
	if err != nil {
		t.Fatalf("denied share allocated listener: %v", err)
	}
	ln.Close()
}

func TestCreateShareInvalidRequestSkipsHook(t *testing.T) {
	srv, _, _, _ := testShareServer(t)
	var calls atomic.Int32
	hook := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer hook.Close()
	srv.cfg.UserHookUrl = hook.URL
	router := shareAPIRouter(srv)

	for _, body := range []string{
		`{`,
		`{"kind":"terminal"}`,
		`{"kind":"invalid","deviceId":"device"}`,
		`{"kind":"tcp","deviceId":"device","targetIp":"invalid","targetPort":22}`,
		`{"kind":"serial","deviceId":"device"}`,
		`{"kind":"terminal","deviceId":"device","idleSeconds":-1}`,
	} {
		response := shareAPIRequest(router, "POST", "/api/shares", body)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid request accepted: %d %s", response.Code, body)
		}
	}

	if calls.Load() != 0 || len(srv.shares.list()) != 0 {
		t.Fatal("invalid request reached hook or allocated a share")
	}
}

func TestListAndDeleteSharesUseStoredTarget(t *testing.T) {
	srv, dev, _, _ := testShareServer(t)
	blockedDev := &Device{id: "secret", group: "private", proto: 6, ctx: dev.ctx}
	if !srv.AddDevice(blockedDev) {
		t.Fatal("add private device")
	}

	var shares []*Share

	for _, device := range []*Device{dev, blockedDev} {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		listener.Close()
		srv.cfg.SharePortStart, srv.cfg.SharePortEnd = port, port

		s, _, err := srv.shares.create(shareRequest{Kind: "tcp", DeviceID: device.id, Group: device.group, TargetIP: "127.0.0.1", TargetPort: 22}, "localhost")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(s.close)
		shares = append(shares, s)
	}

	var unavailable atomic.Bool
	var calls atomic.Int32
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		method := r.Header.Get("X-Original-Method")
		if method != "GET" && method != "DELETE" {
			t.Errorf("invalid original method: %q", method)
		}

		if unavailable.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		if r.Header.Get("X-Rttys-Device-ID") != "device" || r.Header.Get("X-Rttys-Group") != "" {
			w.WriteHeader(http.StatusForbidden)
		}
	}))
	defer hook.Close()
	srv.cfg.UserHookUrl = hook.URL
	router := shareAPIRouter(srv)

	response := shareAPIRequest(router, "GET", "/api/shares?group=public", "")
	var visible []shareInfo
	if err := json.Unmarshal(response.Body.Bytes(), &visible); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || len(visible) != 1 || visible[0].ID != shares[0].info.ID || calls.Load() != 2 {
		t.Fatalf("list leaked or omitted shares: status=%d body=%s", response.Code, response.Body)
	}

	response = shareAPIRequest(router, "DELETE", "/api/shares/"+shares[1].info.ID+"?deviceId=device&group=", "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("private share deletion: %d", response.Code)
	}

	select {
	case <-shares[1].done:
		t.Fatal("unauthorized deletion closed share")
	default:
	}

	unavailable.Store(true)
	response = shareAPIRequest(router, "GET", "/api/shares", "")
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != "[]" {
		t.Fatal("hook failure disclosed shares")
	}

	response = shareAPIRequest(router, "DELETE", "/api/shares/"+shares[0].info.ID, "")
	if response.Code != http.StatusForbidden {
		t.Fatal("hook failure allowed deletion")
	}

	unavailable.Store(false)

	response = shareAPIRequest(router, "DELETE", "/api/shares/"+shares[0].info.ID, "")
	if response.Code != http.StatusNoContent {
		t.Fatalf("allowed deletion: %d", response.Code)
	}

	select {
	case <-shares[0].done:
	default:
		t.Fatal("authorized deletion did not close share")
	}

	before := calls.Load()
	response = shareAPIRequest(router, "DELETE", "/api/shares/missing", "")
	if response.Code != http.StatusNotFound || calls.Load() != before {
		t.Fatal("missing share reached authorization without a target")
	}
}

func TestSharesWithoutHook(t *testing.T) {
	srv, _, _, _ := testShareServer(t)
	router := shareAPIRouter(srv)
	response := shareAPIRequest(router, "POST", "/api/shares", `{"kind":"tcp","deviceId":"device","targetIp":"127.0.0.1","targetPort":22}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("create without hook: %d", response.Code)
	}

	response = shareAPIRequest(router, "GET", "/api/shares", "")
	var shares []shareInfo
	if err := json.Unmarshal(response.Body.Bytes(), &shares); err != nil {
		t.Fatal(err)
	}
	if len(shares) != 1 {
		t.Fatal("list without hook")
	}

	response = shareAPIRequest(router, "DELETE", "/api/shares/"+shares[0].ID, "")
	if response.Code != http.StatusNoContent {
		t.Fatal("delete without hook")
	}

	response = shareAPIRequest(router, "GET", "/api/shares", "")
	if !bytes.Equal(bytes.TrimSpace(response.Body.Bytes()), []byte("[]")) {
		t.Fatal("empty list is not []")
	}
}

func TestNonShareHookOverridesSpoofedContext(t *testing.T) {
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.Header.Get("X-Rttys-Device-ID") != "device" || r.Header.Get("X-Rttys-Group") != "lab" ||
			r.Header.Get("X-Rttys-Hook") != "true" || r.Header.Get("X-Original-URL") != "/api/connect/device" {
			t.Error("non-share hook received spoofed share context")
		}
	}))
	defer hook.Close()
	a := &APIServer{srv: New(Config{UserHookUrl: hook.URL})}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/connect/device", nil)
	c.Request.Header.Set("X-Rttys-Device-ID", "forged")
	c.Request.Header.Set("X-Rttys-Group", "forged")
	if !a.callUserHookUrl(c, "device", "lab") {
		t.Fatal("legacy hook denied")
	}
}
