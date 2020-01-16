package main

import (
	"flag"
	"github.com/kylelemons/go-gypsy/yaml"
	log "github.com/sirupsen/logrus"
	"os"
)

type RttysConfig struct {
	addrDev  string
	addrUser string
	sslCert  string
	sslKey   string
	username string
	password string
	token    string
	baseURL  string
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

	flag.StringVar(&cfg.addrDev, "addr-dev", ":5912", "address to listen device")
	flag.StringVar(&cfg.addrUser, "addr-user", ":5913", "address to listen user")
	flag.StringVar(&cfg.sslCert, "ssl-cert", "./rttys.crt", "certFile Path")
	flag.StringVar(&cfg.sslKey, "ssl-key", "./rttys.key", "keyFile Path")
	flag.StringVar(&cfg.token, "token", "", "token to use")
	flag.StringVar(&cfg.baseURL, "base-url", "/", "base url to serve on")
	conf := flag.String("conf", "./rttys.conf", "config file to load")
	genToken := flag.Bool("gen-token", false, "generate token")

	flag.Parse()

	if *genToken {
		genTokenAndExit()
	}

	yamlCfg, err := yaml.ReadFile(*conf)
	if err == nil {
		setConfigOpt(yamlCfg, "addr-dev", &cfg.addrDev)
		setConfigOpt(yamlCfg, "addr-user", &cfg.addrUser)
		setConfigOpt(yamlCfg, "ssl-cert", &cfg.sslCert)
		setConfigOpt(yamlCfg, "ssl-key", &cfg.sslKey)
		setConfigOpt(yamlCfg, "username", &cfg.username)
		setConfigOpt(yamlCfg, "password", &cfg.password)
		setConfigOpt(yamlCfg, "token", &cfg.token)
		setConfigOpt(yamlCfg, "base-url", &cfg.baseURL)
	}

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

	return cfg
}
