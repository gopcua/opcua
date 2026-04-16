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

// WithErrorKey replaces the default error message key with the supplied value
func WithErrorKey(key string) option {
	return func(_ *config) {
		ErrorKey = key
	}
}

// WithHandler allows ualog to create a new logger directly from the supplied
// [log/slog.Handler]
func WithHandler(h slog.Handler) option {
	return func(c *config) {
		c.logger = slog.New(h)
	}
}

// WithLogger is an option that can be used when the caller wants to create a new
// ualog logger based on an already decorated [log/slog.Logger]
func WithLogger(l *slog.Logger) option {
	return func(c *config) {
		c.logger = l
	}
}
