package main

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/kylelemons/go-gypsy/yaml"
	"github.com/zhaojh329/rttys/version"

	"github.com/howeyc/gopass"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
)

type RttysConfig struct {
	addr     string
	sslCert  string
	sslKey   string
	username string
	password string
	token    string
	baseURL  string
}

func init() {
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		return
	}
	log.AddHook(lfshook.NewHook("/var/log/rttys.log", &log.TextFormatter{}))
}

func main() {
	cfg := parseConfig()

	if os.Getuid() > 0 && cfg.username == "" {
		log.Error("Operation not permitted. Please start as root or define Username and Password in configuration file")
		os.Exit(1)
	}

	log.Info("Go Version: ", runtime.Version())
	log.Info("Go OS/Arch: ", runtime.GOOS, "/", runtime.GOARCH)

	log.Info("Rttys Version: ", version.Version())
	log.Info("Git Commit: ", version.GitCommit())
	log.Info("Build Time: ", version.BuildTime())

	br := newBroker()
	go br.run()

	httpStart(br, cfg)
}

func genUniqueID(extra string) string {
	buf := make([]byte, 20)

	binary.BigEndian.PutUint32(buf, uint32(time.Now().Unix()))
	io.ReadFull(rand.Reader, buf[4:])

	h := md5.New()
	h.Write(buf)
	h.Write([]byte(extra))

	return hex.EncodeToString(h.Sum(nil))
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

	flag.StringVar(&cfg.addr, "addr", ":5912", "address to listen")
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
		setConfigOpt(yamlCfg, "addr", &cfg.addr)
		setConfigOpt(yamlCfg, "ssl-cert", &cfg.sslCert)
		setConfigOpt(yamlCfg, "ssl-key", &cfg.sslKey)
		setConfigOpt(yamlCfg, "username", &cfg.username)
		setConfigOpt(yamlCfg, "password", &cfg.password)
		setConfigOpt(yamlCfg, "token", &cfg.token)
		setConfigOpt(yamlCfg, "base-url", &cfg.baseURL)
	}

	return cfg
}

func genTokenAndExit() {
	password, err := gopass.GetPasswdPrompt("Please set a password:", true, os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}

	token := genUniqueID(string(password))

	fmt.Println("Your token is:", token)

	os.Exit(0)
}
