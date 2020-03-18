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

func authorizedDev(devid string, cfg *RttysConfig) bool {
	if cfg.whiteList == nil {
		return true
	}

	_, ok := cfg.whiteList[devid]
	return ok
}

func httpAuth(c *gin.Context) bool {
	cookie, err := c.Cookie("sid")
	if err != nil || !httpSessions.Have(cookie) {
		return false
	}

	// Update
	httpSessions.Del(cookie)
	httpSessions.Set(cookie, true, 0)

	return true
}

func httpStart(br *Broker, cfg *RttysConfig) {
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
		c.JSON(http.StatusOK, gin.H{"size": cfg.fontSize})
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

		cfg.fontSize = r.Size
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

	authorized.GET("/cmd/:devid/:token", func(c *gin.Context) {
		allowOrigin(c.Writer)

		done := make(chan struct{})
		req := &CommandReq{
			done: done,
			w:    c.Writer,
		}

		req.token = c.Param("token")

		br.cmdReq <- req
		<-done
	})

	authorized.POST("/cmd/:devid", func(c *gin.Context) {
		allowOrigin(c.Writer)

		done := make(chan struct{})
		req := &CommandReq{
			done:  done,
			w:     c.Writer,
			devid: c.Param("devid"),
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

	r.GET("/authorized/:devid", func(c *gin.Context) {
		authorized := authorizedDev(c.Param("devid"), cfg) || httpAuth(c)
		c.JSON(http.StatusOK, gin.H{
			"authorized": authorized,
		})
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
			c.Request.URL.Path = "/"
			r.HandleContext(c)
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
