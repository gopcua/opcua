// Package specrun decides whether the spec suite runs: every test
// package under tests/spec skips unless the environment variable
// SPECTEST is set to 1, so a plain `go test ./...` stays fast and the
// skip stays visible in every package's `go test -v` and `-json`
// output.
package specrun

import (
	"fmt"
	"os"
	"testing"
)

// envSpecTest and enableValue spell the gate's variable and value
// once; both the check and the skip message read them.
const (
	envSpecTest = "SPECTEST"
	enableValue = "1"
)

var enabled = envSpecTest + "=" + enableValue

// Enabled reports whether the spec suite runs: SPECTEST=1.
func Enabled() bool {
	return os.Getenv(envSpecTest) == enableValue
}

// SkipReason is the message a test that skips reports.
func SkipReason() string {
	return "set " + enabled + " to run the spec suite"
}

// Main runs a test package's tests behind the suite gate.
func Main(m *testing.M) {
	if !Enabled() {
		fmt.Println(SkipReason())
		os.Exit(0)
	}
	os.Exit(m.Run())
}
