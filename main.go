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
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/kylelemons/go-gypsy/yaml"

	"github.com/howeyc/gopass"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
)

type RttysConfig struct {
	addr     string
	sslCert  string
	sslKey   string
	username string
	password string
	token    string
}

func init() {
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		return
	}
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

func setConfigOpt(yamlCfg *yaml.File, name string, opt *string) {
	val, err := yamlCfg.Get(name)
	if err != nil {
		return
	}
	*opt = val
}

func parseConfig() *RttysConfig {
	cfg := &RttysConfig{}

	flag.StringVar(&cfg.addr, "addr", ":5912", "address to listen")
	flag.StringVar(&cfg.sslCert, "ssl-cert", "./rttys.crt", "certFile Path")
	flag.StringVar(&cfg.sslKey, "ssl-key", "./rttys.key", "keyFile Path")
	flag.StringVar(&cfg.token, "token", "", "token to use")
	conf := flag.String("conf", "./rttys.conf", "config file to load")
	genToken := flag.Bool("gen-token", false, "generate token")

	flag.Parse()

	if *genToken {
		genTokenAndExit()
	}

	yamlCfg, err := yaml.ReadFile(*conf)
	if err == nil {
		setConfigOpt(yamlCfg, "addr", &cfg.addr)
		setConfigOpt(yamlCfg, "ssl-cert", &cfg.sslCert)
		setConfigOpt(yamlCfg, "ssl-key", &cfg.sslKey)
		setConfigOpt(yamlCfg, "username", &cfg.username)
		setConfigOpt(yamlCfg, "password", &cfg.password)
		setConfigOpt(yamlCfg, "token", &cfg.token)
	}

	return cfg
}

func genTokenAndExit() {
	password, err := gopass.GetPasswdPrompt("Please set a password:", true, os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}

	token := genUniqueID(string(password))

	fmt.Println("Your token is:", token)

	os.Exit(0)
}
