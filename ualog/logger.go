package ualog

import (
	"context"
	"log/slog"
	"slices"
)

func New(ctx context.Context, opts ...option) context.Context {
	cfg := newConfig()

	for _, opt := range opts {
		opt(cfg)
	}

	return newContextFromConfig(ctx, cfg)
}

func log(ctx context.Context, logger *slog.Logger, level slog.Level, msg string, args ...Attr) {

	storedAttributes := attributesFromContext(ctx)
	if len(storedAttributes) > 0 {
		args = slices.Concat(storedAttributes, args)
	}

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
	case 5:
		logger.LogAttrs(ctx, level, msg, slog.Attr(args[0]), slog.Attr(args[1]), slog.Attr(args[2]), slog.Attr(args[3]), slog.Attr(args[4]))
	case 6:
		logger.LogAttrs(ctx, level, msg, slog.Attr(args[0]), slog.Attr(args[1]), slog.Attr(args[2]), slog.Attr(args[3]), slog.Attr(args[4]), slog.Attr(args[5]))
	default:
		slogAttrs := make([]slog.Attr, 0, len(args))
		for argIdx := range len(args) {
			slogAttrs = append(slogAttrs, slog.Attr(args[argIdx]))
		}
		logger.LogAttrs(ctx, level, msg, slogAttrs...)
	}
}
