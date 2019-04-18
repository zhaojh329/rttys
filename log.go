/*
 * Copyright (C) 2017 Jianhui Zhao <jianhuizhao329@gmail.com>
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU Lesser General Public
 * License as published by the Free Software Foundation; either
 * version 2.1 of the License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public
 * License along with this library; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301
 * USA
 */

package main

import (
	"fmt"
	slog "log"
	"os"

	"github.com/mattn/go-isatty"
)

type RttysLog struct {
	file string
}

var log = LogInit()

const logFile = "/var/log/rttys.log"

func (l *RttysLog) Write(b []byte) (n int, err error) {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		return fmt.Fprintf(os.Stderr, "%s", b)
	}

	file, err := os.OpenFile(l.file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return 0, nil
	}
	defer file.Close()

	st, _ := file.Stat()
	if st.Size() > 1024*1024 {
		file.Truncate(0)
	}

	return fmt.Fprintf(file, "%s", b)
}

func LogInit() *slog.Logger {
	return slog.New(&RttysLog{logFile}, "", slog.LstdFlags)
}
