package faults

import (
	"fmt"
	"os"
	"testing"

	"github.com/gopcua/opcua/tests/spec/internal/suitegate"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFaults(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "faults")
}

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
