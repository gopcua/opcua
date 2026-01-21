package ualog

import (
	"context"
	"log/slog"
)

func New(ctx context.Context, opts ...option) context.Context {
	cfg := newConfig()

	for _, opt := range opts {
		opt(cfg)
	}

	return newContextFromConfig(ctx, cfg)
}

func log(ctx context.Context, logger *slog.Logger, level slog.Level, msg string, args ...Attr) {
	switch len(args) {
	case 0:
		logger.LogAttrs(ctx, level, msg)
	case 1:
		logger.LogAttrs(ctx, level, msg, slog.Attr(args[0]))
	case 2:
		logger.LogAttrs(ctx, level, msg, slog.Attr(args[0]), slog.Attr(args[1]))
	case 3:
		logger.LogAttrs(ctx, level, msg, slog.Attr(args[0]), slog.Attr(args[1]), slog.Attr(args[2]))
	case 4:
		logger.LogAttrs(ctx, level, msg, slog.Attr(args[0]), slog.Attr(args[1]), slog.Attr(args[2]), slog.Attr(args[3]))
	default:
		slogAttrs := make([]slog.Attr, 0, len(args))
		for argIdx := range len(args) {
			attr := slog.Attr(args[argIdx])
			slogAttrs = append(slogAttrs, attr)
		}
		logger.LogAttrs(ctx, level, msg, slogAttrs...)
	}
}
