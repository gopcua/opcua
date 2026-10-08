package matrix_test

import (
	"flag"
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/matrix"
	"github.com/gopcua/opcua/tests/spec/internal/matrix/suites"
	"github.com/gopcua/opcua/tests/spec/internal/rules"
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
	matrix.RegisterSuites([]matrix.Suite{suites.P4_06_07(), suites.P4_05_14()}, matrix.KnownDefects(), fault.AllFaults...)
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

// TestPlanOverTheRealSuite asserts the plan over both suites and the
// full fault catalogue builds one case per scenario × fault with the
// predicted defect table attached, and prints how many apply and how
// many skip per scenario.
func TestPlanOverTheRealSuite(t *testing.T) {
	all := append([]matrix.Suite{suites.P4_06_07()}, suites.P4_05_14())
	cases, err := matrix.Plan(all, fault.AllFaults, matrix.KnownDefects())
	if err != nil {
		t.Fatalf("Plan over the suites returned an error: %v", err)
	}
	scenarioCount := 0
	for _, suite := range all {
		scenarioCount += len(suite.Scenarios())
	}
	if len(cases) != scenarioCount*len(fault.AllFaults) {
		t.Fatalf("Plan built %d cases, want %d (%d scenarios × %d faults)", len(cases), scenarioCount*len(fault.AllFaults), scenarioCount, len(fault.AllFaults))
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

	t.Run("the scenario values plan the same cases", func(t *testing.T) {
		var planned []matrix.Case
		for _, suite := range all {
			for _, scenario := range asScenarios(suite, matrix.KnownDefects()) {
				scenarioCases, err := scenario.Cases(fault.AllFaults)
				if err != nil {
					t.Fatalf("planning %s returned an error: %v", scenario.Name, err)
				}
				planned = append(planned, scenarioCases...)
			}
		}
		samePlans(t, all, cases, planned)
	})
}

// clauseOrdinals are the ordinals the clause files give the suites'
// scenarios.
var clauseOrdinals = map[string]int{
	"SessionSurvives": 1, "SessionLost": 2, "SubscriptionsLost": 3, "SteadyPublishing": 4, "CancelThenSubscribe": 5,
}

// asScenarios restates one suite's scenarios as scenario values: the
// rules of each fault's category, the subscription invariants, and the
// known defects that apply to the scenario under some fault.
func asScenarios(suite matrix.Suite, defects []matrix.SuiteDefect) []matrix.Scenario {
	var scenarios []matrix.Scenario
	for _, old := range suite.Scenarios() {
		name := old.Name()
		var own []matrix.KnownDefect
		for _, defect := range defects {
			if !slices.ContainsFunc(fault.AllFaults, func(f fault.Fault) bool { return defect.Applies(name, f) }) {
				continue
			}
			own = append(own, matrix.KnownDefect{Issue: defect.Issue, Check: defect.Check, Applies: func(f fault.Fault) bool { return defect.Applies(name, f) }})
		}
		scenarios = append(scenarios, matrix.Scenario{
			Clause:       suite.Clause(),
			Name:         name,
			Ordinal:      clauseOrdinals[name],
			Sends:        old.Sends(),
			Rules:        func(f fault.Fault) []rules.Rule { return suite.Rules(old.Category(f)) },
			Invariants:   matrix.SubscriptionInvariants,
			KnownDefects: own,
		})
	}
	return scenarios
}

// samePlans asserts the scenario values planned exactly the suites'
// cases: the same paths in the same order, the same skip reasons, the
// same checks in order with the same labels and phases, and the value
// block the suite derives.
func samePlans(t *testing.T, clauses []matrix.Suite, old, planned []matrix.Case) {
	t.Helper()
	if len(planned) != len(old) {
		t.Fatalf("the scenario values planned %d cases, the suites %d", len(planned), len(old))
	}
	scenarioNamed := map[string]matrix.SuiteScenario{}
	for _, suite := range clauses {
		for _, scenario := range suite.Scenarios() {
			scenarioNamed[suite.Clause()+"/"+scenario.Name()] = scenario
		}
	}
	for i := range old {
		was, now := old[i], planned[i]
		if !slices.Equal(was.Path, now.Path) {
			t.Fatalf("case %d is %v in the scenario values, %v in the suites", i, now.Path, was.Path)
		}
		if (was.Skip == nil) != (now.Skip == nil) || (was.Skip != nil && *was.Skip != *now.Skip) {
			t.Errorf("%v skips with %v in the scenario values, %v in the suites", now.Path, now.Skip, was.Skip)
		}
		if !reflect.DeepEqual(was.Checks, now.Checks) {
			t.Errorf("%v has the checks\n%+v\nin the scenario values, and\n%+v\nin the suites", now.Path, now.Checks, was.Checks)
		}
		f := namedFault(t, was.Path[2])
		if want := suites.CaseBlock(scenarioNamed[was.Path[0]+"/"+was.Path[1]], f); now.Block != want {
			t.Errorf("%v has the value block %d in the scenario values, %d in the suites", now.Path, now.Block, want)
		}
	}
}

func namedFault(t *testing.T, name string) fault.Fault {
	t.Helper()
	for _, f := range fault.AllFaults {
		if f.Name() == name {
			return f
		}
	}
	t.Fatalf("no fault named %s", name)
	return nil
}

// TestMain runs the package's tests only when the spec suite runs, so
// a plain `go test ./...` skips the package visibly instead of paying
// for its specs.
func TestMain(m *testing.M) {
	specrun.Main(m)
}
