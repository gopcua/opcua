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
//
// It defaults to writing to os.Stderr but, since it is an exported
// *log.Logger, applications can redirect it at runtime with
// Logger.SetOutput(w) (or replace it entirely, e.g. Logger = log.New(...)),
// for example to route opcua debug logs through the app's own logging
// framework. NewPrefixLogger honors the same destination since it derives
// its writer from Logger.Writer().
var Logger = log.New(os.Stderr, "debug: ", 0)

// PrefixLogger returns a new debug logger when debug logging is enabled.
// Otherwise, a discarding logger is returned.
func NewPrefixLogger(format string, args ...interface{}) *log.Logger {
	if !Enable {
		return log.New(io.Discard, "", 0)
	}
	// Logger is exported and therefore assignable, so it can be nil. Fall back
	// to os.Stderr in that case, which is where this function wrote before it
	// derived its writer from Logger.
	w := io.Writer(os.Stderr)
	if Logger != nil {
		w = Logger.Writer()
	}
	return log.New(w, "debug: "+fmt.Sprintf(format, args...), 0)
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
