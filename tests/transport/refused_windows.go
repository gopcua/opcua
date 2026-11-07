//go:build conformance && windows

// This file is not a row: it holds one build-tagged helper,
// refused below, that the address row's setup probe calls. The
// helper cannot live in harness.go, because a build constraint applies
// to a whole file, and harness.go must build on every platform the
// suite does. The pair is named on the race_on.go/race_off.go model in
// tests/server.

package transport

import (
	"errors"
	"syscall"
)

// errWSAEConnRefused is Winsock's WSAECONNREFUSED, errno 10061, the
// error a refused connect reports on Windows. Go's syscall package has
// no such constant on windows: its syscall.ECONNREFUSED there is an
// invented APPLICATION_ERROR value (zerrors_windows.go), and Errno.Is
// (syscall_windows.go) has no case that maps a refused connection to
// errors.Is's target, so errors.Is(err, syscall.ECONNREFUSED) is false
// for a real refusal. errors.Is(err, syscall.Errno(errWSAEConnRefused))
// is true for it, because the error the dial returns is that errno.
const errWSAEConnRefused = syscall.Errno(10061) // WSAECONNREFUSED

// refused reports whether a dial failed because the peer refused the
// connection.
func refused(err error) bool {
	return errors.Is(err, errWSAEConnRefused)
}
