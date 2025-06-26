package config

import (
	"fmt"
	"strconv"

	"github.com/kylelemons/go-gypsy/yaml"
	"github.com/urfave/cli/v3"
)

// Config struct
type Config struct {
	AddrDev              string
	AddrUser             string
	AddrHttpProxy        string
	HttpProxyRedirURL    string
	HttpProxyRedirDomain string
	HttpProxyPort        int
	Token                string
	DevHookUrl           string
	LocalAuth            bool
	Password             string
	AllowOrigins         bool
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

	getConfigOpt(yamlCfg, "token", &cfg.Token)
	getConfigOpt(yamlCfg, "dev-hook-url", &cfg.DevHookUrl)
	getConfigOpt(yamlCfg, "local-auth", &cfg.LocalAuth)
	getConfigOpt(yamlCfg, "password", &cfg.Password)
	getConfigOpt(yamlCfg, "allow-origins", &cfg.AllowOrigins)

	return nil
}

func getFlagOpt(c *cli.Command, name string, opt any) {
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
func Parse(c *cli.Command) (*Config, error) {
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
	getFlagOpt(c, "dev-hook-url", &cfg.DevHookUrl)
	getFlagOpt(c, "local-auth", &cfg.LocalAuth)
	getFlagOpt(c, "token", &cfg.Token)
	getFlagOpt(c, "password", &cfg.Password)
	getFlagOpt(c, "allow-origins", &cfg.AllowOrigins)

	return cfg, nil
}
