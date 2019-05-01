package pwauth

import (
	"errors"
	"io/ioutil"
	"os"
	"os/user"
	"strings"

	"github.com/GehirnInc/crypt"
	_ "github.com/GehirnInc/crypt/apr1_crypt"
	_ "github.com/GehirnInc/crypt/md5_crypt"
	_ "github.com/GehirnInc/crypt/sha256_crypt"
	_ "github.com/GehirnInc/crypt/sha512_crypt"
)

func getPassword(name string) (string, error) {
	/* Disallow potentially-malicious user names */
	if name == "" || name[0] == '.' || strings.Contains(name, "/") {
		return "", errors.New("Invalid")
	}

	data, err := ioutil.ReadFile("/etc/master.passwd")
	if err != nil {
		return "", err
	}

	for _, l := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(l, name+":") {
			continue
		}

		s := strings.Split(l, ":")

		return s[1], nil
	}

	return "", errors.New("Not found")
}

func auth(username, password string) error {
	if os.Getuid() != 0 {
		return errors.New("Cannot possibly work without effective root")
	}

	if _, err := user.Lookup(username); err != nil {
		return err
	}

	pw, err := getPassword(username)
	if err != nil {
		return err
	}

	c := crypt.NewFromHash(pw)
	return c.Verify(pw, []byte(password))
}
