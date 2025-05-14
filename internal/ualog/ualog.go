// Package ualog contains helper functions for logging.
package ualog

import (
	"context"
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

// map slog global functions into ualog so that we can add Fatal
var (
	Default = slog.Default
	Group   = slog.Group
	With    = slog.With

	Debug = slog.Debug
	Info  = slog.Info
	Warn  = slog.Warn
	Error = slog.Error

	DebugContext = slog.DebugContext
	InfoContext  = slog.InfoContext
	WarnContext  = slog.WarnContext
	ErrorContext = slog.ErrorContext
)

// Fatal is a convenience function for logging an error and exiting immediately.
func Fatal(msg string, args ...any) {
	Error(msg, args...)
	os.Exit(1)
}

// FatalContext is a convenience function for logging an error and exiting immediately.
func FatalContext(ctx context.Context, msg string, args ...any) {
	ErrorContext(ctx, msg, args...)
	os.Exit(1)
}
