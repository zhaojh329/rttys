/*
 * MIT License
 *
 * Copyright (c) 2019 Jianhui Zhao <zhaojh329@gmail.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package main

import (
	"fmt"
	"strconv"

	"github.com/kylelemons/go-gypsy/yaml"
	"github.com/urfave/cli/v3"
)

type Config struct {
	AddrDev              string
	AddrUser             string
	AddrHttpProxy        string
	HttpProxyRedirURL    string
	HttpProxyRedirDomain string
	Token                string
	DevHookUrl           string
	UserHookUrl          string
	LocalAuth            bool
	Password             string
	AllowOrigins         bool
}

func (cfg *Config) Parse(c *cli.Command) error {
	conf := c.String("conf")
	if conf != "" {
		err := parseYamlCfg(cfg, conf)
		if err != nil {
			return err
		}
	}

	getFlagOpt(c, "addr-dev", &cfg.AddrDev)
	getFlagOpt(c, "addr-user", &cfg.AddrUser)
	getFlagOpt(c, "addr-http-proxy", &cfg.AddrHttpProxy)
	getFlagOpt(c, "http-proxy-redir-url", &cfg.HttpProxyRedirURL)
	getFlagOpt(c, "http-proxy-redir-domain", &cfg.HttpProxyRedirDomain)
	getFlagOpt(c, "dev-hook-url", &cfg.DevHookUrl)
	getFlagOpt(c, "user-hook-url", &cfg.UserHookUrl)
	getFlagOpt(c, "local-auth", &cfg.LocalAuth)
	getFlagOpt(c, "token", &cfg.Token)
	getFlagOpt(c, "password", &cfg.Password)
	getFlagOpt(c, "allow-origins", &cfg.AllowOrigins)

	return nil
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
	getConfigOpt(yamlCfg, "user-hook-url", &cfg.UserHookUrl)
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
