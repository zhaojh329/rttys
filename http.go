/*
 * MIT License
 *
 * Copyright (c) 2019 Jianhui Zhao <zhaojh329@gmail.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

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
	"sync/atomic"
	"time"

	"rttys/utils"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/valyala/bytebufferpool"
)

type HttpProxySession struct {
	expire atomic.Int64
	ctx    context.Context
	cancel context.CancelFunc
}

var httpProxySessions = sync.Map{}

const httpProxySessionsExpire = 15 * time.Minute

func (ses *HttpProxySession) Expire() {
	ses.expire.Store(time.Now().Add(httpProxySessionsExpire).Unix())
}

func (srv *RttyServer) ListenHttpProxy() {
	cfg := &srv.cfg

	if cfg.AddrHttpProxy != "" {
		addr, err := net.ResolveTCPAddr("tcp", cfg.AddrHttpProxy)
		if err != nil {
			log.Warn().Msg("invalid http proxy addr: " + err.Error())
		} else {
			srv.httpProxyPort = addr.Port
		}
	}

	ln, err := net.Listen("tcp", cfg.AddrHttpProxy)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer ln.Close()

	srv.httpProxyPort = ln.Addr().(*net.TCPAddr).Port

	log.Info().Msgf("Listen http proxy on: %s", ln.Addr().(*net.TCPAddr))

	go httpProxySessionsClean()

	for {
		c, err := ln.Accept()
		if err != nil {
			log.Error().Msg(err.Error())
			continue
		}

		go doHttpProxy(srv, c)
	}
}

func httpProxySessionsClean() {
	for {
		time.Sleep(time.Second * 30)

		httpProxySessions.Range(func(key, value any) bool {
			ses := value.(*HttpProxySession)
			if time.Now().Unix() > ses.expire.Load() {
				log.Debug().Msgf("Http proxy session '%s' expired", key)
				ses.cancel()
				httpProxySessions.Delete(key)
			}
			return true
		})
	}
}

func doHttpProxy(srv *RttyServer, c net.Conn) {
	defer logPanic()
	defer c.Close()

	br := bufio.NewReader(c)

	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}

	cookie, err := req.Cookie("rtty-http-devid")
	if err != nil {
		log.Debug().Msg(`not found cookie "rtty-http-devid"`)
		return
	}
	devid := cookie.Value

	group := ""

	cookie, err = req.Cookie("rtty-http-group")
	if err == nil {
		group = cookie.Value
	}

	dev := srv.GetDevice(group, devid)
	if dev == nil {
		log.Debug().Msgf(`device "%s" offline`, devid)
		return
	}

	cookie, err = req.Cookie("rtty-http-sid")
	if err != nil {
		log.Debug().Msgf(`not found cookie "rtty-http-sid", devid "%s"`, devid)
		return
	}
	sid := cookie.Value

	https := false
	cookie, _ = req.Cookie("rtty-http-proto")
	if cookie != nil && cookie.Value == "https" {
		https = true
	}

	hostHeaderRewrite := "localhost"
	cookie, err = req.Cookie("rtty-http-destaddr")
	if err == nil {
		hostHeaderRewrite, _ = url.QueryUnescape(cookie.Value)
	}

	destAddr := genDestAddr(hostHeaderRewrite)
	srcAddr := tcpAddr2Bytes(c.RemoteAddr().(*net.TCPAddr))

	sesVal, ok := httpProxySessions.Load(sid)
	if !ok {
		log.Debug().Msgf(`not found httpProxySession "%s", devid "%s"`, sid, devid)
		return
	}

	ses := sesVal.(*HttpProxySession)

	ctx, cancel := context.WithCancel(ses.ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		c.Close()
		log.Debug().Msgf("http proxy conn closed, devid: %s, https: %v, destaddr: %s", devid, https, hostHeaderRewrite)
		dev.https.Delete(string(srcAddr))
	}()

	log.Debug().Msgf("new http proxy conn, devid: %s, https: %v, destaddr: %s", devid, https, hostHeaderRewrite)

	dev.https.Store(string(srcAddr), c)

	hpw := &HttpProxyWriter{destAddr, srcAddr, hostHeaderRewrite, dev, https}

	req.Host = hostHeaderRewrite
	hpw.WriteRequest(req)

	if req.Header.Get("Upgrade") == "websocket" {
		b := make([]byte, 4096)

		for {
			n, err := c.Read(b)
			if err != nil {
				return
			}
			sendHttpReq(dev, https, srcAddr, destAddr, b[:n])
			ses.Expire()
		}
	} else {
		for {
			req, err := http.ReadRequest(br)
			if err != nil {
				return
			}
			hpw.WriteRequest(req)
			ses.Expire()
		}
	}
}

func httpProxyRedirect(srv *RttyServer, c *gin.Context, group string) {
	cfg := &srv.cfg
	devid := c.Param("devid")
	proto := c.Param("proto")
	addr := c.Param("addr")
	rawPath := c.Param("path")

	log.Debug().Msgf("httpProxyRedirect devid: %s, proto: %s, addr: %s, path: %s", devid, proto, addr, rawPath)

	_, _, err := httpProxyVaildAddr(addr)
	if err != nil {
		log.Debug().Msgf("invalid addr: %s", addr)
		c.Status(http.StatusBadRequest)
		return
	}

	path, err := url.Parse(rawPath)
	if err != nil {
		log.Debug().Msgf("invalid path: %s", rawPath)
		c.Status(http.StatusBadRequest)
		return
	}

	dev := srv.GetDevice(group, devid)
	if dev == nil {
		c.Redirect(http.StatusFound, "/error/offline")
		return
	}

	location := c.Request.Header.Get("HttpProxyRedir")
	if location == "" {
		location = cfg.HttpProxyRedirURL
		if location != "" {
			log.Debug().Msgf("use HttpProxyRedirURL from config: %s, devid: %s", location, devid)
		}
	} else {
		log.Debug().Msgf("use HttpProxyRedir from HTTP header: %s, devid: %s", location, devid)
	}

	if location == "" {
		host, _, err := net.SplitHostPort(c.Request.Host)
		if err != nil {
			host = c.Request.Host
		}

		location = "http://" + host

		if srv.httpProxyPort != 80 {
			location += fmt.Sprintf(":%d", srv.httpProxyPort)
		}
	}

	location += path.Path

	location += fmt.Sprintf("?_=%d", time.Now().Unix())

	if path.RawQuery != "" {
		location += "&" + path.RawQuery
	}

	sid, err := c.Cookie("rtty-http-sid")
	if err == nil {
		if v, loaded := httpProxySessions.LoadAndDelete(sid); loaded {
			s := v.(*HttpProxySession)
			s.cancel()
			log.Debug().Msgf(`del old httpProxySession "%s" for device "%s"`, sid, devid)
		}
	}

	sid = utils.GenUniqueID()

	ctx, cancel := context.WithCancel(dev.ctx)

	ses := &HttpProxySession{
		ctx:    ctx,
		cancel: cancel,
	}
	ses.Expire()
	httpProxySessions.Store(sid, ses)

	log.Debug().Msgf(`new httpProxySession "%s" for device "%s"`, sid, devid)

	domain := c.Request.Header.Get("HttpProxyRedirDomain")
	if domain == "" {
		domain = cfg.HttpProxyRedirDomain
		if domain != "" {
			log.Debug().Msgf("set cookie domain from config: %s, devid: %s", domain, devid)
		}
	} else {
		log.Debug().Msgf("set cookie domain from HTTP header: %s, devid: %s", domain, devid)
	}

	c.SetCookie("rtty-http-sid", sid, 0, "", domain, false, true)
	c.SetCookie("rtty-http-group", group, 0, "", domain, false, true)
	c.SetCookie("rtty-http-devid", devid, 0, "", domain, false, true)
	c.SetCookie("rtty-http-proto", proto, 0, "", domain, false, true)
	c.SetCookie("rtty-http-destaddr", addr, 0, "", domain, false, true)

	c.Redirect(http.StatusFound, location)
}

func sendHttpReq(dev *Device, https bool, srcAddr []byte, destAddr []byte, data []byte) {
	bb := bytebufferpool.Get()
	defer bytebufferpool.Put(bb)

	if dev.proto > 3 {
		if https {
			bb.WriteByte(1)
		} else {
			bb.WriteByte(0)
		}
	}

	bb.Write(srcAddr)
	bb.Write(destAddr)
	bb.Write(data)

	dev.WriteMsg(msgTypeHttp, "", bb.Bytes())
}

func genDestAddr(addr string) []byte {
	destIP, destPort, err := httpProxyVaildAddr(addr)
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

func httpProxyVaildAddr(addr string) (net.IP, uint16, error) {
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

type HttpProxyWriter struct {
	destAddr          []byte
	srcAddr           []byte
	hostHeaderRewrite string
	dev               *Device
	https             bool
}

func (rw *HttpProxyWriter) Write(p []byte) (n int, err error) {
	sendHttpReq(rw.dev, rw.https, rw.srcAddr, rw.destAddr, p)
	return len(p), nil
}

func (rw *HttpProxyWriter) WriteRequest(req *http.Request) {
	req.Host = rw.hostHeaderRewrite
	req.Write(rw)
}
