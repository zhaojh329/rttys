/*
 * Copyright (C) 2017 Jianhui Zhao <jianhuizhao329@gmail.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
    "flag"
    "log"
    "fmt"
    "time"
    "strconv"
    "math/rand"
    "net/http"
    "crypto/md5"
    "encoding/hex"
    "encoding/json"
     _ "github.com/zhaojh329/rttys/statik"
    "github.com/rakyll/statik/fs"
)

type DeviceInfo struct {
    ID string `json:"id"`
    Uptime int64 `json:"uptime"`
    Description string `json:"description"`
}

func generateSID(devid string) string {
    md5Ctx := md5.New()
    md5Ctx.Write([]byte(devid + strconv.FormatFloat(rand.Float64(), 'e', 6, 32)))
    cipherStr := md5Ctx.Sum(nil)
    return hex.EncodeToString(cipherStr)
}

func main() {
    port := flag.Int("port", 5912, "http service port")
    cert := flag.String("cert", "", "certFile Path")
    key := flag.String("key", "", "keyFile Path")

    flag.Parse()

    rand.Seed(time.Now().Unix())

    br := newBridge()
    go br.run()

    statikFS, err := fs.New()
    if err != nil {
        log.Fatal(err)
        return
    }

    http.Handle("/", http.FileServer(statikFS))

    http.HandleFunc("/devs", func(w http.ResponseWriter, r *http.Request) {
        devs := make([]DeviceInfo, 0)
        for _, c := range br.devices {
            if c.isDev {
                d := DeviceInfo{c.devid, time.Now().Unix() - c.timestamp, c.description}
                devs = append(devs, d)
            }
        }

        js, _ := json.Marshal(devs)
        fmt.Fprintf(w, "%s", js)
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
