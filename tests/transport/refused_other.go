//go:build conformance && !windows

// This file is not a row: it holds one build-tagged helper, refused
// below, that the address row's setup probe calls. The helper cannot
// live in harness.go, because a build constraint applies to a whole
// file, and harness.go must build on every platform the suite does.
// The pair is named on the race_on.go/race_off.go model in
// tests/server.

package transport

import (
	"errors"
	"syscall"
)

// refused reports whether a dial failed because the peer refused the
// connection. On the non-windows platforms a refused connect reports
// syscall.ECONNREFUSED.
func refused(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED)
}
