package utils

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/howeyc/gopass"
	"github.com/rs/zerolog/log"
)

// GenUniqueID generate a unique ID
func GenUniqueID(extra string) string {
	buf := make([]byte, 20)

	binary.BigEndian.PutUint32(buf, uint32(time.Now().Unix()))
	io.ReadFull(rand.Reader, buf[4:])

	h := md5.New()
	h.Write(buf)
	h.Write([]byte(extra))

	return hex.EncodeToString(h.Sum(nil))
}

// GetMD5 generate md5
func GetMD5(extra string) string {
	h := md5.New()
	h.Write([]byte(extra))
	return hex.EncodeToString(h.Sum(nil))
}

// GenToken generate a token
func GenToken() {
	password, err := gopass.GetPasswdPrompt("Please set a password:", true, os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	token := GenUniqueID(string(password))

	fmt.Println("Your token is:", token)
}
