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
		matrix.RegisterSuites([]matrix.Suite{suites.P4_06_07(), suites.P4_05_14()}, matrix.KnownDefects(), faults.AllFaults...)
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

// TestPlanOverTheRealSuite asserts the plan over both suites and the
// full fault catalogue builds one case per scenario × fault with the
// predicted defect table attached, and prints how many apply and how
// many skip per scenario.
func TestPlanOverTheRealSuite(t *testing.T) {
	all := append([]matrix.Suite{suites.P4_06_07()}, suites.P4_05_14())
	cases, err := matrix.Plan(all, faults.AllFaults, matrix.KnownDefects())
	if err != nil {
		t.Fatalf("Plan over the suites returned an error: %v", err)
	}
	scenarioCount := 0
	for _, suite := range all {
		scenarioCount += len(suite.Scenarios())
	}
	if len(cases) != scenarioCount*len(faults.AllFaults) {
		t.Fatalf("Plan built %d cases, want %d (%d scenarios × %d faults)", len(cases), scenarioCount*len(faults.AllFaults), scenarioCount, len(faults.AllFaults))
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
	for _, suite := range all {
		for _, scenario := range suite.Scenarios() {
			t.Logf("%s: %d applicable, %d skipped", scenario.Name(), applicable[scenario.Name()], skipped[scenario.Name()])
		}
	}
}
