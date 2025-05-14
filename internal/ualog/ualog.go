// Package ualog contains helper functions for logging.
package ualog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

type key int

const loggerKey key = 0

// NewContext returns a context with the logger as value.
func NewContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// FromContext returns the logger from the context.
func FromContext(ctx context.Context) *slog.Logger {
	return ctx.Value(loggerKey).(*slog.Logger)
}

// NewTextHandler returns a handler with log level DEBUG
// if [debug] is true. Otherwise, it defaults to INFO.
func NewTextHandler(debug bool, attrs ...any) slog.Handler {
	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})
	return AddAttrs(h, attrs...)
}

// NewJSONHandler returns a handler with log level DEBUG
// if [debug] is true. Otherwise, it defaults to INFO.
func NewJSONHandler(debug bool, attrs ...any) slog.Handler {
	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}
	h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})
	return AddAttrs(h, attrs...)
}

// Fatal is a convenience function for logging an error and exiting immediately.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

// TypeOf returns the type name of [v].
func TypeOf(v any) string {
	return fmt.Sprintf("%T", v)
}

// AddAttrs adds attributes to a handler from "pairs"
// like the slog.Logger.With() function.
func AddAttrs(h slog.Handler, attrs ...any) slog.Handler {
	return h.WithAttrs(argsToAttrSlice(attrs))
}

// copied from src/log/slog/{attr,record}.go

const badKey = "!BADKEY"

func argsToAttrSlice(args []any) []slog.Attr {
	var (
		attr  slog.Attr
		attrs []slog.Attr
	)
	for len(args) > 0 {
		attr, args = argsToAttr(args)
		attrs = append(attrs, attr)
	}
	return attrs
}

// argsToAttr turns a prefix of the nonempty args slice into an Attr
// and returns the unconsumed portion of the slice.
// If args[0] is an Attr, it returns it.
// If args[0] is a string, it treats the first two elements as
// a key-value pair.
// Otherwise, it treats args[0] as a value with a missing key.
func argsToAttr(args []any) (slog.Attr, []any) {
	switch x := args[0].(type) {
	case string:
		if len(args) == 1 {
			return slog.String(badKey, x), nil
		}
		return slog.Any(x, args[1]), args[2:]

	case slog.Attr:
		return x, args[1:]

	default:
		return slog.Any(badKey, x), args[1:]
	}
}
