package specrun

import "testing"

// TestMain runs the package's tests only when the spec suite runs, so
// a plain `go test ./...` skips the package visibly instead of paying
// for its tests.
func TestMain(m *testing.M) {
	Main(m)
}
