package matrix

import (
	"flag"
	"os"
	"testing"

	"github.com/gopcua/opcua/tests/spec/spectest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestMatrix runs the registered failure matrix suites. It stays
// behind SPECTEST_MATRIX so a plain go test ./... stays fast and the
// skip is visible; the self-spec has its own bootstrap and always
// runs. The label filter resolves exactly as the part4 suite does.
func TestMatrix(t *testing.T) {
	if os.Getenv("SPECTEST_MATRIX") == "" {
		t.Skip("set SPECTEST_MATRIX=1 to run the failure matrix")
		return
	}
	suiteConfig, reporterConfig := GinkgoConfiguration()
	var labelPassed bool
	flag.Visit(func(passed *flag.Flag) {
		if passed.Name == "ginkgo.label-filter" {
			labelPassed = true
		}
	})
	suiteConfig.LabelFilter = spectest.LabelFilter(labelPassed, suiteConfig.LabelFilter, os.Getenv("SPECTEST_LABEL_FILTER"))
	RegisterFailHandler(Fail)
	RunSpecs(t, "matrix", suiteConfig, reporterConfig)
}
