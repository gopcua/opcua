package ualog

import (
	"fmt"
	"log/slog"
)

// logmask implements the slog.LogValuer interface that makes it possible
// to defer the string formatting of the bitmask until it is actually
// requested by the logger
type logmask struct {
	value uint32
}

func (m logmask) LogValue() slog.Value {
	return slog.StringValue(fmt.Sprintf("%08d", m.value))
}

var _ slog.LogValuer = &logmask{}
