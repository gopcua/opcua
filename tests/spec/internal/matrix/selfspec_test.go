package matrix

import (
	"flag"
	"os"
	"testing"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/specrun"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// faultNamed returns the catalogue's fault of that name.
func faultNamed(name string) fault.Fault {
	for _, f := range fault.AllFaults {
		if f.Name() == name {
			return f
		}
	}
	panic("no fault named " + name)
}

// TestSelfSpec runs the scenarios registered in this binary standalone,
// so the label filter resolves through specrun.LabelFilter and the
// default run excludes the labelled known-defect checks. This binary
// registers no scenario; the nested failure-continuation run
// (SPECTEST_FAILFIRST) registers its own.
func TestSelfSpec(t *testing.T) {
	suiteConfig, reporterConfig := ginkgo.GinkgoConfiguration()
	var labelPassed bool
	flag.Visit(func(passed *flag.Flag) {
		if passed.Name == "ginkgo.label-filter" {
			labelPassed = true
		}
	})
	suiteConfig.LabelFilter = specrun.LabelFilter(labelPassed, suiteConfig.LabelFilter, os.Getenv("SPECTEST_LABEL_FILTER"))
	RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "matrix self-spec", suiteConfig, reporterConfig)
}
