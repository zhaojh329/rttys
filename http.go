package main

import (
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/rakyll/statik/fs"
	"github.com/rs/zerolog/log"
	"github.com/zhaojh329/rttys/cache"
	_ "github.com/zhaojh329/rttys/statik"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"time"
)

type Credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

var httpSessions *cache.Cache

func allowOrigin(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")
}

func httpLogin(cfg *RttysConfig, creds *Credentials) bool {
	if cfg.httpUsername != creds.Username {
		return false
	}

	if cfg.httpPassword != "" {
		return cfg.httpPassword == creds.Password
	}

	return true
}

func httpStart(br *Broker, cfg *RttysConfig) {
	httpSessions = cache.New(30*time.Minute, 5*time.Second)

	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	authorized := r.Group("/", func(c *gin.Context) {
		cookie, err := c.Cookie("sid")
		if err != nil || !httpSessions.Have(cookie) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// Update
		httpSessions.Del(cookie)
		httpSessions.Set(cookie, true, 0)
	})

	authorized.GET("/fontsize", func(c *gin.Context) {
		c.String(http.StatusOK, "%d", cfg.fontSize)
	})

	authorized.POST("/fontsize", func(c *gin.Context) {
		size, err := strconv.Atoi(c.PostForm("size"))
		if err == nil {
			cfg.fontSize = size
		}
		c.String(http.StatusOK, "OK")
	})

	authorized.GET("/connect/:devid", func(c *gin.Context) {
		serveUser(br, c)
	})

	authorized.GET("/devs", func(c *gin.Context) {
		type DeviceInfo struct {
			ID          string `json:"id"`
			Uptime      int64  `json:"uptime"`
			Description string `json:"description"`
		}

		devs := make([]DeviceInfo, 0)

		for id, dev := range br.devices {
			dev := DeviceInfo{id, time.Now().Unix() - dev.timestamp, dev.desc}
			devs = append(devs, dev)
		}

		allowOrigin(c.Writer)

		c.JSON(http.StatusOK, devs)
	})

	r.GET("/cmd", func(c *gin.Context) {
		allowOrigin(c.Writer)

		done := make(chan struct{})
		req := &CommandReq{
			done: done,
			w:    c.Writer,
		}

		req.token = c.Query("token")

		br.cmdReq <- req
		<-done
	})

	r.POST("/cmd", func(c *gin.Context) {
		allowOrigin(c.Writer)

		done := make(chan struct{})
		req := &CommandReq{
			done: done,
			w:    c.Writer,
		}

		content, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		sid := jsoniter.Get(content, "sid").ToString()
		if _, ok := httpSessions.Get(sid); !ok {
			c.Status(http.StatusForbidden)
			return
		}

		req.content = content

		br.cmdReq <- req
		<-done
	})

	r.POST("/signin", func(c *gin.Context) {
		var creds Credentials

		err := jsoniter.NewDecoder(c.Request.Body).Decode(&creds)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}

		if httpLogin(cfg, &creds) {
			sid := genUniqueID("http")
			httpSessions.Set(sid, true, 0)

			c.SetCookie("sid", sid, 0, "", "", false, true)
			c.String(http.StatusOK, sid)
			return
		}

		c.Status(http.StatusForbidden)
	})

	statikFS, err := fs.New()
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	r.NoRoute(func(c *gin.Context) {
		f, err := statikFS.Open(path.Clean(c.Request.URL.Path))
		if err != nil {
			c.Redirect(http.StatusFound, "/")
			return
		}
		f.Close()
		http.FileServer(statikFS).ServeHTTP(c.Writer, c.Request)
	})

	if cfg.sslCert != "" && cfg.sslKey != "" {
		log.Info().Msgf("Listen user on: %s SSL on", cfg.addrUser)
		err = r.RunTLS(cfg.addrUser, cfg.sslCert, cfg.sslKey)
	} else {
		log.Info().Msgf("Listen user on: %s SSL off", cfg.addrUser)
		err = r.Run(cfg.addrUser)
	}

	log.Fatal().Err(err)
}
