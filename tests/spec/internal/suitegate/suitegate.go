// Package suitegate decides whether the spec suite runs: every test
// package under tests/spec skips unless the environment variable
// SPECTEST is set to 1, so a plain `go test ./...` stays fast and the
// skip stays visible in every package's output.
package suitegate

import (
	"flag"
	"os"
	"strings"
)

const enabled = "SPECTEST=1"

// Enabled reports whether the spec suite runs: SPECTEST=1.
func Enabled() bool {
	return os.Getenv("SPECTEST") == "1"
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
