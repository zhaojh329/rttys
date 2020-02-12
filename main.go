package main

import (
	"flag"
	"github.com/dwdcth/consoleEx"
	"github.com/mattn/go-colorable"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/zhaojh329/rttys/version"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type LogFileHook struct {
	err  error
	path string
}

var logFile = &LogFileHook{}

func (h *LogFileHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if h.err != nil {
		return
	}

	f, err := os.OpenFile(h.path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		h.err = err
		log.Fatal().Msg(err.Error())
		return
	}
	defer f.Close()

	f.WriteString(zerolog.TimestampFunc().Format(zerolog.TimeFieldFormat) + " |")
	f.WriteString(strings.ToUpper(level.String()) + "| ")

	_, file, line, ok := runtime.Caller(3)
	if ok {
		f.WriteString(zerolog.CallerMarshalFunc(file, line) + " |")
	}

	f.WriteString(msg)
	f.WriteString("\n")
}

func init() {
	zerolog.CallerMarshalFunc = func(file string, line int) string {
		return filepath.Base(file) + ":" + strconv.Itoa(line)
	}

	out := consoleEx.ConsoleWriterEx{Out: colorable.NewColorableStdout()}
	logger := zerolog.New(out).With().Caller().Timestamp().Logger()

	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		logger = logger.Hook(logFile)
	}

	log.Logger = logger
}

func main() {
	if runtime.GOOS == "windows" {
		flag.StringVar(&logFile.path, "log", "rttys.log", "log file path")
	} else {
		flag.StringVar(&logFile.path, "log", "/var/log/rttys.log", "log file path")
	}

	cfg := parseConfig()

	if cfg.httpUsername == "" {
		log.Fatal().Msg("You must configure the http username by commandline or config file")
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

	br := newBroker(cfg.token)
	go br.run()

	go listenDevice(br, cfg)
	go httpStart(br, cfg)

	for {
		time.Sleep(time.Second)
	}
}
