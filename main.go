package main

import (
	"database/sql"
	"fmt"
	"os"
	"runtime"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/zhaojh329/rttys/config"
	rlog "github.com/zhaojh329/rttys/log"
	"github.com/zhaojh329/rttys/utils"
	"github.com/zhaojh329/rttys/version"

	_ "github.com/go-sql-driver/mysql"
)

func initDb(cfg *config.Config) error {
	db, err := sql.Open("mysql", cfg.DB)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS config(name VARCHAR(512) PRIMARY KEY NOT NULL, value TEXT NOT NULL)")
	if err != nil {
		return err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS account(username VARCHAR(512) PRIMARY KEY NOT NULL, password TEXT NOT NULL, admin INT NOT NULL)")
	if err != nil {
		return err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS device(id VARCHAR(512) PRIMARY KEY NOT NULL, description TEXT NOT NULL, online DATETIME NOT NULL, username TEXT NOT NULL)")

	return err
}

func runRttys(c *cli.Context) {
	rlog.SetPath(c.String("log"))

	cfg := config.Parse(c)

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

	err := initDb(cfg)
	if err != nil {
		log.Error().Msg(err.Error())
		return
	}

	br := newBroker(cfg)
	go br.run()

	listenDevice(br)
	listenDeviceWeb(br)
	httpStart(br)

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
					&cli.StringFlag{
						Name:  "db",
						Value: "rttys:rttys@tcp(localhost)/rttys",
						Usage: "mysql database source",
					},
					&cli.BoolFlag{
						Name:  "local-auth",
						Usage: "need auth for local",
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
