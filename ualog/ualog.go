package ualog

import (
	"context"
	"log/slog"
)

// Debug forwards the provided message and attributes to the current logger
// iff its handler's debug level is enabled
func Debug(ctx context.Context, msg string, attrs ...Attr) {
	state := stateFromContext(ctx)
	if state.logger.Handler().Enabled(ctx, slog.LevelDebug) {
		log(ctx, state.logger, slog.LevelDebug, msg, attrs...)
	}
}

// DebugFunc forwards the provided message and attributes, retrieved from the attrs callback,
// to the current logger iff its handler's debug level is enabled.
//
// This debug method allows the deferral of log attribute creation to only happen when
// they will actually be used. The result is a reduced performance penalty in non debug modes.
func DebugFunc(ctx context.Context, msg string, attrs func() []Attr) {
	state := stateFromContext(ctx)
	if state.logger.Handler().Enabled(ctx, slog.LevelDebug) {
		log(ctx, state.logger, slog.LevelDebug, msg, attrs()...)
	}
}

// Error forwards the provided message, error and attributes to the current logger
func Error(ctx context.Context, msg string, err error, attrs ...Attr) {
	state := stateFromContext(ctx)

	if err != nil {
		attrs = append([]Attr{String(state.errorKey, err.Error())}, attrs...)
	}

	log(ctx, state.logger, slog.LevelError, msg, attrs...)
}

// Info forwards the provided message and attributes to the current logger
func Info(ctx context.Context, msg string, attrs ...Attr) {
	state := stateFromContext(ctx)
	log(ctx, state.logger, slog.LevelInfo, msg, attrs...)
}

// Warn forwards the provided message and attributes to the current logger
func Warn(ctx context.Context, msg string, attrs ...Attr) {
	state := stateFromContext(ctx)
	log(ctx, state.logger, slog.LevelWarn, msg, attrs...)
}

// WithAttrs takes a context and adds the supplied ualog Attr values
// to a potentially already existing stored slice of log attributes
func WithAttrs(ctx context.Context, attrs ...Attr) context.Context {
	attrCount := len(attrs)
	if attrCount == 0 {
		return ctx
	}

	stored := attributesFromContext(ctx)

	next := make([]Attr, 0, len(stored)+len(attrs))
	next = append(next, stored...)
	next = append(next, attrs...)

	return newContextWithLogAttributes(ctx, next)
}
