package matrix_test

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/gopcua/opcua/tests/spec/internal/faults"
	"github.com/gopcua/opcua/tests/spec/internal/matrix"
	"github.com/gopcua/opcua/tests/spec/internal/matrix/suites"
	"github.com/gopcua/opcua/tests/spec/internal/spectest"
	"github.com/gopcua/opcua/tests/spec/internal/suitegate"
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
	if !suitegate.Enabled() {
		t.Skip(suitegate.SkipReason())
		return
	}
	matrix.RegisterSuites([]matrix.Suite{suites.P4_06_07(), suites.P4_05_14()}, matrix.KnownDefects(), faults.AllFaults...)
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
