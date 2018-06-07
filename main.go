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
    "flag"
    "log"
    "time"
    "strconv"
    "math/rand"
    "net/http"
    "encoding/json"
    _ "github.com/zhaojh329/rttys/statik"
    "github.com/rakyll/statik/fs"
)

type DeviceInfo struct {
    ID string `json:"id"`
    Uptime int64 `json:"uptime"`
    Description string `json:"description"`
}

func main() {
    port := flag.Int("port", 5912, "http service port")
    cert := flag.String("cert", "", "certFile Path")
    key := flag.String("key", "", "keyFile Path")

    flag.Parse()

    rand.Seed(time.Now().Unix())

    log.Println("rttys version:", rttys_version())

    br := newBroker()
    go br.run()

    statikFS, err := fs.New()
    if err != nil {
        log.Fatal(err)
        return
    }

    staticfs := http.FileServer(statikFS)
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

    http.HandleFunc("/devs", func(w http.ResponseWriter, r *http.Request) {
        devs := make([]DeviceInfo, 0)
        for _, c := range br.devices {
            if c.isDev {
                d := DeviceInfo{c.devid, time.Now().Unix() - c.timestamp, c.description}
                devs = append(devs, d)
            }
        }

        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
        w.Header().Set("content-type", "application/json")

        js, _ := json.Marshal(devs)
        w.Write(js)
    })

    http.HandleFunc("/cmd", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
        w.Header().Set("content-type", "application/json")

        serveCmd(br, w, r)
    })

    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        serveWs(br, w, r)
    })

    if *cert != "" && *key != "" {
        log.Println("Listen on: ", *port, "SSL on")
        log.Fatal(http.ListenAndServeTLS(":" + strconv.Itoa(*port), *cert, *key, nil))
    } else {
        log.Println("Listen on: ", *port, "SSL off")
        log.Fatal(http.ListenAndServe(":" + strconv.Itoa(*port), nil))
    }
}
