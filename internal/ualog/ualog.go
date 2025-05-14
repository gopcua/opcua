// Package ualog contains helper functions for logging.
package ualog

import (
	"log/slog"
	"os"
)

// NewTextHandler returns a handler with log level DEBUG
// if [debug] is true. Otherwise, it defaults to INFO.
func NewTextHandler(debug bool) *slog.TextHandler {
	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}
	return slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})
}

// Fatal is a convenience function for logging an error and exiting immediately.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
