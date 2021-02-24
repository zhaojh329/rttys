package main

import (
	"database/sql"
	"embed"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
	"github.com/zhaojh329/rttys/cache"
	"github.com/zhaojh329/rttys/config"
	"github.com/zhaojh329/rttys/utils"
)

type credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

var httpSessions *cache.Cache

//go:embed frontend/dist
var static embed.FS

func allowOrigin(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")
}

func httpLogin(cfg *config.Config, creds *credentials) bool {
	if creds.Username == "" || creds.Password == "" {
		return false
	}

	db, err := sql.Open("sqlite3", cfg.DB)
	if err != nil {
		log.Error().Msg(err.Error())
		return false
	}
	defer db.Close()

	cnt := 0

	db.QueryRow("SELECT COUNT(*) FROM account WHERE username = ? AND password = ?", creds.Username, creds.Password).Scan(&cnt)
	if cnt == 0 {
		return false
	}

	return true
}

func authorizedDev(devid string, cfg *config.Config) bool {
	if cfg.WhiteList == nil {
		return true
	}

	_, ok := cfg.WhiteList[devid]
	return ok
}

func httpAuth(c *gin.Context) bool {
	addr, _ := net.ResolveTCPAddr("tcp", c.Request.RemoteAddr)
	if addr.IP.IsLoopback() {
		return true
	}

	cookie, err := c.Cookie("sid")
	if err != nil || !httpSessions.Have(cookie) {
		return false
	}

	httpSessions.Active(cookie, 0)

	return true
}

func initDb(cfg *config.Config) {
	db, err := sql.Open("sqlite3", cfg.DB)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}
	defer db.Close()

	db.Exec("CREATE TABLE IF NOT EXISTS account(username TEXT PRIMARY KEY NOT NULL, password TEXT NOT NULL)")
}

func httpStart(br *broker) {
	cfg := br.cfg

	initDb(cfg)

	httpSessions = cache.New(30*time.Minute, 5*time.Second)

	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	authorized := r.Group("/", func(c *gin.Context) {
		devid := c.Param("devid")
		if devid != "" && authorizedDev(devid, cfg) {
			return
		}

		if !httpAuth(c) {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
	})

	authorized.GET("/fontsize/:devid", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"size": cfg.FontSize})
	})

	authorized.POST("/fontsize/:devid", func(c *gin.Context) {
		type Resp struct {
			Size int `json:"size"`
		}
		var r Resp
		err := c.BindJSON(&r)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		cfg.FontSize = r.Size
		c.String(http.StatusOK, "OK")
	})

	authorized.GET("/connect/:devid", func(c *gin.Context) {
		if c.GetHeader("Upgrade") != "websocket" {
			c.Redirect(http.StatusFound, "/rtty/"+c.Param("devid"))
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
		}

		devs := make([]DeviceInfo, 0)

		for id, dev := range br.devices {
			dev := dev.(*device)
			devs = append(devs, DeviceInfo{id, uint32(time.Now().Unix() - dev.timestamp), dev.uptime, dev.desc})
		}

		allowOrigin(c.Writer)

		c.JSON(http.StatusOK, devs)
	})

	authorized.POST("/cmd/:devid", func(c *gin.Context) {
		allowOrigin(c.Writer)

		handleCmdReq(br, c)
	})

	r.Any("/web/:devid/:addr/*path", func(c *gin.Context) {
		webReqRedirect(br, c)
	})

	r.GET("/authorized/:devid", func(c *gin.Context) {
		authorized := authorizedDev(c.Param("devid"), cfg) || httpAuth(c)
		c.JSON(http.StatusOK, gin.H{
			"authorized": authorized,
		})
	})

	r.POST("/signin", func(c *gin.Context) {
		var creds credentials

		err := jsoniter.NewDecoder(c.Request.Body).Decode(&creds)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		if httpLogin(cfg, &creds) {
			sid := utils.GenUniqueID("http")
			httpSessions.Set(sid, true, 0)

			c.SetCookie("sid", sid, 0, "", "", false, true)

			c.JSON(http.StatusOK, gin.H{
				"sid":      sid,
				"username": creds.Username,
			})
			return
		}

		c.Status(http.StatusForbidden)
	})

	r.POST("/signup", func(c *gin.Context) {
		var creds credentials

		err := jsoniter.NewDecoder(c.Request.Body).Decode(&creds)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		db, err := sql.Open("sqlite3", cfg.DB)
		if err != nil {
			log.Error().Msg(err.Error())
			c.Status(http.StatusInternalServerError)
			return
		}
		defer db.Close()

		_, err = db.Exec("INSERT INTO account values(?,?)", creds.Username, creds.Password)
		if err != nil {
			log.Error().Msg(err.Error())

			if strings.Contains(err.Error(), "UNIQUE constraint failed") {
				c.Status(http.StatusForbidden)
			} else {
				c.Status(http.StatusInternalServerError)
			}

			return
		}

		c.Status(http.StatusOK)
	})

	r.NoRoute(func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/frontend/dist/") {
			c.Request.URL.Path = "/frontend/dist" + c.Request.URL.Path
			r.HandleContext(c)
			return
		}

		p := path.Clean(c.Request.URL.Path)

		if p != "/frontend/dist/" {
			f, err := static.Open(p[1:])
			if err != nil {
				c.Request.URL.Path = "/frontend/dist/"
				r.HandleContext(c)
				return
			}
			f.Close()
		}

		http.FileServer(http.FS(static)).ServeHTTP(c.Writer, c.Request)
	})

	go func() {
		var err error

		if cfg.SslCert != "" && cfg.SslKey != "" {
			log.Info().Msgf("Listen user on: %s SSL on", cfg.AddrUser)
			err = r.RunTLS(cfg.AddrUser, cfg.SslCert, cfg.SslKey)
		} else {
			log.Info().Msgf("Listen user on: %s SSL off", cfg.AddrUser)
			err = r.Run(cfg.AddrUser)
		}

		log.Fatal().Err(err)
	}()
}
