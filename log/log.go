package log

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/dwdcth/consoleEx"
	"github.com/mattn/go-colorable"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"
)

type logFileHook struct {
	err  error
	path string
}

var logFile = &logFileHook{}

func (h *logFileHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
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

	pc, file, line, ok := runtime.Caller(3)
	if ok {
		f.WriteString(zerolog.CallerMarshalFunc(pc, file, line) + " |")
	}

	f.WriteString(msg)
	f.WriteString("\n")
}

func init() {
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		return filepath.Base(file) + ":" + strconv.Itoa(line)
	}

	out := consoleEx.ConsoleWriterEx{Out: colorable.NewColorableStdout()}
	logger := zerolog.New(out).With().Timestamp().Logger()

	if !term.IsTerminal(int(os.Stdout.Fd())) {
		logger = logger.Hook(logFile)
	}

	log.Logger = logger

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

// SetPath set the log file path
func SetPath(path string) {
	logFile.path = path
}

func Verbose() {
	log.Logger = log.Logger.With().Caller().Logger()
}
