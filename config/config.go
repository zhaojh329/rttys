package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/kylelemons/go-gypsy/yaml"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

// Config struct
type Config struct {
	AddrDev           string
	AddrUser          string
	AddrHttpProxy     string
	DisableSignUp     bool
	HttpProxyRedirURL string
	HttpProxyPort     int
	SslCert           string
	SslKey            string
	SslCacert         string // mTLS for device
	WebUISslCert      string
	WebUISslKey       string
	Token             string
	DevAuthUrl        string
	WhiteList         map[string]bool
	DB                string
	LocalAuth         bool
	SeparateSslConfig bool
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
	case *bool:
		*opt, _ = strconv.ParseBool(val)
	}
}

// Parse config
func Parse(c *cli.Context) *Config {
	cfg := &Config{
		AddrDev:           c.String("addr-dev"),
		AddrUser:          c.String("addr-user"),
		AddrHttpProxy:     c.String("addr-http-proxy"),
		DisableSignUp:     c.Bool("disable-sign-up"),
		HttpProxyRedirURL: c.String("http-proxy-redir-url"),
		SslCert:           c.String("ssl-cert"),
		SslKey:            c.String("ssl-key"),
		SslCacert:         c.String("ssl-cacert"),
		SeparateSslConfig: c.Bool("separate-ssl-config"),
		WebUISslCert:      c.String("webui-ssl-cert"),
		WebUISslKey:       c.String("webui-ssl-key"),
		Token:             c.String("token"),
		DevAuthUrl:        c.String("dev-auth-url"),
		DB:                c.String("db"),
		LocalAuth:         c.Bool("local-auth"),
	}

	cfg.WhiteList = make(map[string]bool)

	whiteList := c.String("white-list")

	if whiteList == "*" {
		cfg.WhiteList = nil
	} else {
		for _, id := range strings.Fields(whiteList) {
			cfg.WhiteList[id] = true
		}
	}

	yamlCfg, err := yaml.ReadFile(c.String("conf"))
	if err == nil {
		getConfigOpt(yamlCfg, "addr-dev", &cfg.AddrDev)
		getConfigOpt(yamlCfg, "addr-user", &cfg.AddrUser)
		getConfigOpt(yamlCfg, "addr-http-proxy", &cfg.AddrHttpProxy)
		getConfigOpt(yamlCfg, "disable-sign-up", &cfg.DisableSignUp)
		getConfigOpt(yamlCfg, "http-proxy-redir-url", &cfg.HttpProxyRedirURL)
		getConfigOpt(yamlCfg, "ssl-cert", &cfg.SslCert)
		getConfigOpt(yamlCfg, "ssl-key", &cfg.SslKey)
		getConfigOpt(yamlCfg, "ssl-cacert", &cfg.SslCacert)
		getConfigOpt(yamlCfg, "separate-ssl-config", &cfg.SeparateSslConfig)
		if cfg.SeparateSslConfig {
			getConfigOpt(yamlCfg, "webui-ssl-cert", &cfg.WebUISslCert)
			getConfigOpt(yamlCfg, "webui-ssl-key", &cfg.WebUISslKey)
		} else {
			cfg.WebUISslCert = cfg.SslCert
			cfg.WebUISslKey = cfg.SslKey
		}
		getConfigOpt(yamlCfg, "token", &cfg.Token)
		getConfigOpt(yamlCfg, "dev-auth-url", &cfg.DevAuthUrl)
		getConfigOpt(yamlCfg, "db", &cfg.DB)
		getConfigOpt(yamlCfg, "local-auth", &cfg.LocalAuth)
		val, err := yamlCfg.Get("white-list")
		if err == nil {
			if val == "*" || val == "\"*\"" {
				cfg.WhiteList = nil
			} else {
				for _, id := range strings.Fields(val) {
					cfg.WhiteList[id] = true
				}
			}
		}
	}

	if cfg.SslCert != "" && cfg.SslKey != "" {
		_, err := os.Lstat(cfg.SslCert)
		if err != nil {
			log.Error().Msg(err.Error())
			cfg.SslCert = ""
		}

		_, err = os.Lstat(cfg.SslKey)
		if err != nil {
			log.Error().Msg(err.Error())
			cfg.SslKey = ""
		}
	}

	return cfg
}
