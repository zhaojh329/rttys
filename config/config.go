package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/kylelemons/go-gypsy/yaml"
	"github.com/urfave/cli/v2"
)

// Config struct
type Config struct {
	AddrDev              string
	AddrUser             string
	AddrHttpProxy        string
	HttpProxyRedirURL    string
	HttpProxyRedirDomain string
	HttpProxyPort        int
	SslCert              string
	SslKey               string
	SslCacert            string // mTLS for device
	WebUISslCert         string
	WebUISslKey          string
	Token                string
	DevAuthUrl           string
	LocalAuth            bool
	SeparateSslConfig    bool
	Password             string
}

func getConfigOpt(yamlCfg *yaml.File, name string, opt any) {
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

func parseYamlCfg(cfg *Config, conf string) error {
	yamlCfg, err := yaml.ReadFile(conf)
	if err != nil {
		return fmt.Errorf(`read config file: %s`, err.Error())
	}

	getConfigOpt(yamlCfg, "addr-dev", &cfg.AddrDev)
	getConfigOpt(yamlCfg, "addr-user", &cfg.AddrUser)
	getConfigOpt(yamlCfg, "addr-http-proxy", &cfg.AddrHttpProxy)
	getConfigOpt(yamlCfg, "http-proxy-redir-url", &cfg.HttpProxyRedirURL)
	getConfigOpt(yamlCfg, "http-proxy-redir-domain", &cfg.HttpProxyRedirDomain)
	getConfigOpt(yamlCfg, "ssl-cert", &cfg.SslCert)
	getConfigOpt(yamlCfg, "ssl-key", &cfg.SslKey)
	getConfigOpt(yamlCfg, "ssl-cacert", &cfg.SslCacert)
	getConfigOpt(yamlCfg, "separate-ssl-config", &cfg.SeparateSslConfig)

	if cfg.SeparateSslConfig {
		getConfigOpt(yamlCfg, "webui-ssl-cert", &cfg.WebUISslCert)
		getConfigOpt(yamlCfg, "webui-ssl-key", &cfg.WebUISslKey)
	}

	getConfigOpt(yamlCfg, "token", &cfg.Token)
	getConfigOpt(yamlCfg, "dev-auth-url", &cfg.DevAuthUrl)
	getConfigOpt(yamlCfg, "local-auth", &cfg.LocalAuth)
	getConfigOpt(yamlCfg, "password", &cfg.Password)

	return nil
}

func getFlagOpt(c *cli.Context, name string, opt any) {
	if !c.IsSet(name) {
		return
	}

	switch opt := opt.(type) {
	case *string:
		*opt = c.String(name)
	case *int:
		*opt = c.Int(name)
	case *bool:
		*opt = c.Bool(name)
	}
}

// Parse config
func Parse(c *cli.Context) (*Config, error) {
	cfg := &Config{
		AddrDev:   ":5912",
		AddrUser:  ":5913",
		LocalAuth: true,
	}

	conf := c.String("conf")
	if conf != "" {
		err := parseYamlCfg(cfg, conf)
		if err != nil {
			return nil, err
		}
	}

	getFlagOpt(c, "addr-dev", &cfg.AddrDev)
	getFlagOpt(c, "addr-user", &cfg.AddrUser)
	getFlagOpt(c, "addr-http-proxy", &cfg.AddrHttpProxy)
	getFlagOpt(c, "http-proxy-redir-url", &cfg.HttpProxyRedirURL)
	getFlagOpt(c, "http-proxy-redir-domain", &cfg.HttpProxyRedirDomain)
	getFlagOpt(c, "dev-auth-url", &cfg.DevAuthUrl)
	getFlagOpt(c, "local-auth", &cfg.LocalAuth)
	getFlagOpt(c, "token", &cfg.Token)
	getFlagOpt(c, "password", &cfg.Password)

	getFlagOpt(c, "ssl-cacert", &cfg.SslCacert)
	getFlagOpt(c, "ssl-cert", &cfg.SslCert)
	getFlagOpt(c, "ssl-key", &cfg.SslKey)
	getFlagOpt(c, "separate-ssl-config", &cfg.SeparateSslConfig)

	if cfg.SeparateSslConfig {
		getFlagOpt(c, "webui-ssl-cert", &cfg.WebUISslCert)
		getFlagOpt(c, "webui-ssl-key", &cfg.WebUISslKey)
	} else {
		cfg.WebUISslCert = cfg.SslCert
		cfg.WebUISslKey = cfg.SslKey
	}

	if cfg.SslCacert != "" {
		if _, err := os.Lstat(cfg.SslCacert); err != nil {
			return nil, fmt.Errorf(`SslCacert "%s" not exist`, cfg.SslCacert)
		}
	}

	if cfg.SslCert != "" {
		if _, err := os.Lstat(cfg.SslCert); err != nil {
			return nil, fmt.Errorf(`SslCert "%s" not exist`, cfg.SslCert)
		}
	}

	if cfg.SslKey != "" {
		if _, err := os.Lstat(cfg.SslKey); err != nil {
			return nil, fmt.Errorf(`SslKey "%s" not exist`, cfg.SslKey)
		}
	}

	if cfg.WebUISslCert != "" {
		if _, err := os.Lstat(cfg.WebUISslCert); err != nil {
			return nil, fmt.Errorf(`WebUISslCert "%s" not exist`, cfg.WebUISslCert)
		}
	}

	if cfg.WebUISslKey != "" {
		if _, err := os.Lstat(cfg.WebUISslKey); err != nil {
			return nil, fmt.Errorf(`WebUISslKey "%s" not exist`, cfg.WebUISslKey)
		}
	}

	return cfg, nil
}
