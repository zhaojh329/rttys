/* SPDX-License-Identifier: MIT */
/*
 * Author: Jianhui Zhao <zhaojh329@gmail.com>
 */

package main

import (
	"fmt"

	"github.com/kylelemons/go-gypsy/yaml"
	"github.com/urfave/cli/v3"
)

type Config struct {
	AddrDev       string
	AddrUser      string
	AddrHttpProxy string

	HttpProxyRedirURL    string
	HttpProxyRedirDomain string

	Token        string
	DevHookUrl   string
	UserHookUrl  string
	LocalAuth    bool
	Password     string
	AllowOrigins bool

	PprofAddr string

	SslCert string
	SslKey  string
	CaCert  string

	KCP            bool
	KcpNodelay     bool
	KcpInterval    int
	KcpResend      int
	KcpNc          bool
	KcpSndwnd      int
	KcpRcvwnd      int
	KcpMtu         int
	KcpPassword    string
	KcpDataShard   int
	KcpParityShard int
}

func (cfg *Config) Parse(c *cli.Command) error {
	var yamlCfg *yaml.File
	var err error

	conf := c.String("conf")
	if conf != "" {
		yamlCfg, err = yaml.ReadFile(conf)
		if err != nil {
			return fmt.Errorf(`read config file: %s`, err.Error())
		}

	}

	fields := map[string]any{
		"addr-dev":        &cfg.AddrDev,
		"addr-user":       &cfg.AddrUser,
		"addr-http-proxy": &cfg.AddrHttpProxy,

		"http-proxy-redir-url":    &cfg.HttpProxyRedirURL,
		"http-proxy-redir-domain": &cfg.HttpProxyRedirDomain,

		"token":         &cfg.Token,
		"dev-hook-url":  &cfg.DevHookUrl,
		"user-hook-url": &cfg.UserHookUrl,
		"local-auth":    &cfg.LocalAuth,
		"password":      &cfg.Password,
		"allow-origins": &cfg.AllowOrigins,

		"pprof": &cfg.PprofAddr,

		"sslcert": &cfg.SslCert,
		"sslkey":  &cfg.SslKey,
		"cacert":  &cfg.CaCert,

		"kcp":              &cfg.KCP,
		"kcp-nodelay":      &cfg.KcpNodelay,
		"kcp-interval":     &cfg.KcpInterval,
		"kcp-resend":       &cfg.KcpResend,
		"kcp-nc":           &cfg.KcpNc,
		"kcp-sndwnd":       &cfg.KcpSndwnd,
		"kcp-rcvwnd":       &cfg.KcpRcvwnd,
		"kcp-mtu":          &cfg.KcpMtu,
		"kcp-key":          &cfg.KcpPassword,
		"kcp-data-shard":   &cfg.KcpDataShard,
		"kcp-parity-shard": &cfg.KcpParityShard,
	}

	for name, opt := range fields {
		if yamlCfg != nil {
			if err = getConfigOpt(yamlCfg, name, opt); err != nil {
				return err
			}
		}

		getFlagOpt(c, name, opt)
	}

	return nil
}

func getConfigOpt(yamlCfg *yaml.File, name string, opt any) error {
	var err error

	switch opt := opt.(type) {
	case *string:
		var val string
		val, err = yamlCfg.Get(name)
		if err == nil {
			*opt = val
		}
	case *int:
		var val int64
		val, err = yamlCfg.GetInt(name)
		if err == nil {
			*opt = int(val)
		}
	case *bool:
		var val bool
		val, err = yamlCfg.GetBool(name)
		if err == nil {
			*opt = val
		}
	default:
		return fmt.Errorf("unsupported type for option %s", name)
	}

	if err != nil {
		if _, ok := err.(*yaml.NodeNotFound); ok {
			return nil
		}
		return fmt.Errorf(`invalud "%s": %w`, name, err)
	}

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
