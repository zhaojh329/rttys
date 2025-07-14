//go:build windows
// +build windows

package main

import (
	"github.com/rs/zerolog/log"
)

func signalHandle() {
	log.Debug().Msg("Signal handling not supported on Windows")
}
