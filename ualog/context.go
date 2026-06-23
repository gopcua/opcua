package ualog

import (
	"context"
	"log/slog"

	"github.com/gopcua/opcua/ua"
)

type attributesKeyType struct{}
type stateKeyType struct{}

var attributesKey attributesKeyType
var stateKey stateKeyType

type RequestSanitizer func(ua.Request) any

type state struct {
	logger           *slog.Logger
	errorKey         string
	requestSanitizer RequestSanitizer
}

func newContextWithLogAttributes(ctx context.Context, attrs []Attr) context.Context {
	return context.WithValue(ctx, attributesKey, attrs)
}

func newContextWithStateFromConfig(ctx context.Context, cfg *config) context.Context {
	return context.WithValue(ctx, stateKey, newStateFromConfig(cfg))
}

func newStateFromConfig(cfg *config) *state {
	return &state{
		logger:           cfg.logger,
		errorKey:         cfg.errorKey,
		requestSanitizer: cfg.requestSanitizer,
	}
}

func attributesFromContext(ctx context.Context) []Attr {
	attrs, ok := ctx.Value(attributesKey).([]Attr)

	if !ok {
		return []Attr{}
	}

	return attrs
}

func stateFromContext(ctx context.Context) *state {
	theState, ok := ctx.Value(stateKey).(*state)

	if !ok {
		return newStateFromConfig(newConfig())
	}

	return theState
}
