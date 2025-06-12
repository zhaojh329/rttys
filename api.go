package main

import (
	"embed"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"rttys/config"
	"rttys/utils"

	"github.com/fanjindong/go-cache"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

var httpSessions = cache.NewMemCache(cache.WithClearInterval(time.Minute))

const httpSessionExpire = 30 * time.Minute

//go:embed ui/dist
var staticFs embed.FS

func allowOrigin(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")
}

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

		allowOrigin(c.Writer)

		c.JSON(http.StatusOK, devs)
	})

	authorized.GET("/dev/:devid", func(c *gin.Context) {
		allowOrigin(c.Writer)

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
		allowOrigin(c.Writer)

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
			sid := utils.GenUniqueID("http")

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

	r.NoRoute(func(c *gin.Context) {
		fs, _ := fs.Sub(staticFs, "ui/dist")

		path := c.Request.URL.Path

		if path != "/" {
			f, err := fs.Open(path[1:])
			if err != nil {
				c.Request.URL.Path = "/"
				r.HandleContext(c)
				return
			}

			if strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
				if strings.HasSuffix(path, "css") || strings.HasSuffix(path, "js") {
					magic := make([]byte, 2)
					f.Read(magic)
					if magic[0] == 0x1f && magic[1] == 0x8b {
						c.Writer.Header().Set("Content-Encoding", "gzip")
					}
				}
			}

			f.Close()
		}

		http.FileServer(http.FS(fs)).ServeHTTP(c.Writer, c.Request)
	})

	go func() {
		var err error

		if cfg.WebUISslCert != "" && cfg.WebUISslKey != "" {
			log.Info().Msgf("Listen user on: %s SSL on", cfg.AddrUser)
			err = r.RunTLS(cfg.AddrUser, cfg.WebUISslCert, cfg.WebUISslKey)
		} else {
			log.Info().Msgf("Listen user on: %s SSL off", cfg.AddrUser)
			err = r.Run(cfg.AddrUser)
		}

		log.Fatal().Err(err)
	}()
}
