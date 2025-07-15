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
	"embed"
	"io/fs"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"rttys/utils"

	"github.com/fanjindong/go-cache"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

var httpSessions = cache.NewMemCache(cache.WithClearInterval(time.Minute))

const httpSessionExpire = 30 * time.Minute

//go:embed ui/dist
var staticFs embed.FS

func (srv *RttyServer) ListenAPI() error {
	cfg := &srv.cfg

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

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
		if !callUserHookUrl(cfg, c) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		if !cfg.LocalAuth && isLocalRequest(c) {
			return
		}

		if !httpAuth(cfg, c) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
	})

	authorized.GET("/connect/:devid", func(c *gin.Context) {
		if c.GetHeader("Upgrade") != "websocket" {
			group := c.Query("group")
			devid := c.Param("devid")
			if dev := srv.GetDevice(group, devid); dev == nil {
				c.Redirect(http.StatusFound, "/error/offline")
				return
			}

			url := "/rtty/" + devid

			if group != "" {
				url += "?group=" + group
			}

			c.Redirect(http.StatusFound, url)
		} else {
			handleUserConnection(srv, c)
		}
	})

	authorized.GET("/counts", func(c *gin.Context) {
		count := 0

		srv.groups.Range(func(key, value any) bool {
			count += int(value.(*DeviceGroup).count.Load())
			return true
		})

		c.JSON(http.StatusOK, gin.H{"count": count})
	})

	authorized.GET("/groups", func(c *gin.Context) {
		groups := []string{""}

		srv.groups.Range(func(key, value any) bool {
			if key != "" {
				groups = append(groups, key.(string))
			}
			return true
		})

		c.JSON(http.StatusOK, groups)
	})

	authorized.GET("/devs", func(c *gin.Context) {
		devs := make([]*DeviceInfo, 0)
		g := srv.GetGroup(c.Query("group"), false)

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
	})

	authorized.GET("/dev/:devid", func(c *gin.Context) {
		if dev := srv.GetDevice(c.Query("group"), c.Param("devid")); dev != nil {
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
	})

	authorized.POST("/cmd/:devid", func(c *gin.Context) {
		cmdInfo := &CommandReqInfo{}

		err := c.BindJSON(&cmdInfo)
		if err != nil || cmdInfo.Cmd == "" || cmdInfo.Username == "" {
			cmdErrResp(c, rttyCmdErrInvalid)
			return
		}

		dev := srv.GetDevice(c.Query("group"), c.Param("devid"))
		if dev == nil {
			cmdErrResp(c, rttyCmdErrOffline)
			return
		}

		dev.handleCmdReq(c, cmdInfo)
	})

	authorized.Any("/web/:devid/:proto/:addr/*path", func(c *gin.Context) {
		httpProxyRedirect(srv, c, "")
	})

	authorized.Any("/web2/:group/:devid/:proto/:addr/*path", func(c *gin.Context) {
		group := c.Param("group")
		httpProxyRedirect(srv, c, group)
	})

	authorized.GET("/signout", func(c *gin.Context) {
		sid, err := c.Cookie("sid")
		if err != nil || !httpSessions.Exists(sid) {
			return
		}

		httpSessions.Del(sid)

		c.Status(http.StatusOK)
	})

	r.POST("/signin", func(c *gin.Context) {
		type credentials struct {
			Password string `json:"password"`
		}

		creds := credentials{}

		err := c.BindJSON(&creds)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		if httpLogin(cfg, creds.Password) {
			sid := utils.GenUniqueID()

			httpSessions.Set(sid, true, cache.WithEx(httpSessionExpire))

			c.SetCookie("sid", sid, 0, "", "", false, true)
			c.Status(http.StatusOK)
			return
		}

		c.Status(http.StatusUnauthorized)
	})

	r.GET("/alive", func(c *gin.Context) {
		if !httpAuth(cfg, c) {
			c.AbortWithStatus(http.StatusUnauthorized)
		} else {
			c.Status(http.StatusOK)
		}
	})

	fs, err := fs.Sub(staticFs, "ui/dist")
	if err != nil {
		return err
	}

	root := http.FS(fs)
	fh := http.FileServer(root)

	r.NoRoute(func(c *gin.Context) {
		upath := path.Clean(c.Request.URL.Path)

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
				r.HandleContext(c)
				return
			}
			defer f.Close()
		}

		fh.ServeHTTP(c.Writer, c.Request)
	})

	ln, err := net.Listen("tcp", cfg.AddrUser)
	if err != nil {
		return err
	}
	defer ln.Close()

	log.Info().Msgf("Listen users on: %s", ln.Addr().(*net.TCPAddr))

	return r.RunListener(ln)
}

func callUserHookUrl(cfg *Config, c *gin.Context) bool {
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

func httpLogin(cfg *Config, password string) bool {
	return cfg.Password == "" || cfg.Password == password
}

func isLocalRequest(c *gin.Context) bool {
	addr, _ := net.ResolveTCPAddr("tcp", c.Request.RemoteAddr)
	return addr.IP.IsLoopback()
}

func httpAuth(cfg *Config, c *gin.Context) bool {
	if !cfg.LocalAuth && isLocalRequest(c) {
		return true
	}

	sid, err := c.Cookie("sid")
	if err != nil || !httpSessions.Exists(sid) {
		return false
	}

	httpSessions.Expire(sid, httpSessionExpire)

	return true
}
