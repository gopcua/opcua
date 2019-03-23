// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// Package debug provides functions for debug logging.
package debug

import (
	"log"
	"os"
)

// Enable controls whether debug logging is enabled. It is disabled by default.
var Enable bool

// Logger logs the debug messages when debug logging is enabled.
var Logger = log.New(os.Stderr, "debug: ", 0)

// Printf logs the message with Logger.Printf() when debug logging is enabled.
func Printf(format string, args ...interface{}) {
	if !Enable {
		return
	}
	Logger.Printf(format, args...)
}
