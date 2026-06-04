package ualog

import (
	"context"
	"log/slog"
)

type keyType struct{}

var attributesKey = keyType{}
var loggerKey = keyType{}

func newContextWithLogAttributes(ctx context.Context, attrs []Attr) context.Context {
	return context.WithValue(ctx, attributesKey, attrs)
}

func newContextWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func attributesFromContext(ctx context.Context) []Attr {
	attrs, ok := ctx.Value(attributesKey).([]Attr)

	if !ok {
		return []Attr{}
	}

	return attrs
}

func loggerFromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(loggerKey).(*slog.Logger)

	if !ok {
		return slog.Default()
	}

	return logger
}
