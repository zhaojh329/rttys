package main

import (
	"errors"
	"io/ioutil"
	"strings"
	"syscall"

	"github.com/GehirnInc/crypt"
	_ "github.com/GehirnInc/crypt/apr1_crypt"
	_ "github.com/GehirnInc/crypt/md5_crypt"
	_ "github.com/GehirnInc/crypt/sha256_crypt"
	_ "github.com/GehirnInc/crypt/sha512_crypt"
)

type spwd struct {
	sp_namp string
	sp_pwdp string
}

func getspnam(name string) (*spwd, error) {
	/* Disallow potentially-malicious user names */
	if name == "" || name[0] == '.' || strings.Contains(name, "/") {
		return nil, errors.New("Invalid")
	}

	data, err := ioutil.ReadFile("/etc/shadow")
	if err != nil {
		return nil, err
	}

	for _, l := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(l, name+":") {
			continue
		}

		s := strings.Split(l, ":")

		return &spwd{s[0], s[1]}, nil
	}

	return nil, errors.New("Not found")
}

func checkUser() bool {
	return syscall.Getuid() == 0
}

func login(username, password string) bool {
	sp, _ := getspnam(username)
	if sp == nil {
		return false
	}

	c := crypt.NewFromHash(sp.sp_pwdp)
	return c.Verify(sp.sp_pwdp, []byte(password)) != nil
}
