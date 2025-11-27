package ualog

import (
	"context"
	"log/slog"
)

type config struct {
	logger *slog.Logger
}

func newConfig() *config {
	return &config{
		logger: slog.Default(),
	}
}

func newContextFromConfig(ctx context.Context, cfg *config) context.Context {
	return newContext(ctx, cfg.logger)
}

type option func(*config)

func WithHandler(h slog.Handler) option {
	return func(c *config) {
		c.logger = slog.New(h)
	}
}

func WithLogger(l *slog.Logger) option {
	return func(c *config) {
		c.logger = l
	}
}
