package main

import (
	"embed"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"rttys/config"
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

func httpLogin(cfg *config.Config, password string) bool {
	return cfg.Password == "" || cfg.Password == password
}

func isLocalRequest(c *gin.Context) bool {
	addr, _ := net.ResolveTCPAddr("tcp", c.Request.RemoteAddr)
	return addr.IP.IsLoopback()
}

func httpAuth(cfg *config.Config, c *gin.Context) bool {
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

func apiStart(br *broker) {
	cfg := br.cfg

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	r.Use(gin.Recovery())

	if cfg.AllowOrigins {
		log.Debug().Msg("Allow all origins")
		r.Use(cors.Default())
	}

	authorized := r.Group("/", func(c *gin.Context) {
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
			devid := c.Param("devid")
			if _, ok := br.getDevice(devid); !ok {
				c.Redirect(http.StatusFound, "/error/offline")
				return
			}

			c.Redirect(http.StatusFound, "/rtty/"+devid)
			return
		}
		serveUser(br, c)
	})

	authorized.GET("/devs", func(c *gin.Context) {
		type DeviceInfo struct {
			ID          string `json:"id"`
			Connected   uint32 `json:"connected"`
			Uptime      uint32 `json:"uptime"`
			Description string `json:"description"`
			Proto       uint8  `json:"proto"`
		}

		devs := make([]DeviceInfo, 0)

		br.devices.Range(func(key, value any) bool {
			dev := value.(*device)

			devs = append(devs, DeviceInfo{
				ID:          dev.id,
				Description: dev.desc,
				Connected:   uint32(time.Now().Unix() - dev.timestamp),
				Uptime:      dev.uptime,
				Proto:       dev.proto,
			})

			return true
		})

		c.JSON(http.StatusOK, devs)
	})

	authorized.GET("/dev/:devid", func(c *gin.Context) {
		if dev, ok := br.getDevice(c.Param("devid")); ok {
			c.JSON(http.StatusOK, gin.H{
				"description": dev.desc,
				"connected":   uint32(time.Now().Unix() - dev.timestamp),
				"uptime":      dev.uptime,
				"proto":       dev.proto,
			})
		} else {
			c.Status(http.StatusNotFound)
		}
	})

	authorized.POST("/cmd/:devid", func(c *gin.Context) {
		handleCmdReq(br, c)
	})

	authorized.Any("/web/:devid/:proto/:addr/*path", func(c *gin.Context) {
		httpProxyRedirect(br, c)
	})

	authorized.GET("/signout", func(c *gin.Context) {
		sid, err := c.Cookie("sid")
		if err != nil || !httpSessions.Exists(sid) {
			return
		}

		httpSessions.Del(sid)

		c.Status(http.StatusOK)
	})

	authorized.GET("/file/:sid", func(c *gin.Context) {
		sid := c.Param("sid")
		if fp, ok := br.fileProxy.Load(sid); ok {
			fp := fp.(*fileProxy)

			if s, ok := br.getSession(sid); ok {
				fp.Ack(s.dev, sid)
			}

			defer func() {
				if err := recover(); err != nil {
					if ne, ok := err.(*net.OpError); ok {
						if se, ok := ne.Err.(*os.SyscallError); ok {
							if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
								fp.reader.Close()
							}
						}
					}
				}
			}()

			c.DataFromReader(http.StatusOK, -1, "application/octet-stream", fp.reader, nil)
			br.fileProxy.Delete(sid)
		}
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

	fs, _ := fs.Sub(staticFs, "ui/dist")
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

	go func() {
		log.Info().Msgf("Listen user on: %s", cfg.AddrUser)
		log.Fatal().Err(r.Run(cfg.AddrUser))
	}()
}
