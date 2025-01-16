package main

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"rttys/client"
	"rttys/utils"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type httpResp struct {
	data []byte
	dev  client.Client
}

type httpReq struct {
	devid string
	data  []byte
}

var httpProxyCons sync.Map
var httpProxySessions sync.Map

func handleHttpProxyResp(resp *httpResp) {
	data := resp.data
	addr := data[:18]
	data = data[18:]

	if cons, ok := httpProxyCons.Load(resp.dev.DeviceID()); ok {
		if c, ok := cons.(*sync.Map).Load(string(addr)); ok {
			c := c.(net.Conn)
			if len(data) == 0 {
				c.Close()
			} else {
				c.Write(data)
			}
		}
	}
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

type HttpProxyWriter struct {
	destAddr          []byte
	srcAddr           []byte
	hostHeaderRewrite string
	br                *broker
	dev               client.Client
	https             bool
}

func sendHttpReq(br *broker, c client.Client, https bool, srcAddr []byte, destAddr []byte, data []byte) {
	msg := []byte{}
	dev := c.(*device)

	if dev.proto > 3 {
		if https {
			msg = append(msg, 1)
		} else {
			msg = append(msg, 0)
		}
	}

	msg = append(msg, srcAddr...)
	msg = append(msg, destAddr...)
	msg = append(msg, data...)

	br.httpReq <- &httpReq{dev.id, msg}
}

func (rw *HttpProxyWriter) Write(p []byte) (n int, err error) {
	sendHttpReq(rw.br, rw.dev, rw.https, rw.srcAddr, rw.destAddr, p)
	return len(p), nil
}

func (rw *HttpProxyWriter) WriteRequest(req *http.Request) {
	req.Host = rw.hostHeaderRewrite
	req.Write(rw)
}

func doHttpProxy(brk *broker, c net.Conn) {
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

	dev, ok := brk.devices[devid]
	if !ok {
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

	if cons, _ := httpProxyCons.LoadOrStore(devid, &sync.Map{}); true {
		cons := cons.(*sync.Map)
		cons.Store(string(srcAddr), c)
	}

	exit := make(chan struct{})

	if v, ok := httpProxySessions.Load(sid); ok {
		go func() {
			select {
			case <-v.(chan struct{}):
				c.Close()
			case <-exit:
			}

			cons, ok := httpProxyCons.Load(devid)
			if ok {
				cons := cons.(*sync.Map)
				cons.Delete(string(srcAddr))
			}
		}()
	} else {
		log.Debug().Msgf(`not found session "%s", devid "%s"`, sid, devid)
		return
	}

	log.Debug().Msgf("doHttpProxy devid: %s, https: %v, destaddr: %s", devid, https, hostHeaderRewrite)

	hpw := &HttpProxyWriter{destAddr, srcAddr, hostHeaderRewrite, brk, dev.(*device), https}

	req.Host = hostHeaderRewrite
	hpw.WriteRequest(req)

	if req.Header.Get("Upgrade") == "websocket" {
		b := make([]byte, 4096)

		for {
			n, err := c.Read(b)
			if err != nil {
				close(exit)
				return
			}

			sendHttpReq(brk, dev, https, srcAddr, destAddr, b[:n])
		}
	} else {
		for {
			req, err := http.ReadRequest(br)
			if err != nil {
				close(exit)
				return
			}

			hpw.WriteRequest(req)
		}
	}
}

func listenHttpProxy(brk *broker) {
	cfg := brk.cfg

	if cfg.AddrHttpProxy != "" {
		addr, err := net.ResolveTCPAddr("tcp", cfg.AddrHttpProxy)
		if err != nil {
			log.Warn().Msg("invalid http proxy addr: " + err.Error())
		} else {
			cfg.HttpProxyPort = addr.Port
		}
	}

	if cfg.HttpProxyPort == 0 {
		log.Info().Msg("Automatically select an available port for http proxy")
	}

	ln, err := net.Listen("tcp4", cfg.AddrHttpProxy)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	if cfg.WebUISslCert != "" && cfg.WebUISslKey != "" {
		crt, err := tls.LoadX509KeyPair(cfg.SslCert, cfg.SslKey)
		if err != nil {
			log.Fatal().Msg(err.Error())
		}

		tlsConfig := &tls.Config{Certificates: []tls.Certificate{crt}}

		ln = tls.NewListener(ln, tlsConfig)
	}

	cfg.HttpProxyPort = ln.Addr().(*net.TCPAddr).Port

	if cfg.WebUISslCert != "" && cfg.WebUISslKey != "" {
		log.Info().Msgf("Listen http proxy on: %s SSL on", ln.Addr().(*net.TCPAddr))
	} else {
		log.Info().Msgf("Listen http proxy on: %s SSL off", ln.Addr().(*net.TCPAddr))
	}

	go func() {
		defer ln.Close()

		for {
			c, err := ln.Accept()
			if err != nil {
				log.Error().Msg(err.Error())
				continue
			}

			go doHttpProxy(brk, c)
		}
	}()
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

func httpProxyRedirect(br *broker, c *gin.Context) {
	cfg := br.cfg
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

	_, ok := br.devices[devid]
	if !ok {
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

		if cfg.WebUISslCert != "" && cfg.WebUISslKey != "" {
			location = "https://" + host
		} else {
			location = "http://" + host
		}

		if cfg.HttpProxyPort != 80 {
			location += fmt.Sprintf(":%d", cfg.HttpProxyPort)
		}
	}

	location += path.Path

	location += fmt.Sprintf("?_=%d", time.Now().Unix())

	if path.RawQuery != "" {
		location += "&" + path.RawQuery
	}

	sid, err := c.Cookie("rtty-http-sid")
	if err == nil {
		if v, ok := httpProxySessions.Load(sid); ok {
			close(v.(chan struct{}))
			httpProxySessions.Delete(sid)
			log.Debug().Msgf(`del old httpProxySession "%s" for device "%s"`, sid, devid)
		}
	}

	sid = utils.GenUniqueID("http-proxy")

	httpProxySessions.Store(sid, make(chan struct{}))

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
	c.SetCookie("rtty-http-devid", devid, 0, "", domain, false, true)
	c.SetCookie("rtty-http-proto", proto, 0, "", domain, false, true)
	c.SetCookie("rtty-http-destaddr", addr, 0, "", domain, false, true)

	c.Redirect(http.StatusFound, location)
}
