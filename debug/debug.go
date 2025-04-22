// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// Package debug provides functions for debug logging.
package debug

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"slices"
	"strings"
)

// Flags contains the debug flags set by OPC_DEBUG.
//
//   - codec : print detailed debugging information when encoding/decoding
var Flags = os.Getenv("OPC_DEBUG")

// Enable controls whether debug logging is enabled. It is disabled by default.
var Enable bool = FlagSet("debug")

// Logger logs the debug messages when debug logging is enabled.
var Logger = log.New(os.Stderr, "debug: ", 0)

// PrefixLogger returns a new debug logger when debug logging is enabled.
// Otherwise, a discarding logger is returned.
func NewPrefixLogger(format string, args ...interface{}) *log.Logger {
	if !Enable {
		return log.New(io.Discard, "", 0)
	}
	return log.New(os.Stderr, "debug: "+fmt.Sprintf(format, args...), 0)
}

// Printf logs the message with Logger.Printf() when debug logging is enabled.
func Printf(format string, args ...interface{}) {
	if !Enable {
		return
	}

	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}

	short := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			break
		}
	}
	file = short

	prefix := fmt.Sprintf(" %v:%v ", file, line)
	Logger.Printf(prefix+format, args...)

	//Logger.Printf(format, args...)
}

// ToJSON returns the JSON representation of v when debug logging
// is enabled.
func ToJSON(v interface{}) string {
	if !Enable {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

// FlagSet returns true if the OPC_DEBUG environment variable contains the
// given flag.
func FlagSet(name string) bool {
	return slices.Contains(strings.Fields(Flags), name)
}
