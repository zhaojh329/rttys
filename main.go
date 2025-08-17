/* SPDX-License-Identifier: MIT */
/*
 * Author: Jianhui Zhao <zhaojh329@gmail.com>
 */

package main

import (
	"context"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/debug"

	xlog "github.com/zhaojh329/rttys/v5/log"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"
)

const RttysVersion = "5.5.2"

var (
	GitCommit = ""
	BuildTime = ""
)

func main() {
	cmd := &cli.Command{
		Name:    "rttys",
		Usage:   "The server side for rtty",
		Version: RttysVersion,
		Flags: []cli.Flag{
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
				Name:    "token",
				Aliases: []string{"t"},
				Usage:   "token to use",
			},
			&cli.StringFlag{
				Name:  "dev-hook-url",
				Usage: "called when the device is connected",
			},
			&cli.StringFlag{
				Name:  "user-hook-url",
				Usage: "called when user accesses /connect/:devid, /cmd/:devid, /web/, or /web2/ APIs",
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
				Name:  "allow-origins",
				Usage: "allow all origins for cross-domain request",
			},
			&cli.StringFlag{
				Name:  "pprof",
				Usage: "enable pprof and listen on specified address (e.g. localhost:6060)",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"V"},
				Usage:   "more detailed log output",
			},
			&cli.StringFlag{
				Name:  "sslcert",
				Usage: "SSL/TLS certificate for device",
			},
			&cli.StringFlag{
				Name:  "sslkey",
				Usage: "SSL/TLS private key for device",
			},
			&cli.StringFlag{
				Name:  "cacert",
				Usage: "CA certificate to verify devices (mTLS)",
			},
		},
		Action: cmdAction,
	}

	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}

func cmdAction(c context.Context, cmd *cli.Command) error {
	defer logPanic()

	switch cmd.String("log-level") {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if cmd.Bool("verbose") {
		xlog.Verbose()
	}

	log.Info().Msg("Go Version: " + runtime.Version())
	log.Info().Msgf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)

	log.Info().Msg("Rttys Version: " + RttysVersion)

	if GitCommit != "" {
		log.Info().Msg("Git Commit: " + GitCommit)
	}

	if BuildTime != "" {
		log.Info().Msg("Build Time: " + BuildTime)
	}

	if runtime.GOOS != "windows" {
		go signalHandle()
	}

	cfg := Config{
		AddrDev:   ":5912",
		AddrUser:  ":5913",
		LocalAuth: true,
	}

	err := cfg.Parse(cmd)
	if err != nil {
		return err
	}

	srv := &RttyServer{cfg: cfg}

	return srv.Run()
}

func logPanic() {
	if r := recover(); r != nil {
		saveCrashLog(r, debug.Stack())
		os.Exit(2)
	}
}

func saveCrashLog(p any, stack []byte) {
	log.Error().Msgf("%v", p)
	log.Error().Msg(string(stack))
}
