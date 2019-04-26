package main

import (
	"encoding/json"
	"fmt"
	"github.com/rakyll/statik/fs"
	log "github.com/sirupsen/logrus"
	"github.com/zhaojh329/rttys/cache"
	_ "github.com/zhaojh329/rttys/statik"
	"net/http"
	"os"
	"strconv"
	"time"
)

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

func httpLogin(cfg *RttysConfig, username, password string) bool {
	if cfg.username != "" {
		if cfg.username != username {
			return false
		}

		if cfg.password != "" {
			return cfg.password == password
		}

		return true
	}

	return login(username, password)
}

func httpStart(br *Broker, cfg *RttysConfig) {
	httpSessions = cache.New(30*time.Minute, 5*time.Second)

	statikFS, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}

	staticfs := http.FileServer(statikFS)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(br, w, r, cfg)
	})

	http.HandleFunc("/cmd", func(w http.ResponseWriter, r *http.Request) {
		allowOrigin(w)
		serveCmd(br, w, r)
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		username := r.PostFormValue("username")
		password := r.PostFormValue("password")

		if httpLogin(cfg, username, password) {
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

	http.HandleFunc("/devs", func(w http.ResponseWriter, r *http.Request) {
		if !httpAuth(w, r) {
			return
		}

		devs := []DeviceInfo{}

		for id, dev := range br.devices {
			dev := DeviceInfo{id, time.Now().Unix() - dev.timestamp, dev.desc}
			devs = append(devs, dev)
		}

		allowOrigin(w)

		resp, _ := json.Marshal(devs)

		w.Write(resp)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			t := r.URL.Query().Get("t")
			id := r.URL.Query().Get("id")

			if t == "" && id == "" {
				http.Redirect(w, r, "/?t="+strconv.FormatInt(time.Now().Unix(), 10), http.StatusFound)
				return
			}
		}

		staticfs.ServeHTTP(w, r)
	})

	if cfg.sslCert != "" && cfg.sslKey != "" {
		_, err := os.Lstat(cfg.sslCert)
		if err != nil {
			log.Error(err)
			cfg.sslCert = ""
		}

		_, err = os.Lstat(cfg.sslKey)
		if err != nil {
			log.Error(err)
			cfg.sslKey = ""
		}
	}

	if cfg.sslCert != "" && cfg.sslKey != "" {
		log.Info("Listen on: ", cfg.addr, " SSL on")
		log.Fatal(http.ListenAndServeTLS(cfg.addr, cfg.sslCert, cfg.sslKey, nil))
	} else {
		log.Info("Listen on: ", cfg.addr, " SSL off")
		log.Fatal(http.ListenAndServe(cfg.addr, nil))
	}
}
