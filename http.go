/* SPDX-License-Identifier: MIT */
/*
 * Author: Jianhui Zhao <zhaojh329@gmail.com>
 */

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zhaojh329/rtty-go/proto"
	"github.com/zhaojh329/rttys/v5/utils"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/valyala/bytebufferpool"
)

type HttpProxySession struct {
	expire   atomic.Int64
	ctx      context.Context
	cancel   context.CancelFunc
	devid    string
	group    string
	destaddr string
	https    bool
}

var httpProxySessions = sync.Map{}

const httpProxySessionsExpire = 15 * time.Minute

func (ses *HttpProxySession) Expire() {
	ses.expire.Store(time.Now().Add(httpProxySessionsExpire).Unix())
}

func (ses *HttpProxySession) String() string {
	return fmt.Sprintf("{devid: %s, group: %s, destaddr: %s, https: %v}",
		ses.devid, ses.group, ses.destaddr, ses.https)
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

var httpBufPool = sync.Pool{
	New: func() any {
		return &HttpBuf{
			buf: make([]byte, 1024*32),
		}
	},
}

type HttpBuf struct {
	buf []byte
}

func doHttpProxy(srv *RttyServer, c net.Conn) {
	defer logPanic()
	defer c.Close()

	head := bytebufferpool.Get()
	defer bytebufferpool.Put(head)

	br := bufio.NewReader(c)

	ses := httpReadRequest(c, br, head)
	if ses == nil {
		return
	}

	dev := srv.GetDevice(ses.group, ses.devid)
	if dev == nil {
		log.Debug().Msgf(`device "%s" group "%s" offline`, ses.devid, ses.group)
		sendHTTPErrorResponse(c, "offline")
		return
	}

	destAddr, host := genDestAddrAndHost(ses.destaddr, ses.https)

	var srcAddr [18]byte

	tcpAddr2Bytes(c.RemoteAddr().(*net.TCPAddr), srcAddr[:])

	ctx, cancel := context.WithCancel(ses.ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		c.Close()
		log.Debug().Msgf("http proxy conn closed: %s", ses)
		dev.https.Delete(srcAddr)
	}()

	log.Debug().Msgf("new http proxy conn: %s", ses)

	dev.https.Store(srcAddr, c)

	// Rewrite the HTTP host
	head.WriteString("Host: " + host + "\r\n")

	sendHttpReq(dev, ses.https, srcAddr[:], destAddr, head.B)

	hb := httpBufPool.Get().(*HttpBuf)
	defer httpBufPool.Put(hb)

	for {
		n, err := br.Read(hb.buf)
		if err != nil {
			return
		}
		sendHttpReq(dev, ses.https, srcAddr[:], destAddr, hb.buf[:n])
		ses.Expire()
	}
}

func httpReadRequest(conn net.Conn, br *bufio.Reader, head *bytebufferpool.ByteBuffer) *HttpProxySession {
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	// First line: GET /index.html HTTP/1.0
	line, err := br.ReadString('\n')
	if err != nil {
		return nil
	}

	head.WriteString(line)

	sid := ""

	for {
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))

		line, err := br.ReadString('\n')
		if err != nil {
			return nil
		}
		if line == "\r\n" {
			break
		}

		k, v, ok := strings.Cut(line, ":")
		if !ok {
			break
		}

		k = strings.ToLower(k)
		v = strings.TrimLeft(v, " \t")

		if k == "host" {
			continue
		}

		head.WriteString(line)

		if k == "cookie" {
			cookies, err := http.ParseCookie(v)
			if err != nil {
				break
			}

			for _, cookie := range cookies {
				if cookie.Name == "rtty-http-sid" {
					sid = cookie.Value
					break
				}
			}
			break
		}
	}

	if sid == "" {
		log.Debug().Msgf(`not found cookie "rtty-http-sid"`)
		sendHTTPErrorResponse(conn, "unauthorized")
		return nil
	}

	ses, ok := httpProxySessions.Load(sid)
	if !ok {
		log.Debug().Msgf(`not found httpProxySession "%s"`, sid)
		sendHTTPErrorResponse(conn, "unauthorized")
		return nil
	}

	conn.SetReadDeadline(time.Time{})

	return ses.(*HttpProxySession)
}

func httpProxyRedirect(a *APIServer, c *gin.Context, group string) {
	srv := a.srv
	cfg := &srv.cfg

	devid := c.Param("devid")
	proto := c.Param("proto")
	addr := c.Param("addr")
	rawPath := c.Param("path")

	if !a.callUserHookUrl(c) {
		c.Status(http.StatusForbidden)
		return
	}

	log.Debug().Msgf("httpProxyRedirect devid: %s, proto: %s, addr: %s, path: %s", devid, proto, addr, rawPath)

	_, _, err := httpProxyVaildAddr(addr, proto == "https")
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
		ctx:      ctx,
		cancel:   cancel,
		devid:    devid,
		group:    group,
		destaddr: addr,
		https:    proto == "https",
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

	dev.WriteMsg(proto.MsgTypeHttp, bb)
}

func genDestAddrAndHost(addr string, https bool) ([]byte, string) {
	destIP, destPort, err := httpProxyVaildAddr(addr, https)
	if err != nil {
		return nil, ""
	}

	b := make([]byte, 6)
	copy(b, destIP)

	binary.BigEndian.PutUint16(b[4:], destPort)

	host := addr

	ips, ports, _ := net.SplitHostPort(addr)
	if ports == "80" || ports == "443" {
		host = ips
	}

	return b, host
}

func tcpAddr2Bytes(addr *net.TCPAddr, b []byte) {
	binary.BigEndian.PutUint16(b[:2], uint16(addr.Port))
	copy(b[2:], addr.IP)
}

func httpProxyVaildAddr(addr string, https bool) (net.IP, uint16, error) {
	ips, ports, err := net.SplitHostPort(addr)
	if err != nil {
		ips = addr

		if https {
			ports = "443"
		} else {
			ports = "80"
		}
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

func sendHTTPErrorResponse(conn net.Conn, errorType string) {
	fs, _ := fs.Sub(staticFs, "assets")

	f, err := fs.Open("http-proxy-err.html")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		fmt.Println(err)
		return
	}

	content = bytes.ReplaceAll(content, []byte("{{.}}"), []byte(errorType))

	response := "HTTP/1.1 200 OK\r\n"
	response += "Content-Type: text/html; charset=utf-8\r\n"
	response += fmt.Sprintf("Content-Length: %d\r\n", len(content))
	response += "Connection: close\r\n"
	response += "\r\n"

	conn.Write([]byte(response))
	conn.Write(content)
}
