package part4

import (
	"testing"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/matrix"
)

// TestPlanOverTheRealSuite asserts the plan over both clauses'
// scenarios and the full fault catalogue builds one case per scenario
// × fault with the predicted defects attached, and prints how many
// apply and how many skip per scenario.
func TestPlanOverTheRealSuite(t *testing.T) {
	scenarios := []matrix.Scenario{sessionSurvives, sessionLost, subscriptionsLost, steadyPublishing, cancelThenSubscribe}
	applicable := map[string]int{}
	skipped := map[string]int{}
	for _, scenario := range scenarios {
		cases, err := scenario.Cases(fault.AllFaults)
		if err != nil {
			t.Fatalf("Plan over the suites returned an error: %v", err)
		}
		if len(cases) != len(fault.AllFaults) {
			t.Fatalf("Plan built %d cases for %s, want one per fault (%d)", len(cases), scenario.Name, len(fault.AllFaults))
		}
		for _, c := range cases {
			if c.Skip != nil {
				skipped[c.Path[1]]++
				continue
			}
			applicable[c.Path[1]]++
		}
	}
	for _, scenario := range scenarios {
		t.Logf("%s: %d applicable, %d skipped", scenario.Name, applicable[scenario.Name], skipped[scenario.Name])
	}
}

// TestRuleNamesDifferFromInvariantNames pins that no rule a scenario
// returns under any fault shares a name with an invariant of that
// scenario or with HaveFired. The plan and the known-defect table find
// a check by its name, so a shared name would make them confuse the
// rule with the invariant.
func TestRuleNamesDifferFromInvariantNames(t *testing.T) {
	scenarios := []matrix.Scenario{sessionSurvives, sessionLost, subscriptionsLost, steadyPublishing, cancelThenSubscribe}
	for _, scenario := range scenarios {
		invariantNames := map[string]bool{"HaveFired": true}
		for _, invariant := range scenario.Invariants {
			invariantNames[invariant.Name] = true
		}
		if scenario.Rules == nil {
			continue
		}
		for _, f := range fault.AllFaults {
			for _, rule := range scenario.Rules(f) {
				if invariantNames[rule.Name] {
					t.Errorf("the rule %s of %s under %s shares an invariant's name, so the runner would assert the invariant instead", rule.Name, scenario.Name, f.Name())
				}
			}
		}
	}
}

// TestCaseValuesNeverOverlap pins that no two cases of the matrix share
// a value: every case derives its values — first value, answered
// values, retained value, sentinel and consumer burst alike — from its
// own block, so a received value matches the notification that carried
// it by value and never a value another case answered.
func TestCaseValuesNeverOverlap(t *testing.T) {
	reestablishingBlock := func(block int32) []int32 {
		values := reestablishingValuesOf(block)
		all := []int32{values.first, values.v1, values.v2, values.v3, values.vArm, values.sentinel}
		return append(all, values.burst[:]...)
	}
	publishingBlock := func(block int32) []int32 {
		values := publishingValuesOf(block)
		all := []int32{values.first, values.vArm, values.sentinel}
		all = append(all, values.steady[:]...)
		return append(all, values.cycles[:]...)
	}
	every := []struct {
		scenario matrix.Scenario
		values   func(block int32) []int32
	}{
		{sessionSurvives, reestablishingBlock},
		{sessionLost, reestablishingBlock},
		{subscriptionsLost, reestablishingBlock},
		{steadyPublishing, publishingBlock},
		{cancelThenSubscribe, publishingBlock},
	}
	seen := map[int32]string{}
	for _, entry := range every {
		cases, err := entry.scenario.Cases(fault.AllFaults)
		if err != nil {
			t.Fatalf("planning %s returned an error: %v", entry.scenario.Name, err)
		}
		for _, c := range cases {
			owner := c.Path[1] + "/" + c.Path[2]
			for _, value := range entry.values(c.Block) {
				if previous, taken := seen[value]; taken {
					t.Errorf("value %d belongs to %s and %s", value, previous, owner)
				}
				seen[value] = owner
			}
		}
	}
	if len(seen) < 100 {
		t.Fatalf("collected %d distinct values over every case, want at least one per case and value", len(seen))
	}
}
