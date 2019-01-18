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

package rlog

import (
	"fmt"
	"log"
	"os"

	"github.com/mattn/go-isatty"
)

type RttyLog struct {
	file string
}

const LOG_FILE = "/var/log/rtty.log"

var rLog *log.Logger

func (l *RttyLog) Write(b []byte) (n int, err error) {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		fmt.Fprintf(os.Stderr, "%s", b)
		return 0, nil
	}

	if l.file == "" {
		return 0, nil
	}

	file, err := os.OpenFile(l.file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return 0, nil
	}

	st, _ := file.Stat()
	if st.Size() > 1024*1024 {
		file.Truncate(0)
	}

	defer file.Close()

	fmt.Fprintf(file, "%s", b)

	return 0, nil
}

func init() {
	rLog = log.New(&RttyLog{LOG_FILE}, "", log.LstdFlags)
}

func Print(v ...interface{}) {
	rLog.Print(v...)
}

func Println(v ...interface{}) {
	rLog.Println(v...)
}

func Printf(format string, v ...interface{}) {
	rLog.Printf(format, v...)
}

func Fatal(v ...interface{}) {
	rLog.Fatal(v...)
}
