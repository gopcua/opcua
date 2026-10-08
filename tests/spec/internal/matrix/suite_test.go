package matrix_test

import (
	"flag"
	"os"
	"testing"

	"github.com/gopcua/opcua/tests/spec/internal/specrun"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestMatrix runs the registered failure matrix suites. It stays
// behind the suite gate so a plain go test ./... stays fast and the
// skip is visible; the self-spec has its own bootstrap and always runs,
// except inside a matrix run, where TestMatrix covers it — a binary may
// run RunSpecs once. The suites register here, inside TestMatrix and
// nowhere in init: ginkgo collects top-level containers until RunSpecs,
// and the matrix's 2443-spec tree must not burden every other RunSpecs
// of this binary — TestSelfSpec runs its own tree, and
// TestContinueOnFailure's nested run inherits this gate. The label
// filter resolves exactly as the part4 suite does.
func TestMatrix(t *testing.T) {
	if !specrun.Enabled() {
		t.Skip(specrun.SkipReason())
		return
	}
	suiteConfig, reporterConfig := GinkgoConfiguration()
	var labelPassed bool
	flag.Visit(func(passed *flag.Flag) {
		if passed.Name == "ginkgo.label-filter" {
			labelPassed = true
		}
	})
	suiteConfig.LabelFilter = specrun.LabelFilter(labelPassed, suiteConfig.LabelFilter, os.Getenv("SPECTEST_LABEL_FILTER"))
	RegisterFailHandler(Fail)
	RunSpecs(t, "matrix", suiteConfig, reporterConfig)
}

// TestMain runs the package's tests only when the spec suite runs, so
// a plain `go test ./...` skips the package visibly instead of paying
// for its specs.
func TestMain(m *testing.M) {
	specrun.Main(m)
}
