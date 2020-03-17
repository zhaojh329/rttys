package main

import (
	"flag"
	"github.com/kylelemons/go-gypsy/yaml"
	"github.com/rs/zerolog/log"
	"os"
	"strconv"
	"strings"
)

type RttysConfig struct {
	addrDev      string
	addrUser     string
	sslCert      string
	sslKey       string
	httpUsername string
	httpPassword string
	token        string
	fontSize     int
	whiteList    map[string]bool
}

func getConfigOpt(yamlCfg *yaml.File, name string, opt interface{}) {
	val, err := yamlCfg.Get(name)
	if err != nil {
		return
	}

	switch opt := opt.(type) {
	case *string:
		*opt = val
	case *int:
		*opt, _ = strconv.Atoi(val)
	}
}

func parseConfig() *RttysConfig {
	cfg := &RttysConfig{}

	cfg.whiteList = make(map[string]bool)

	flag.StringVar(&cfg.addrDev, "addr-dev", ":5912", "address to listen device")
	flag.StringVar(&cfg.addrUser, "addr-user", ":5913", "address to listen user")
	flag.StringVar(&cfg.sslCert, "ssl-cert", "", "certFile Path")
	flag.StringVar(&cfg.sslKey, "ssl-key", "", "keyFile Path")
	flag.StringVar(&cfg.httpUsername, "http-username", "", "username for http auth")
	flag.StringVar(&cfg.httpPassword, "http-password", "", "password for http auth")
	flag.StringVar(&cfg.token, "token", "", "token to use")

	conf := flag.String("conf", "./rttys.conf", "config file to load")
	genToken := flag.Bool("gen-token", false, "generate token")

	whiteList := flag.String("white-list", "", "white list(device IDs separated by spaces or *)")

	if *whiteList == "*" {
		cfg.whiteList = nil
	} else {
		for _, id := range strings.Fields(*whiteList) {
			cfg.whiteList[id] = true
		}
	}

	flag.Parse()

	if *genToken {
		genTokenAndExit()
	}

	yamlCfg, err := yaml.ReadFile(*conf)
	if err == nil {
		getConfigOpt(yamlCfg, "addr-dev", &cfg.addrDev)
		getConfigOpt(yamlCfg, "addr-user", &cfg.addrUser)
		getConfigOpt(yamlCfg, "ssl-cert", &cfg.sslCert)
		getConfigOpt(yamlCfg, "ssl-key", &cfg.sslKey)
		getConfigOpt(yamlCfg, "http-username", &cfg.httpUsername)
		getConfigOpt(yamlCfg, "http-password", &cfg.httpPassword)
		getConfigOpt(yamlCfg, "token", &cfg.token)
		getConfigOpt(yamlCfg, "font-size", &cfg.fontSize)

		val, err := yamlCfg.Get("white-list")
		if err == nil {
			if val == "*" || val == "\"*\"" {
				cfg.whiteList = nil
			} else {
				for _, id := range strings.Fields(val) {
					cfg.whiteList[id] = true
				}
			}
		}
	}

	if cfg.fontSize == 0 {
		cfg.fontSize = 16
	}

	if cfg.fontSize < 12 {
		cfg.fontSize = 12
	}

	if cfg.sslCert != "" && cfg.sslKey != "" {
		_, err := os.Lstat(cfg.sslCert)
		if err != nil {
			log.Error().Msg(err.Error())
			cfg.sslCert = ""
		}

		_, err = os.Lstat(cfg.sslKey)
		if err != nil {
			log.Error().Msg(err.Error())
			cfg.sslKey = ""
		}
	}

	return cfg
}
