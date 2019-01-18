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

	"github.com/kylelemons/go-gypsy/yaml"
	"github.com/rakyll/statik/fs"
	_ "github.com/zhaojh329/rttys/statik"
)

type HttpSession struct {
	active time.Duration
}

type rttysConfig struct {
	addr     string
	cert     string
	key      string
	username string
	password string
}

const MAX_SESSION_TIME = 30 * time.Minute

var log = logInit()
var hsMutex sync.Mutex
var httpSessions = make(map[string]*HttpSession)

func main() {
	cfg := parseConfig()

	if !checkUser() {
		log.Println("Operation not permitted")
		os.Exit(1)
	}

	log.Printf("go version: %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	log.Println("rttys version:", rttys_version())

	br := newBroker()
	go br.run()

	statikFS, err := fs.New()
	if err != nil {
		log.Fatal(err)
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
			sid := genUniqueID("http")
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
		for id, dev := range br.devices {
			devs += fmt.Sprintf(`%s{"id":"%s","uptime":%d,"description":"%s"}`,
				comma, id, time.Now().Unix()-dev.timestamp, dev.desc)
			if comma == "" {
				comma = ","
			}
		}

		devs += "]"

		allowOrigin(w)

		io.WriteString(w, devs)
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
		log.Println("Listen on: ", cfg.addr, "SSL on")
		log.Fatal(http.ListenAndServeTLS(cfg.addr, cfg.cert, cfg.key, nil))
	} else {
		log.Println("Listen on: ", cfg.addr, "SSL off")
		log.Fatal(http.ListenAndServe(cfg.addr, nil))
	}
}

func allowOrigin(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json")
}

func genUniqueID(extra string) string {
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
	addr := flag.String("addr", ":5912", "address to listen")
	cert := flag.String("ssl-cert", "", "certFile Path")
	key := flag.String("ssl-key", "", "keyFile Path")
	conf := flag.String("conf", "./rttys.conf", "config file to load")

	flag.Parse()

	cfg := rttysConfig{}

	config, _ := yaml.ReadFile(*conf)
	if config != nil {
		cfg.addr, _ = config.Get("addr")
		cfg.cert, _ = config.Get("ssl-cert")
		cfg.key, _ = config.Get("ssl-key")
		cfg.username, _ = config.Get("username")
		cfg.password, _ = config.Get("password")
	}

	if cfg.addr == "" {
		cfg.addr = *addr
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
