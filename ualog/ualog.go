package ualog

import (
	"context"
	"log/slog"
	"slices"
)

// Debug forwards the provided message and attributes to the current logger
// iff its handler's debug level is enabled
func Debug(ctx context.Context, msg string, attrs ...Attr) {
	logger := loggerFromContext(ctx)
	if logger.Handler().Enabled(ctx, slog.LevelDebug) {
		log(ctx, loggerFromContext(ctx), slog.LevelDebug, msg, attrs...)
	}
}

// DebugFunc forwards the provided message and attributes, retrieved from the attrs callback,
// to the current logger iff its handler's debug level is enabled.
//
// This debug method allows the deferal of log attribute creation to only happen when
// they will actually be used. The result is a reduced performace penalty in non debug modes.
func DebugFunc(ctx context.Context, msg string, attrs func() []Attr) {
	logger := loggerFromContext(ctx)
	if logger.Handler().Enabled(ctx, slog.LevelDebug) {
		log(ctx, logger, slog.LevelDebug, msg, attrs()...)
	}
}

// Error forwards the provided message and attributes to the current logger
func Error(ctx context.Context, msg string, attrs ...Attr) {
	log(ctx, loggerFromContext(ctx), slog.LevelError, msg, attrs...)
}

// Info forwards the provided message and attributes to the current logger
func Info(ctx context.Context, msg string, attrs ...Attr) {
	log(ctx, loggerFromContext(ctx), slog.LevelInfo, msg, attrs...)
}

// Warn forwards the provided message and attributes to the current logger
func Warn(ctx context.Context, msg string, attrs ...Attr) {
	log(ctx, loggerFromContext(ctx), slog.LevelWarn, msg, attrs...)
}

// WithAttrs takes a context and adds the supplied ualog Attr values
// to a potentially already existing stored slice of log attributes
func WithAttrs(ctx context.Context, attrs ...Attr) context.Context {
	attrCount := len(attrs)
	if attrCount == 0 {
		return ctx
	}

	args := slices.Grow(attributesFromContext(ctx), attrCount)

	for idx := range attrs {
		args = append(args, attrs[idx])
	}

	return newContextWithLogAttributes(ctx, args)
}
