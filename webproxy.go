package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"rttys/cache"
	"rttys/client"
	"rttys/utils"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type webSession struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type webResp struct {
	data []byte
	dev  client.Client
}

var webCons sync.Map
var webSessions *cache.Cache

func handleWebResp(resp *webResp) {
	data := resp.data
	addr := data[:18]
	data = data[18:]

	if len(data) == 0 {
		return
	}

	if cons, ok := webCons.Load(resp.dev.DeviceID()); ok {
		if c, ok := cons.(*sync.Map).Load(string(addr)); ok {
			c.(net.Conn).Write(data)
		}
	}
}

func genDestAddr(addr string) []byte {
	destIP, destPort, err := webReqVaildAddr(addr)
	if err != nil {
		return nil
	}

	b := make([]byte, 6)
	copy(b, destIP)

	binary.BigEndian.PutUint16(b[4:], destPort)

	return b
}

func tcpAddr2Bytes(addr *net.TCPAddr) []byte {
	b := make([]byte, 18)

	binary.BigEndian.PutUint16(b[:2], uint16(addr.Port))

	copy(b[2:], addr.IP)

	return b
}

type RttyWebWriter struct {
	destAddr          []byte
	srcAddr           []byte
	hostHeaderRewrite string
	dev               client.Client
}

func (rw *RttyWebWriter) Write(p []byte) (n int, err error) {
	msg := append([]byte{}, rw.srcAddr...)
	msg = append(msg, rw.destAddr...)
	msg = append(msg, p...)

	dev := rw.dev.(*device)

	dev.WriteMsg(msgTypeWeb, msg)

	return len(p), nil
}

func (rw *RttyWebWriter) WriteRequest(req *http.Request) {
	req.Host = rw.hostHeaderRewrite
	req.Write(rw)
}

func webProxy(brk *broker, c net.Conn) {
	defer c.Close()

	br := bufio.NewReader(c)

	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}

	cookie, err := req.Cookie("rtty-web-devid")
	if err != nil {
		return
	}
	devid := cookie.Value

	dev, ok := brk.devices[devid]
	if !ok {
		return
	}

	cookie, err = req.Cookie("rtty-web-sid")
	if err != nil {
		return
	}
	sid := cookie.Value

	var ctx context.Context
	var cancel context.CancelFunc

	if v, ok := webSessions.Get(sid); ok {
		webSessions.Active(sid, 0)
		ctx, cancel = context.WithCancel(v.(*webSession).ctx)
	} else {
		return
	}

	hostHeaderRewrite := "localhost"
	cookie, err = req.Cookie("rtty-web-destaddr")
	if err == nil {
		hostHeaderRewrite, _ = url.QueryUnescape(cookie.Value)
	}

	destAddr := genDestAddr(hostHeaderRewrite)
	srcAddr := tcpAddr2Bytes(c.RemoteAddr().(*net.TCPAddr))

	if cons, _ := webCons.LoadOrStore(devid, &sync.Map{}); true {
		cons := cons.(*sync.Map)
		cons.Store(string(srcAddr), c)
	}

	rw := &RttyWebWriter{destAddr, srcAddr, hostHeaderRewrite, dev}

	req.Host = hostHeaderRewrite
	rw.WriteRequest(req)

	go func() {
		<-ctx.Done()

		// needed, for canceled by new proxy in the same web browser
		c.Close()
	}()

	defer func() {
		cons, ok := webCons.Load(devid)
		if ok {
			cons := cons.(*sync.Map)
			cons.Delete(string(srcAddr))
		}
		cancel()
	}()

	for {
		req, err := http.ReadRequest(br)
		if err != nil {
			return
		}

		webSessions.Active(sid, 0)

		rw.WriteRequest(req)
	}
}

func listenDeviceWeb(brk *broker) {
	cfg := brk.cfg

	webSessions = cache.New(10*time.Minute, 5*time.Second)

	if cfg.AddrWeb != "" {
		addr, err := net.ResolveTCPAddr("tcp", cfg.AddrWeb)
		if err != nil {
			log.Warn().Msg("invalid web proxy addr: " + err.Error())
		} else {
			cfg.WebPort = addr.Port
		}
	}

	if cfg.WebPort == 0 {
		log.Info().Msg("Automatically select an available port for web proxy")
	}

	ln, err := net.Listen("tcp", cfg.AddrWeb)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	cfg.WebPort = ln.Addr().(*net.TCPAddr).Port

	log.Info().Msgf("Listen web proxy on: %s", ln.Addr().(*net.TCPAddr))

	go func() {
		defer ln.Close()

		for {
			c, err := ln.Accept()
			if err != nil {
				log.Error().Msg(err.Error())
				continue
			}

			go webProxy(brk, c)
		}
	}()
}

func webReqVaildAddr(addr string) (net.IP, uint16, error) {
	ips, ports, err := net.SplitHostPort(addr)
	if err != nil {
		ips = addr
		ports = "80"
	}

	ip := net.ParseIP(ips)
	if ip == nil {
		return nil, 0, errors.New("invalid IPv4 Addr")
	}

	ip = ip.To4()
	if ip == nil {
		return nil, 0, errors.New("invalid IPv4 Addr")
	}

	port, _ := strconv.Atoi(ports)

	return ip, uint16(port), nil
}

func webReqRedirect(br *broker, c *gin.Context) {
	cfg := br.cfg
	devid := c.Param("devid")
	addr := c.Param("addr")
	path := c.Param("path")

	_, _, err := webReqVaildAddr(addr)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	_, ok := br.devices[devid]
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}

	location := cfg.WebRedirURL

	if location == "" {
		host, _, err := net.SplitHostPort(c.Request.Host)
		if err != nil {
			host = c.Request.Host
		}
		location = "http://" + host
		if cfg.WebPort != 80 {
			location += fmt.Sprintf(":%d", cfg.WebPort)
		}
	}

	location += path

	location += fmt.Sprintf("?_=%d", time.Now().Unix())

	sid, err := c.Cookie("rtty-web-sid")
	if err == nil {
		if v, ok := webSessions.Get(sid); ok {
			v.(*webSession).cancel()
			webSessions.Del(sid)
		}
	}

	sid = utils.GenUniqueID("web")

	ctx, cancel := context.WithCancel(context.Background())
	webSessions.Set(sid, &webSession{ctx, cancel}, 0)

	c.SetCookie("rtty-web-sid", sid, 0, "", "", false, true)
	c.SetCookie("rtty-web-devid", devid, 0, "", "", false, true)
	c.SetCookie("rtty-web-destaddr", addr, 0, "", "", false, true)

	c.Redirect(http.StatusFound, location)
}
