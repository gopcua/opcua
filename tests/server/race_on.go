//go:build race

package server

// raceDetector reports whether this binary was built with -race. This
// build-tagged const pair mirrors Go's own internal/race package
// (race.go and norace.go), the standard idiom for a constant the build
// flags decide.
const raceDetector = true
