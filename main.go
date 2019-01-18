/*
 * Copyright (C) 2017 Jianhui Zhao <jianhuizhao329@gmail.com>
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU Lesser General Public
 * License as published by the Free Software Foundation; either
 * version 2.1 of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public
 * License along with this library; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301
 * USA
 */

package main

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/zhaojh329/rttys/internal/rlog"

	"github.com/kylelemons/go-gypsy/yaml"
	"github.com/rakyll/statik/fs"
	_ "github.com/zhaojh329/rttys/statik"
)

const MAX_SESSION_TIME = 30 * time.Minute

type HttpSession struct {
	active time.Duration
}

type rttysConfig struct {
	port     int
	cert     string
	key      string
	username string
	password string
}

func allowOrigin(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")
}

var hsMutex sync.Mutex
var httpSessions = make(map[string]*HttpSession)

func UniqueId(extra string) string {
	buf := make([]byte, 20)

	binary.BigEndian.PutUint32(buf, uint32(time.Now().Unix()))
	io.ReadFull(rand.Reader, buf[4:])

	h := md5.New()
	h.Write(buf)
	h.Write([]byte(extra))

	return hex.EncodeToString(h.Sum(nil))
}

func cleanHttpSession() {
	defer hsMutex.Unlock()

	hsMutex.Lock()
	for sid, s := range httpSessions {
		s.active = s.active - time.Second
		if s.active == 0 {
			delete(httpSessions, sid)
		}
	}
	time.AfterFunc(1*time.Second, cleanHttpSession)
}

func httpAuth(w http.ResponseWriter, r *http.Request) bool {
	c, err := r.Cookie("sid")
	if err != nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return false
	}

	defer hsMutex.Unlock()

	hsMutex.Lock()

	s, ok := httpSessions[c.Value]
	if !ok {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return false
	}

	s.active = MAX_SESSION_TIME

	return true
}

func parseConfig() rttysConfig {
	port := flag.Int("port", 5912, "http service port")
	cert := flag.String("ssl-cert", "", "certFile Path")
	key := flag.String("ssl-key", "", "keyFile Path")
	conf := flag.String("conf", "./rttys.conf", "config file to load")

	flag.Parse()

	cfg := rttysConfig{}

	config, _ := yaml.ReadFile(*conf)
	if config != nil {
		port, _ := config.GetInt("port")
		cfg.port = int(port)
		cfg.cert, _ = config.Get("ssl-cert")
		cfg.key, _ = config.Get("ssl-key")
		cfg.username, _ = config.Get("username")
		cfg.password, _ = config.Get("password")
	}

	if cfg.port == 0 {
		cfg.port = *port
	}

	if cfg.cert == "" {
		cfg.cert = *cert
	}

	if cfg.key == "" {
		cfg.key = *key
	}

	return cfg
}

func httpLogin(cfg rttysConfig, username, password string) bool {
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

func main() {
	cfg := parseConfig()

	if !checkUser() {
		rlog.Println("Operation not permitted")
		os.Exit(1)
	}

	rlog.Printf("go version: %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	rlog.Println("rttys version:", rttys_version())

	br := newBroker()
	go br.run()

	statikFS, err := fs.New()
	if err != nil {
		rlog.Fatal(err)
		return
	}

	staticfs := http.FileServer(statikFS)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(br, w, r)
	})

	http.HandleFunc("/cmd", func(w http.ResponseWriter, r *http.Request) {
		allowOrigin(w)
		serveCmd(br, w, r)
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		username := r.PostFormValue("username")
		password := r.PostFormValue("password")

		if httpLogin(cfg, username, password) {
			sid := UniqueId("http")
			cookie := http.Cookie{
				Name:     "sid",
				Value:    sid,
				HttpOnly: true,
			}

			hsMutex.Lock()
			httpSessions[sid] = &HttpSession{
				active: MAX_SESSION_TIME,
			}
			hsMutex.Unlock()

			w.Header().Set("Set-Cookie", cookie.String())
			fmt.Fprint(w, sid)
			return
		}

		http.Error(w, "Forbidden", http.StatusForbidden)
	})

	http.HandleFunc("/devs", func(w http.ResponseWriter, r *http.Request) {
		if !httpAuth(w, r) {
			return
		}

		devs := "["
		comma := ""
		for _, c := range br.devices {
			if c.isDev {
				devs += fmt.Sprintf(`%s{"id":"%s","uptime":%d,"description":"%s"}`,
					comma, c.devid, time.Now().Unix()-c.timestamp, c.desc)
				if comma == "" {
					comma = ","
				}
			}
		}

		devs += "]"

		allowOrigin(w)

		w.Write([]byte(devs))
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

	if cfg.cert != "" && cfg.key != "" {
		rlog.Println("Listen on: ", cfg.port, "SSL on")
		rlog.Fatal(http.ListenAndServeTLS(":"+strconv.Itoa(cfg.port), cfg.cert, cfg.key, nil))
	} else {
		rlog.Println("Listen on: ", cfg.port, "SSL off")
		rlog.Fatal(http.ListenAndServe(":"+strconv.Itoa(cfg.port), nil))
	}
}
