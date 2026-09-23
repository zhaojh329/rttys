//go:build windows
// +build windows

/* SPDX-License-Identifier: MIT */
/*
 * Author: Jianhui Zhao <zhaojh329@gmail.com>
 */

package main

import (
	"github.com/rs/zerolog/log"
)

func signalHandle() {
	log.Debug().Msg("Signal handling not supported on Windows")
}
