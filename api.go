/* SPDX-License-Identifier: MIT */
/*
 * Author: Jianhui Zhao <zhaojh329@gmail.com>
 */

package main

import (
	"io/fs"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/zhaojh329/rttys/v5/utils"

	"github.com/fanjindong/go-cache"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

const httpSessionExpire = 30 * time.Minute

func (srv *RttyServer) ListenAPI() error {
	cfg := &srv.cfg

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	fs, err := fs.Sub(staticFs, "assets/dist")
	if err != nil {
		return err
	}

	root := http.FS(fs)

	a := &APIServer{
		sessions: cache.NewMemCache(cache.WithClearInterval(time.Minute)),
		fh:       http.FileServer(root),
		srv:      srv,
		r:        r,
		root:     root,
	}

	r.Use(func(c *gin.Context) {
		c.Next()
		log.Debug().Msgf("%s - \"%s %s %s %d\"", c.ClientIP(),
			c.Request.Method, c.Request.URL.Path, c.Request.Proto, c.Writer.Status())
	})

	if cfg.AllowOrigins {
		log.Debug().Msg("Allow all origins")
		r.Use(cors.Default())
	}

	authorized := r.Group("/", func(c *gin.Context) {
		if !cfg.LocalAuth && isLocalRequest(c) {
			return
		}

		if !a.auth(c) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
	})

	authorized.GET("/connect/:devid", a.handleConnect)
	authorized.GET("/counts", a.handleCounts)
	authorized.GET("/groups", a.handleGroups)
	authorized.GET("/devs", a.handleDevs)
	authorized.GET("/dev/:devid", a.handleDev)
	authorized.POST("/cmd/:devid", a.handleCmd)
	authorized.Any("/web/:devid/:proto/:addr/*path", a.handleWeb)
	authorized.Any("/web2/:group/:devid/:proto/:addr/*path", a.handleWeb2)
	authorized.GET("/signout", a.handleSignout)

	r.POST("/signin", a.handleSignin)
	r.GET("/alive", a.handleAlive)

	r.NoRoute(a.handleFile)

	ln, err := net.Listen("tcp", cfg.AddrUser)
	if err != nil {
		return err
	}
	defer ln.Close()

	log.Info().Msgf("Listen users on: %s", ln.Addr().(*net.TCPAddr))

	return r.RunListener(ln)
}

func isLocalRequest(c *gin.Context) bool {
	addr, _ := net.ResolveTCPAddr("tcp", c.Request.RemoteAddr)
	return addr.IP.IsLoopback()
}

type APIServer struct {
	srv      *RttyServer
	sessions cache.ICache
	root     http.FileSystem
	fh       http.Handler
	r        *gin.Engine
}

func (a *APIServer) auth(c *gin.Context) bool {
	cfg := &a.srv.cfg

	if !cfg.LocalAuth && isLocalRequest(c) {
		return true
	}

	if cfg.Password == "" {
		return true
	}

	sid, err := c.Cookie("sid")
	if err != nil || !a.sessions.Exists(sid) {
		return false
	}

	a.sessions.Expire(sid, httpSessionExpire)

	return true
}

func (a *APIServer) callUserHookUrl(c *gin.Context) bool {
	cfg := &a.srv.cfg

	if cfg.UserHookUrl == "" {
		return true
	}

	upath := c.Request.URL.RawPath

	// Create HTTP request with original headers
	req, err := http.NewRequest("GET", cfg.UserHookUrl, nil)
	if err != nil {
		log.Error().Err(err).Msgf("create hook request for \"%s\" fail", upath)
		return false
	}

	// Copy all headers from original request
	for key, values := range c.Request.Header {
		lowerKey := strings.ToLower(key)
		if lowerKey == "upgrade" || lowerKey == "connection" || lowerKey == "accept-encoding" {
			continue
		}

		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Add custom headers for hook identification
	req.Header.Set("X-Rttys-Hook", "true")
	req.Header.Set("X-Original-Method", c.Request.Method)
	req.Header.Set("X-Original-URL", c.Request.URL.String())

	cli := &http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := cli.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("call user hook url for \"%s\" fail", upath)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("call user hook url for \"%s\", StatusCode: %d", upath, resp.StatusCode)
		return false
	}

	return true
}

func (a *APIServer) handleConnect(c *gin.Context) {
	if !a.callUserHookUrl(c) {
		c.Status(http.StatusForbidden)
		return
	}

	if c.GetHeader("Upgrade") != "websocket" {
		group := c.Query("group")
		devid := c.Param("devid")
		if dev := a.srv.GetDevice(group, devid); dev == nil {
			c.Redirect(http.StatusFound, "/error/offline")
			return
		}

		url := "/rtty/" + devid

		if group != "" {
			url += "?group=" + group
		}

		c.Redirect(http.StatusFound, url)
	} else {
		handleUserConnection(a.srv, c)
	}
}

func (a *APIServer) handleCounts(c *gin.Context) {
	count := 0

	a.srv.groups.Range(func(key, value any) bool {
		count += int(value.(*DeviceGroup).count.Load())
		return true
	})

	c.JSON(http.StatusOK, gin.H{"count": count})
}

func (a *APIServer) handleGroups(c *gin.Context) {
	groups := []string{""}

	a.srv.groups.Range(func(key, value any) bool {
		if key != "" {
			groups = append(groups, key.(string))
		}
		return true
	})

	c.JSON(http.StatusOK, groups)
}

func (a *APIServer) handleDevs(c *gin.Context) {
	devs := make([]*DeviceInfo, 0)
	g := a.srv.GetGroup(c.Query("group"), false)

	if g == nil {
		c.JSON(http.StatusOK, devs)
		return
	}

	g.devices.Range(func(key, value any) bool {
		dev := value.(*Device)

		devs = append(devs, &DeviceInfo{
			Group:     dev.group,
			ID:        dev.id,
			Desc:      dev.desc,
			Connected: uint32(time.Now().Unix() - dev.timestamp),
			Uptime:    dev.uptime,
			Proto:     dev.proto,
			IPaddr:    dev.conn.RemoteAddr().(*net.TCPAddr).IP.String(),
		})

		return true
	})

	c.JSON(http.StatusOK, devs)
}

func (a *APIServer) handleDev(c *gin.Context) {
	if dev := a.srv.GetDevice(c.Query("group"), c.Param("devid")); dev != nil {
		info := &DeviceInfo{
			ID:        dev.id,
			Desc:      dev.desc,
			Connected: uint32(time.Now().Unix() - dev.timestamp),
			Uptime:    dev.uptime,
			Proto:     dev.proto,
			IPaddr:    dev.conn.RemoteAddr().(*net.TCPAddr).IP.String(),
		}
		c.JSON(http.StatusOK, info)
	} else {
		c.Status(http.StatusNotFound)
	}
}

func (a *APIServer) handleCmd(c *gin.Context) {
	if !a.callUserHookUrl(c) {
		c.Status(http.StatusForbidden)
		return
	}

	cmdInfo := &CommandReqInfo{}

	err := c.BindJSON(&cmdInfo)
	if err != nil || cmdInfo.Cmd == "" || cmdInfo.Username == "" {
		cmdErrResp(c, rttyCmdErrInvalid)
		return
	}

	dev := a.srv.GetDevice(c.Query("group"), c.Param("devid"))
	if dev == nil {
		cmdErrResp(c, rttyCmdErrOffline)
		return
	}

	dev.handleCmdReq(c, cmdInfo)
}

func (a *APIServer) handleWeb(c *gin.Context) {
	httpProxyRedirect(a, c, "")
}

func (a *APIServer) handleWeb2(c *gin.Context) {
	group := c.Param("group")
	httpProxyRedirect(a, c, group)
}

func (a *APIServer) handleSignout(c *gin.Context) {
	sid, err := c.Cookie("sid")
	if err != nil || !a.sessions.Exists(sid) {
		return
	}

	a.sessions.Del(sid)

	c.Status(http.StatusOK)
}

func (a *APIServer) handleSignin(c *gin.Context) {
	cfg := &a.srv.cfg

	type credentials struct {
		Password string `json:"password"`
	}

	creds := credentials{}

	err := c.BindJSON(&creds)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	if cfg.Password == creds.Password {
		sid := utils.GenUniqueID()

		a.sessions.Set(sid, true, cache.WithEx(httpSessionExpire))

		c.SetCookie("sid", sid, 0, "", "", false, true)
		c.Status(http.StatusOK)
		return
	}

	c.Status(http.StatusUnauthorized)
}

func (a *APIServer) handleAlive(c *gin.Context) {
	if !a.auth(c) {
		c.AbortWithStatus(http.StatusUnauthorized)
	} else {
		c.Status(http.StatusOK)
	}
}

func (a *APIServer) handleFile(c *gin.Context) {
	upath := path.Clean(c.Request.URL.Path)
	root := a.root

	if strings.HasSuffix(upath, ".js") || strings.HasSuffix(upath, ".css") {
		if strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
			f, err := root.Open(upath + ".gz")
			if err == nil {
				f.Close()

				c.Request.URL.Path += ".gz"

				if strings.HasSuffix(upath, ".js") {
					c.Writer.Header().Set("Content-Type", "application/javascript")
				} else if strings.HasSuffix(upath, ".css") {
					c.Writer.Header().Set("Content-Type", "text/css")
				}

				c.Writer.Header().Set("Content-Encoding", "gzip")
			}
		}
	} else if upath != "/" {
		f, err := root.Open(upath)
		if err != nil {
			c.Request.URL.Path = "/"
			a.r.HandleContext(c)
			return
		}
		defer f.Close()
	}

	a.fh.ServeHTTP(c.Writer, c.Request)
}
