package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/zhaojh329/rttys/config"
	rlog "github.com/zhaojh329/rttys/log"
	"github.com/zhaojh329/rttys/utils"
	"github.com/zhaojh329/rttys/version"
)

func runRttys(c *cli.Context) {
	rlog.SetPath(c.String("log"))

	cfg := config.Parse(c)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR1)

	if cfg.HTTPUsername == "" {
		fmt.Println("You must configure the http username by commandline or config file")
		os.Exit(1)
	}

	log.Info().Msg("Go Version: " + runtime.Version())
	log.Info().Msgf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)

	log.Info().Msg("Rttys Version: " + version.Version())

	gitCommit := version.GitCommit()
	buildTime := version.BuildTime()

	if gitCommit != "" {
		log.Info().Msg("Git Commit: " + version.GitCommit())
	}

	if buildTime != "" {
		log.Info().Msg("Build Time: " + version.BuildTime())
	}

	br := newBroker(cfg)
	go br.run()

	listenDevice(br)
	listenDeviceWeb(br)
	httpStart(br)

	go func() {
		for {
			s := <-sigs
			switch s {
			case syscall.SIGUSR1:
				if br.devCertPool != nil {
					log.Info().Msg("Reload certs for mTLS")
					caCert, err := ioutil.ReadFile(cfg.SslCacert)
					if err != nil {
						log.Info().Msgf("mTLS update faled: %s", err.Error())
					} else {
						br.devCertPool.AppendCertsFromPEM(caCert)
					}
				} else {
					log.Warn().Msg("Reload certs failed: mTLS not used")
				}
			}
		}
	}()

	select {}
}

func main() {
	defaultLogPath := "/var/log/rttys.log"
	if runtime.GOOS == "windows" {
		defaultLogPath = "rttys.log"
	}

	app := &cli.App{
		Name:    "rttys",
		Usage:   "The server side for rtty",
		Version: version.Version(),
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Run rttys",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "log",
						Value: defaultLogPath,
						Usage: "log file path",
					},
					&cli.StringFlag{
						Name:    "conf",
						Aliases: []string{"c"},
						Value:   "./rttys.conf",
						Usage:   "config file to load",
					},
					&cli.StringFlag{
						Name:  "addr-dev",
						Value: ":5912",
						Usage: "address to listen device",
					},
					&cli.StringFlag{
						Name:  "addr-user",
						Value: ":5913",
						Usage: "address to listen user",
					},
					&cli.StringFlag{
						Name:  "addr-web",
						Value: ":5914",
						Usage: "address to listen for access device's web",
					},
					&cli.StringFlag{
						Name:  "web-redir-url",
						Value: "",
						Usage: "url to redirect for access device's web",
					},
					&cli.StringFlag{
						Name:  "ssl-cert",
						Value: "",
						Usage: "ssl cert file Path",
					},
					&cli.StringFlag{
						Name:  "ssl-key",
						Value: "",
						Usage: "ssl key file Path",
					},
					&cli.StringFlag{
						Name:  "ssl-cacert",
						Value: "",
						Usage: "mtls CA storage in PEM file Path",
					},
					&cli.StringFlag{
						Name:  "http-username",
						Value: "",
						Usage: "username for http auth",
					},
					&cli.StringFlag{
						Name:  "http-password",
						Value: "",
						Usage: "password for http auth",
					},
					&cli.StringFlag{
						Name:    "token",
						Aliases: []string{"t"},
						Value:   "",
						Usage:   "token to use",
					},
					&cli.StringFlag{
						Name:  "white-list",
						Value: "",
						Usage: "white list(device IDs separated by spaces or *)",
					},
				},
				Action: func(c *cli.Context) error {
					runRttys(c)
					return nil
				},
			},
			{
				Name:  "token",
				Usage: "Generate a token",
				Action: func(c *cli.Context) error {
					utils.GenToken()
					return nil
				},
			},
		},
		Action: func(c *cli.Context) error {
			c.App.Command("run").Run(c)
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
