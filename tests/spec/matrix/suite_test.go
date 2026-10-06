package matrix_test

import (
	"flag"
	"os"
	"testing"

	"github.com/gopcua/opcua/tests/spec/faults"
	"github.com/gopcua/opcua/tests/spec/matrix"
	"github.com/gopcua/opcua/tests/spec/matrix/suites"
	"github.com/gopcua/opcua/tests/spec/spectest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() {
	if os.Getenv("SPECTEST_MATRIX") != "" {
		matrix.RegisterSuites([]matrix.Suite{suites.P4_06_07()}, nil, faults.AllFaults...)
	}
}

// TestMatrix runs the registered failure matrix suites. It stays
// behind SPECTEST_MATRIX so a plain go test ./... stays fast and the
// skip is visible; the self-spec has its own bootstrap and always
// runs, except inside a matrix run, where TestMatrix covers it — a
// binary may run RunSpecs once. The label filter resolves exactly as
// the part4 suite does.
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

// TestPlanOverTheRealSuite asserts the plan over the §6.7 suite and
// the full fault catalogue builds one case per scenario × fault, and
// prints how many apply and how many skip per scenario.
func TestPlanOverTheRealSuite(t *testing.T) {
	cases, err := matrix.Plan([]matrix.Suite{suites.P4_06_07()}, faults.AllFaults, nil)
	if err != nil {
		t.Fatalf("Plan over the §6.7 suite returned an error: %v", err)
	}
	if len(cases) != 3*len(faults.AllFaults) {
		t.Fatalf("Plan built %d cases, want %d (3 scenarios × %d faults)", len(cases), 3*len(faults.AllFaults), len(faults.AllFaults))
	}
	applicable := map[string]int{}
	skipped := map[string]int{}
	for _, c := range cases {
		if c.Skip != nil {
			skipped[c.Path[1]]++
			continue
		}
		applicable[c.Path[1]]++
	}
	for _, scenario := range suites.P4_06_07().Scenarios() {
		t.Logf("%s: %d applicable, %d skipped", scenario.Name(), applicable[scenario.Name()], skipped[scenario.Name()])
	}
}
