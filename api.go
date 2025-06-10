package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"rttys/cache"
	"rttys/config"
	"rttys/utils"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

var httpSessions *cache.Cache

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

func devInWhiteList(devid string, cfg *config.Config) bool {
	if cfg.WhiteList == nil {
		return true
	}

	_, ok := cfg.WhiteList[devid]
	return ok
}

func isLocalRequest(c *gin.Context) bool {
	addr, _ := net.ResolveTCPAddr("tcp", c.Request.RemoteAddr)
	return addr.IP.IsLoopback()
}

func httpAuth(cfg *config.Config, c *gin.Context) bool {
	if !cfg.LocalAuth && isLocalRequest(c) {
		return true
	}

	cookie, err := c.Cookie("sid")
	if err != nil || !httpSessions.Have(cookie) {
		return false
	}

	httpSessions.Active(cookie, 0)

	return true
}

func apiStart(br *broker) {
	cfg := br.cfg

	httpSessions = cache.New(30*time.Minute, 5*time.Second)

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	r.Use(gin.Recovery())

	authorized := r.Group("/", func(c *gin.Context) {
		devid := ""

		if !cfg.LocalAuth && isLocalRequest(c) {
			return
		}

		if strings.HasPrefix(c.Request.URL.Path, "/connect/") {
			devid = c.Param("devid")
			if devid == "" {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			if devInWhiteList(devid, cfg) {
				return
			}
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
			Online      bool   `json:"online"`
			Proto       uint8  `json:"proto"`
		}

		db, err := instanceDB(cfg.DB)
		if err != nil {
			log.Error().Msg(err.Error())
			c.Status(http.StatusInternalServerError)
			return
		}
		defer db.Close()

		sql := "SELECT id, description FROM device"

		devs := make([]DeviceInfo, 0)

		rows, err := db.Query(sql)
		if err != nil {
			log.Error().Msg(err.Error())
			c.Status(http.StatusInternalServerError)
			return
		}

		for rows.Next() {
			id := ""
			desc := ""

			err := rows.Scan(&id, &desc)
			if err != nil {
				log.Error().Msg(err.Error())
				break
			}

			di := DeviceInfo{
				ID:          id,
				Description: desc,
			}

			if dev, ok := br.getDevice(id); ok {
				di.Connected = uint32(time.Now().Unix() - dev.timestamp)
				di.Uptime = dev.uptime
				di.Online = true
				di.Proto = dev.proto
			}

			devs = append(devs, di)
		}

		allowOrigin(c.Writer)

		c.JSON(http.StatusOK, devs)
	})

	authorized.POST("/cmd/:devid", func(c *gin.Context) {
		allowOrigin(c.Writer)

		handleCmdReq(br, c)
	})

	r.Any("/web/:devid/:proto/:addr/*path", func(c *gin.Context) {
		httpProxyRedirect(br, c)
	})

	r.GET("/authorized/:devid", func(c *gin.Context) {
		devid := c.Param("devid")
		authorized := !cfg.LocalAuth && isLocalRequest(c)

		if !authorized && devInWhiteList(devid, cfg) {
			authorized = true
		}

		if !authorized && httpAuth(cfg, c) {
			authorized = httpAuth(cfg, c)
		}

		c.JSON(http.StatusOK, gin.H{
			"authorized": authorized,
		})
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
			httpSessions.Set(sid, true, 0)

			c.SetCookie("sid", sid, 0, "", "", false, true)

			c.JSON(http.StatusOK, gin.H{
				"sid": sid,
			})
			return
		}

		c.Status(http.StatusForbidden)
	})

	r.GET("/alive", func(c *gin.Context) {
		if !httpAuth(cfg, c) {
			c.AbortWithStatus(http.StatusUnauthorized)
		} else {
			c.Status(http.StatusOK)
		}
	})

	r.GET("/signout", func(c *gin.Context) {
		cookie, err := c.Cookie("sid")
		if err != nil || !httpSessions.Have(cookie) {
			return
		}

		httpSessions.Del(cookie)

		c.Status(http.StatusOK)
	})

	authorized.POST("/delete", func(c *gin.Context) {
		type deldata struct {
			Devices []string `json:"devices"`
		}

		data := deldata{}

		err := c.BindJSON(&data)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		db, err := instanceDB(cfg.DB)
		if err != nil {
			log.Error().Msg(err.Error())
			return
		}
		defer db.Close()

		for _, devid := range data.Devices {
			if _, ok := br.getDevice(devid); !ok {
				sql := fmt.Sprintf("DELETE FROM device WHERE id = '%s'", devid)
				db.Exec(sql)
			}
		}

		c.Status(http.StatusOK)
	})

	r.GET("/file/:sid", func(c *gin.Context) {
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
