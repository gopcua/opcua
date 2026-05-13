package ualog

import (
	"context"
	"log/slog"
)

type keyType struct{}

var loggerKey = keyType{}

func newContext(ctx context.Context, logger *slog.Logger, args ...any) context.Context {
	return context.WithValue(ctx, loggerKey, logger.With(args...))
}

func fromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(loggerKey).(*slog.Logger)

	if !ok {
		return slog.Default()
	}

	return logger
}
