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
    "os"
    "fmt"
    "sync"
    "flag"
    "time"
    "runtime"
    "strconv"
    "crypto/md5"
    "math/rand"
    "net/http"
    "encoding/hex"
    "encoding/json"
    _ "github.com/zhaojh329/rttys/statik"
    "github.com/rakyll/statik/fs"
)

const MAX_SESSION_TIME = 30 * time.Minute

type DeviceInfo struct {
    ID string `json:"id"`
    Uptime int64 `json:"uptime"`
    Description string `json:"description"`
}

type HttpSession struct {
    active time.Duration
}

func allowOrigin(w http.ResponseWriter) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
    w.Header().Set("content-type", "application/json")
}

var hsMutex sync.Mutex
var httpSessions = make(map[string]*HttpSession)

func cleanHttpSession() {
    defer hsMutex.Unlock()

    hsMutex.Lock()
    for sid, s := range httpSessions {
        s.active = s.active - time.Second
        if s.active == 0 {
            delete(httpSessions, sid)
        }
    }
    time.AfterFunc(1 * time.Second, cleanHttpSession)
}

func generateHttpSID(username, password string) string {
    md5Ctx := md5.New()
    md5Ctx.Write([]byte(username + strconv.FormatFloat(rand.Float64(), 'e', 6, 32) + password))
    cipherStr := md5Ctx.Sum(nil)
    return hex.EncodeToString(cipherStr)
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

func main() {
    port := flag.Int("port", 5912, "http service port")
    cert := flag.String("cert", "", "certFile Path")
    key := flag.String("key", "", "keyFile Path")

    if !checkUser() {
        rlog.Println("Operation not permitted")
        os.Exit(1)
    }

    flag.Parse()

    rand.Seed(time.Now().Unix())

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

        if login(username, password) {
            sid := generateHttpSID(username, password)
            cookie := http.Cookie{
                Name: "sid",
                Value: sid,
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

        devs := make([]DeviceInfo, 0)
        for _, c := range br.devices {
            if c.isDev {
                d := DeviceInfo{c.devid, time.Now().Unix() - c.timestamp, c.description}
                devs = append(devs, d)
            }
        }

        allowOrigin(w)

        rsp, _ := json.Marshal(devs)
        w.Write(rsp)
    })    

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/" {
            t := r.URL.Query().Get("t")
            id := r.URL.Query().Get("id")

            if t == "" && id == "" {
                http.Redirect(w, r, "/?t=" + strconv.FormatInt(time.Now().Unix(), 10), http.StatusFound)
                return
            }
        }

        staticfs.ServeHTTP(w, r)
    })

    if *cert != "" && *key != "" {
        rlog.Println("Listen on: ", *port, "SSL on")
        rlog.Fatal(http.ListenAndServeTLS(":" + strconv.Itoa(*port), *cert, *key, nil))
    } else {
        rlog.Println("Listen on: ", *port, "SSL off")
        rlog.Fatal(http.ListenAndServe(":" + strconv.Itoa(*port), nil))
    }
}
