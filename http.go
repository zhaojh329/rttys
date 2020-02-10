package main

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/rakyll/statik/fs"
	log "github.com/sirupsen/logrus"
	"github.com/zhaojh329/rttys/cache"
	"github.com/zhaojh329/rttys/pwauth"
	_ "github.com/zhaojh329/rttys/statik"
	"io/ioutil"
	"net/http"
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

func httpAuth(w http.ResponseWriter, r *http.Request) bool {
	c, err := r.Cookie("sid")
	if err != nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return false
	}

	if _, ok := httpSessions.Get(c.Value); !ok {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return false
	}

	// Update
	httpSessions.Del(c.Value)
	httpSessions.Set(c.Value, true, 0)

	return true
}

func httpLogin(cfg *RttysConfig, creds *Credentials) bool {
	if err := pwauth.Auth(creds.Username, creds.Password); err == nil {
		return true
	}

	if cfg.httpUsername != "" {
		if cfg.httpUsername != creds.Username {
			return false
		}

		if cfg.httpPassword != "" {
			return cfg.httpPassword == creds.Password
		}

		return true
	}

	return false
}

func httpStart(br *Broker, cfg *RttysConfig) {
	httpSessions = cache.New(30*time.Minute, 5*time.Second)

	statikFS, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}

	staticfs := http.FileServer(statikFS)

	if cfg.baseURL == "/" {
		cfg.baseURL = ""
	}

	http.HandleFunc(cfg.baseURL+"/ws", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := httpSessions.Get(r.URL.Query().Get("sid")); !ok {
			http.Error(w, "Invalid sid", http.StatusForbidden)
			return
		}

		serveUser(br, w, r)
	})

	http.HandleFunc(cfg.baseURL+"/cmd", func(w http.ResponseWriter, r *http.Request) {
		allowOrigin(w)

		done := make(chan struct{})
		req := &CommandReq{
			done: done,
			w:    w,
		}

		if r.Method == "GET" {
			req.token = r.URL.Query().Get("token")
		} else if r.Method == "POST" {
			content, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Error(err)
				return
			}

			sid := jsoniter.Get(content, "sid").ToString()
			if _, ok := httpSessions.Get(sid); !ok {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			req.content = content
		} else {
			http.Error(w, "MethodNotAllowed", http.StatusMethodNotAllowed)
			return
		}

		br.cmdReq <- req
		<-done
	})

	http.HandleFunc(cfg.baseURL+"/signin", func(w http.ResponseWriter, r *http.Request) {
		var creds Credentials

		err := jsoniter.NewDecoder(r.Body).Decode(&creds)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if httpLogin(cfg, &creds) {
			sid := genUniqueID("http")
			httpSessions.Set(sid, true, 0)

			http.SetCookie(w, &http.Cookie{
				Name:     "sid",
				Value:    sid,
				HttpOnly: true,
			})
			fmt.Fprint(w, sid)
			return
		}

		http.Error(w, "Forbidden", http.StatusForbidden)
	})

	http.HandleFunc(cfg.baseURL+"/devs", func(w http.ResponseWriter, r *http.Request) {
		type DeviceInfo struct {
			ID          string `json:"id"`
			Uptime      int64  `json:"uptime"`
			Description string `json:"description"`
		}

		if !httpAuth(w, r) {
			return
		}

		devs := make([]DeviceInfo, 0)

		for id, dev := range br.devices {
			dev := DeviceInfo{id, time.Now().Unix() - dev.timestamp, dev.desc}
			devs = append(devs, dev)
		}

		allowOrigin(w)

		resp, _ := jsoniter.Marshal(devs)

		w.Write(resp)
	})

	hfunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			t := r.URL.Query().Get("tmr")
			id := r.URL.Query().Get("id")

			if t == "" && id == "" {
				http.Redirect(w, r, cfg.baseURL+"?tmr="+strconv.FormatInt(time.Now().Unix(), 10), http.StatusFound)
				return
			}
		}

		staticfs.ServeHTTP(w, r)
	})

	if cfg.baseURL != "" {
		http.Handle(cfg.baseURL+"/", http.StripPrefix(cfg.baseURL, hfunc))
	} else {
		http.Handle("/", hfunc)
	}

	if cfg.sslCert != "" && cfg.sslKey != "" {
		log.Info("Listen user on: ", cfg.addrUser, " SSL on")
		log.Fatal(http.ListenAndServeTLS(cfg.addrUser, cfg.sslCert, cfg.sslKey, nil))
	} else {
		log.Info("Listen user on: ", cfg.addrUser, " SSL off")
		log.Fatal(http.ListenAndServe(cfg.addrUser, nil))
	}
}
