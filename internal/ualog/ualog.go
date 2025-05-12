package ualog

import (
	"context"
	"log/slog"
	"os"
)

// SetDebugLogger configures the default log level to DEBUG
// if [debug] is true.
func SetDebugLogger(debug bool) {
	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)
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
