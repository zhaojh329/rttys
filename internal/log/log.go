/* SPDX-License-Identifier: MIT */
/*
 * Author: Jianhui Zhao <zhaojh329@gmail.com>
 */

package log

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"

	"github.com/dwdcth/consoleEx"
	"github.com/mattn/go-colorable"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		return filepath.Base(file) + ":" + strconv.Itoa(line)
	}

	out := consoleEx.ConsoleWriterEx{Out: colorable.NewColorableStdout()}
	logger := zerolog.New(out).With().Timestamp().Logger()

	log.Logger = logger

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func Verbose() {
	log.Logger = log.Logger.With().Caller().Logger()
}

func RecoverPanic() {
	if r := recover(); r != nil {
		log.Error().Msgf("%v", r)
		log.Error().Msg(string(debug.Stack()))
		os.Exit(2)
	}
}
