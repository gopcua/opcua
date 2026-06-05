package ualog

import (
	"context"
	"log/slog"
	"strings"
)

type config struct {
	logger *slog.Logger

	errorKey         string
	requestSanitizer RequestSanitizer
}

func newConfig() *config {
	return &config{
		logger:           slog.Default(),
		errorKey:         "err",
		requestSanitizer: DefaultRequestSanitizer,
	}
}

func newContextFromConfig(ctx context.Context, cfg *config) context.Context {
	return newContextWithStateFromConfig(ctx, cfg)
}

type Option func(*config)

// WithErrorKey replaces the default error message key with the supplied value
func WithErrorKey(key string) Option {
	return func(c *config) {
		if key = strings.TrimSpace(key); key != "" {
			c.errorKey = key
		}
	}
}

// WithHandler allows ualog to create a new logger directly from the supplied
// [log/slog.Handler]
func WithHandler(h slog.Handler) Option {
	return func(c *config) {
		if h != nil {
			c.logger = slog.New(h)
		}
	}
}

// WithLogger is an option that can be used when the caller wants to create a new
// ualog logger based on an already decorated [log/slog.Logger]
func WithLogger(l *slog.Logger) Option {
	return func(c *config) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithRequestSanitizer is an option that allows the caller to control what
// information is added to log records when incoming [ua.Request] instances
// are logged
func WithRequestSanitizer(sanitizer RequestSanitizer) Option {
	return func(c *config) {
		if sanitizer != nil {
			c.requestSanitizer = sanitizer
		}
	}
}
