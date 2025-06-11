package main

import (
	"fmt"
	"os"
	"runtime"

	"rttys/config"
	"rttys/utils"
	"rttys/version"

	xlog "rttys/log"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

func runRttys(c *cli.Context) error {
	xlog.SetPath(c.String("log"))

	switch c.String("log-level") {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if c.Bool("verbose") {
		xlog.Verbose()
	}

	cfg, err := config.Parse(c)
	if err != nil {
		return err
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
	listenHttpProxy(br)
	apiStart(br)

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
						Name:  "log-level",
						Value: "info",
						Usage: "log level(debug, info, warn, error)",
					},
					&cli.StringFlag{
						Name:    "conf",
						Aliases: []string{"c"},
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
						Name:  "addr-http-proxy",
						Usage: "address to listen for HTTP proxy (default auto)",
					},
					&cli.StringFlag{
						Name:  "http-proxy-redir-url",
						Usage: "url to redirect for HTTP proxy",
					},
					&cli.StringFlag{
						Name:  "http-proxy-redir-domain",
						Usage: "domain for HTTP proxy set cookie",
					},
					&cli.StringFlag{
						Name:  "ssl-cert",
						Usage: "ssl cert file Path",
					},
					&cli.StringFlag{
						Name:  "ssl-key",
						Usage: "ssl key file Path",
					},
					&cli.StringFlag{
						Name:  "ssl-cacert",
						Usage: "mtls CA storage in PEM file Path",
					},
					&cli.StringFlag{
						Name:    "token",
						Aliases: []string{"t"},
						Usage:   "token to use",
					},
					&cli.StringFlag{
						Name:  "dev-auth-url",
						Usage: "using device auth url instead of token",
					},
					&cli.StringFlag{
						Name:  "white-list",
						Usage: "white list(device IDs separated by spaces or *)",
					},
					&cli.BoolFlag{
						Name:  "local-auth",
						Value: true,
						Usage: "need auth for local",
					},
					&cli.StringFlag{
						Name:  "password",
						Usage: "web management password",
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Aliases: []string{"V"},
						Usage:   "more detailed output",
					},
				},
				Action: func(c *cli.Context) error {
					return runRttys(c)
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
			return c.App.Command("run").Run(c)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
