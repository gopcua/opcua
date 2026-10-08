package part4

import (
	"reflect"
	"slices"
	"testing"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/matrix"
	"github.com/gopcua/opcua/tests/spec/internal/matrix/suites"
)

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
		scenarios := []matrix.Scenario{sessionSurvives, sessionLost, subscriptionsLost, steadyPublishing, cancelThenSubscribe}
		for _, scenario := range scenarios {
			scenarioCases, err := scenario.Cases(fault.AllFaults)
			if err != nil {
				t.Fatalf("planning %s returned an error: %v", scenario.Name, err)
			}
			planned = append(planned, scenarioCases...)
		}
		samePlans(t, all, cases, planned)
	})
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
