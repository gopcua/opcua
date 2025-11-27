package ualog

import (
	"log/slog"
	"time"
)

var (
	ErrorKey  string = "err"
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

// Int converts an int to an int64 and returns an Attr with that value
var Int = func(key string, value int) Attr {
	return Attr(slog.Int(key, value))
}

// Uint64 returns an Attr for a uint64
var Uint64 = func(key string, value uint64) Attr {
	return Attr(slog.Uint64(key, value))
}

// Uint32 converts a uint32 to a uint64 and returns an Attr for that value
var Uint32 = func(key string, value uint32) Attr {
	return Uint64(key, uint64(value))
}

// String returns an Attr for a string value
var String = func(key, value string) Attr {
	return Attr(slog.String(key, value))
}

// Namespace takes a namespace id and returns a Uint32
// Attr with the key "namespace"
var Namespace = func(namespaceId uint16) Attr {
	return Uint32("namespace", uint32(namespaceId))
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

// Err takes an error and returns a String attr with the key
// ualog.ErrorKey and the value given by err.Error(). If err
// is nil, the Attr value will be the empty string.
var Err = func(err error) Attr {
	var errorMessage string
	if err != nil {
		errorMessage = err.Error()
	}
	return String(ErrorKey, errorMessage)
}
