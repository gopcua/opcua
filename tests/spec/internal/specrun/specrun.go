// Package specrun decides whether the spec suite runs: every test
// package under tests/spec skips unless the environment variable
// SPECTEST is set to 1, so a plain `go test ./...` stays fast and the
// skip stays visible in every package's `go test -v` and `-json`
// output.
package specrun

import (
	"flag"
	"fmt"
	"os"
	"strings"
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

// MatrixRuns reports whether the full matrix test runs in this binary:
// the suite gate is open and -run does not exclude TestMatrix. A test
// that cannot share the binary with the matrix's RunSpecs — a second
// RunSpecs fails — skips on it.
func MatrixRuns() bool {
	if !Enabled() {
		return false
	}
	run := flag.Lookup("test.run")
	if run == nil || run.Value.String() == "" {
		return true
	}
	return strings.Contains(run.Value.String(), "TestMatrix")
}

// Main runs a test package's tests behind the suite gate.
func Main(m *testing.M) {
	if !Enabled() {
		fmt.Println(SkipReason())
		os.Exit(0)
	}
	os.Exit(m.Run())
}
