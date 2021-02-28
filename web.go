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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/zhaojh329/rttys/cache"
	"github.com/zhaojh329/rttys/client"
	"github.com/zhaojh329/rttys/utils"
)

type webSession struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type webNewCon struct {
	r *http.Request // First request
	b *bufio.Reader
	c net.Conn
}

type webCon struct {
	dev client.Client
	c   net.Conn
}

type webReq struct {
	addr []byte
	data []byte
	dev  client.Client
}

type webResp struct {
	data []byte
	dev  client.Client
}

var webCons = make(map[string]map[string]*webCon)
var webSessions *cache.Cache

func handleWebReq(req *webReq) {
	dev := req.dev

	if req.data == nil {
		delete(webCons[dev.DeviceID()], string(req.addr))
		return
	}

	dev.WriteMsg(msgTypeWeb, req.data)
}

func handleWebResp(resp *webResp) {
	data := resp.data
	addr := data[:18]
	data = data[18:]

	if len(data) == 0 {
		return
	}

	devcons, ok := webCons[resp.dev.DeviceID()]
	if !ok {
		return
	}

	wc, ok := devcons[string(addr)]
	if !ok {
		return
	}

	c := wc.c

	c.Write(data)
}

func makeWebReqMsg(br *broker, dev client.Client, srcAddr []byte, r *http.Request, hostHeaderRewrite string, destAddr []byte) {
	req := append([]byte{}, srcAddr...)
	req = append(req, destAddr...)
	req = append(req, r.Method...)
	req = append(req, ' ')
	req = append(req, r.RequestURI...)
	req = append(req, ' ')
	req = append(req, "HTTP/1.1\r\n"...)

	for k, v := range r.Header {
		req = append(req, k...)
		req = append(req, ':')
		req = append(req, strings.Join(v, ",")...)
		req = append(req, "\r\n"...)
	}

	req = append(req, "Host:"...)
	req = append(req, hostHeaderRewrite...)
	req = append(req, "\r\n"...)

	req = append(req, "\r\n"...)

	b := make([]byte, 4096)
	n, _ := r.Body.Read(b)
	if n > 0 {
		req = append(req, b[:n]...)
	}

	br.webReq <- &webReq{srcAddr, req, dev}

	for {
		n, err := r.Body.Read(b)
		if n > 0 {
			req := append([]byte{}, srcAddr...)
			req = append(req, destAddr...)
			req = append(req, b[:n]...)
			br.webReq <- &webReq{srcAddr, req, dev}
		}

		if err != nil {
			return
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

func handleWebCon(br *broker, wc *webNewCon) {
	c := wc.c
	r := wc.r

	cookie, err := r.Cookie("rtty-web-devid")
	if err != nil {
		c.Close()
		return
	}
	devid := cookie.Value

	dev, ok := br.devices[devid]
	if !ok {
		c.Close()
		return
	}

	cookie, err = r.Cookie("rtty-web-sid")
	if err != nil {
		c.Close()
		return
	}
	sid := cookie.Value

	var ctx context.Context
	var cancel context.CancelFunc

	if v, ok := webSessions.Get(sid); ok {
		webSessions.Active(sid, 0)
		ctx, cancel = context.WithCancel(v.(*webSession).ctx)
	} else {
		c.Close()
		return
	}

	hostHeaderRewrite := "localhost"
	cookie, err = r.Cookie("rtty-web-destaddr")
	if err == nil {
		hostHeaderRewrite, _ = url.QueryUnescape(cookie.Value)
	}

	destAddr := genDestAddr(hostHeaderRewrite)

	if _, ok := webCons[devid]; !ok {
		webCons[devid] = make(map[string]*webCon)
	}

	srcAddr := tcpAddr2Bytes(c.RemoteAddr().(*net.TCPAddr))

	webCons[devid][string(srcAddr)] = &webCon{dev, c}

	go func() {
		defer func() {
			br.webReq <- &webReq{srcAddr, nil, dev}
			cancel()
		}()

		makeWebReqMsg(br, dev, srcAddr, r, hostHeaderRewrite, destAddr)

		for {
			r, err := http.ReadRequest(wc.b)
			if err != nil {
				return
			}
			makeWebReqMsg(br, dev, srcAddr, r, hostHeaderRewrite, destAddr)
		}
	}()

	go func() {
		<-ctx.Done()
		c.Close()
	}()
}

func listenDeviceWeb(br *broker) {
	cfg := br.cfg

	addr, err := net.ResolveTCPAddr("tcp", cfg.AddrWeb)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	cfg.WebPort = addr.Port

	webSessions = cache.New(30*time.Minute, 5*time.Second)

	log.Info().Msgf("Listen dev web on: %s", cfg.AddrWeb)

	ln, err := net.Listen("tcp", cfg.AddrWeb)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	go func() {
		defer ln.Close()

		for {
			c, err := ln.Accept()
			if err != nil {
				log.Error().Msg(err.Error())
				continue
			}

			go func() {
				b := bufio.NewReader(c)

				r, err := http.ReadRequest(b)
				if err != nil {
					c.Close()
					return
				}

				br.webCon <- &webNewCon{r, b, c}
			}()
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
