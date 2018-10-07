package main

import (
	"github.com/robfig/config"
)

func checkUser() bool {
	return true
}

func login(username, password string) bool {
	c, err := config.ReadDefault("rttys.ini")

	if err != nil {
		return true
	}

	u, _ := c.String("login", "username")
	p, _ := c.String("login", "password")

	return username == u && password == p
}
