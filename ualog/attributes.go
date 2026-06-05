package ualog

import (
	"context"
	"log/slog"
	"time"

	"github.com/gopcua/opcua/ua"
)

const (
	// NodeIdKey should be used when logging the id of a ua.Node
	//
	// This avoids the use of "node", "node_id", "id" in different places.
	NodeIdKey string = "node_id"
)

// An Attr is a key-value pair
type Attr slog.Attr

// Any returns an Attr for the supplied value
var Any = func(key string, value any) Attr {
	return Attr(slog.Any(key, value))
}

// Bitmask wraps a log valuer that formats a bitmask on demand
var Bitmask = func(key string, value uint32) Attr {
	return Attr(slog.Any(key, logmask{value: value}))
}

// Duration returns an Attr for a [time.Duration]
var Duration = func(key string, value time.Duration) Attr {
	return Attr(slog.Duration(key, value))
}

// GroupAttrs returns a single Attr for a group consisting of
// the given Attrs
var GroupAttrs = func(key string, args ...Attr) Attr {
	slogAttrs := make([]slog.Attr, 0, len(args))
	for argIdx := range len(args) {
		slogAttrs = append(slogAttrs, slog.Attr(args[argIdx]))
	}

	return Attr(slog.GroupAttrs(key, slogAttrs...))
}

// Int converts an int to an int64 and returns an Attr with that value
var Int = func(key string, value int) Attr {
	return Attr(slog.Int(key, value))
}

// Namespace takes a namespace id and returns a Uint32
// Attr with the key "namespace"
var Namespace = func(namespaceId uint16) Attr {
	return Uint32("namespace", uint32(namespaceId))
}

// Request sanitizes a request object to prevent/reduce likelyhood
// of exposing sensitive data, before turning it into an Any attribute
var Request = func(ctx context.Context, req ua.Request) Attr {
	return Any("request", stateFromContext(ctx).requestSanitizer(req))
}

// String returns an Attr for a string value
var String = func(key, value string) Attr {
	return Attr(slog.String(key, value))
}

// Uint64 returns an Attr for a uint64
var Uint64 = func(key string, value uint64) Attr {
	return Attr(slog.Uint64(key, value))
}

// Uint32 converts a uint32 to a uint64 and returns an Attr for that value
var Uint32 = func(key string, value uint32) Attr {
	return Uint64(key, uint64(value))
}
