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
	"io"
	"os"
	"runtime"
	"time"

	"github.com/kylelemons/go-gypsy/yaml"

	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
)

type rttysConfig struct {
	addr     string
	cert     string
	key      string
	username string
	password string
}

func init() {
	log.AddHook(lfshook.NewHook("/var/log/rttys.log", &log.TextFormatter{}))
}

func main() {
	cfg := parseConfig()

	if !checkUser() && cfg.username == "" {
		log.Error("Operation not permitted. Please start as root or define Username and Password in configuration file")
		os.Exit(1)
	}

	log.Infof("go version: %s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	log.Info("rttys version:", rttys_version())

	br := newBroker()
	go br.run()

	httpStart(br, cfg)
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

func parseConfig() *rttysConfig {
	addr := flag.String("addr", ":5912", "address to listen")
	cert := flag.String("ssl-cert", "", "certFile Path")
	key := flag.String("ssl-key", "", "keyFile Path")
	conf := flag.String("conf", "./rttys.conf", "config file to load")

	flag.Parse()

	cfg := &rttysConfig{}

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
