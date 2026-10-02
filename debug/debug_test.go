// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package debug

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestLoggerSetOutputRedirectsPrintf ensures that redirecting Logger's
// output via the exported Logger.SetOutput reroutes Printf away from the
// default os.Stderr destination, so that applications can capture/redirect
// opcua debug logs into their own logging framework.
func TestLoggerSetOutputRedirectsPrintf(t *testing.T) {
	origEnable := Enable
	origWriter := Logger.Writer()
	t.Cleanup(func() {
		Enable = origEnable
		Logger.SetOutput(origWriter)
	})

	Enable = true

	var buf bytes.Buffer
	Logger.SetOutput(&buf)

	Printf("hello %s", "world")

	if got := buf.String(); !strings.Contains(got, "hello world") {
		t.Fatalf("expected redirected output to contain log message, got %q", got)
	}
}

// TestLoggerSetOutputRedirectsPrefixLogger ensures loggers returned by
// NewPrefixLogger honor a redirected output set via Logger.SetOutput.
func TestLoggerSetOutputRedirectsPrefixLogger(t *testing.T) {
	origEnable := Enable
	origWriter := Logger.Writer()
	t.Cleanup(func() {
		Enable = origEnable
		Logger.SetOutput(origWriter)
	})

	Enable = true

	var buf bytes.Buffer
	Logger.SetOutput(&buf)

	dlog := NewPrefixLogger("test: ")
	dlog.Printf("prefixed message")

	if got := buf.String(); !strings.Contains(got, "prefixed message") {
		t.Fatalf("expected redirected output to contain prefixed message, got %q", got)
	}
}

// TestLoggerDefaultsToStderr documents that, without calling SetOutput, the
// default destination remains os.Stderr for backwards compatibility.
func TestLoggerDefaultsToStderr(t *testing.T) {
	if Logger.Writer() != os.Stderr {
		t.Fatalf("expected default logger writer to be os.Stderr")
	}
}

// TestNewPrefixLoggerWithNilLogger checks that NewPrefixLogger does not panic
// when Logger has been set to nil. Logger is exported, so an application can
// assign it, and before NewPrefixLogger derived its writer from Logger this
// case wrote to os.Stderr without touching Logger at all.
func TestNewPrefixLoggerWithNilLogger(t *testing.T) {
	origEnable := Enable
	origLogger := Logger
	t.Cleanup(func() {
		Enable = origEnable
		Logger = origLogger
	})

	Enable = true
	Logger = nil

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewPrefixLogger panicked with a nil Logger: %v", r)
		}
	}()

	dlog := NewPrefixLogger("test: ")
	if dlog.Writer() != os.Stderr {
		t.Fatalf("expected fallback to os.Stderr, got %T", dlog.Writer())
	}
}
