package ualog

import (
	"context"
	"log/slog"
)

// Debug forwards the provided message and attributes to the current logger
// iff its handler's debug level is enabled
func Debug(ctx context.Context, msg string, attrs ...Attr) {
	logger := fromContext(ctx)
	if logger.Handler().Enabled(ctx, slog.LevelDebug) {
		log(ctx, fromContext(ctx), slog.LevelDebug, msg, attrs...)
	}
}

// DebugFunc forwards the provided message and attributes, retrieved from the attrs callback,
// to the current logger iff its handler's debug level is enabled.
//
// This debug method allows the deferal of log attribute creation to only happen when
// they will actually be used. The result is a reduced performace penalty in non debug modes.
func DebugFunc(ctx context.Context, msg string, attrs func() []Attr) {
	logger := fromContext(ctx)
	if logger.Handler().Enabled(ctx, slog.LevelDebug) {
		log(ctx, logger, slog.LevelDebug, msg, attrs()...)
	}
}

// Error forwards the provided message and attributes to the current logger
func Error(ctx context.Context, msg string, attrs ...Attr) {
	log(ctx, fromContext(ctx), slog.LevelError, msg, attrs...)
}

// Info forwards the provided message and attributes to the current logger
func Info(ctx context.Context, msg string, attrs ...Attr) {
	log(ctx, fromContext(ctx), slog.LevelInfo, msg, attrs...)
}

// Warn forwards the provided message and attributes to the current logger
func Warn(ctx context.Context, msg string, attrs ...Attr) {
	log(ctx, fromContext(ctx), slog.LevelWarn, msg, attrs...)
}

// WithAttrs takes a context and decorates its current logger using the supplied
// ualog Attr values
func WithAttrs(ctx context.Context, attrs ...Attr) context.Context {

	if attrCount := len(attrs); attrCount > 0 {
		logger := fromContext(ctx)
		for attrIdx := range attrCount {
			logger = logger.With(slog.Attr(attrs[attrIdx]))
		}
		ctx = newContext(ctx, logger)
	}

	return ctx
}
