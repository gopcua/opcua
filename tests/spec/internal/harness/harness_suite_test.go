package harness

import (
	"testing"

	"github.com/gopcua/opcua/tests/spec/internal/specrun"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHarness(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "harness")
}

// TestMain runs the package's tests only when the spec suite runs, so
// a plain `go test ./...` skips the package visibly instead of paying
// for its specs.
func TestMain(m *testing.M) {
	specrun.Main(m)
}
