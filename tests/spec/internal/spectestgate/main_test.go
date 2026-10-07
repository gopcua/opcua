package spectestgate_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/gopcua/opcua/tests/spec/internal/suitegate"
)

// TestMain runs the package's tests only when the spec suite runs, so
// a plain `go test ./...` skips the package visibly instead of paying
// for its specs.
func TestMain(m *testing.M) {
	if !suitegate.Enabled() {
		fmt.Println(suitegate.SkipReason())
		os.Exit(0)
	}
	os.Exit(m.Run())
}
