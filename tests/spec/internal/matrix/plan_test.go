package matrix

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/message"
	"github.com/onsi/gomega"
)

// standInShall and standInShould stand in for a clause's rules: each
// carries a name and a Check, because a rule without either is
// rejected.
var (
	standInShall  = Rule{Name: "StandInShall", Check: func(Context) {}}
	standInShould = Rule{Name: "StandInShould", Check: func(Context) {}}
)

// standInBefore and standInAfter stand in for a scenario's invariants:
// the plan reads only their name and phase, and each fails on the
// broken snapshot and passes on the healthy one, so the fixture in
// TestPlanOrdersInvariantChecksBeforeRules can show which snapshot a
// check reads.
var (
	standInBefore = Invariant{Name: "StandInBefore", Phase: BeforeClose, Assert: func(observed Observed) {
		gomega.Expect(observed.Servers[0].KnownSessions).To(gomega.Equal(1), "the stand-in server holds the wrong session count")
	}}
	standInAfter = Invariant{Name: "StandInAfter", Phase: AfterClose, Assert: func(observed Observed) {
		gomega.Expect(observed.Received).To(gomega.HaveLen(1), "the stand-in client received the wrong values")
	}}
)

// scenarioOf builds a scenario for the unit tests: clause P4-1, the
// stand-in invariants, and the given rules under every fault.
func scenarioOf(name string, ordinal int, sends []message.Message, ruleSet ...Rule) Scenario {
	return Scenario{
		Clause:     "P4-1",
		Name:       name,
		Ordinal:    ordinal,
		Sends:      sends,
		Rules:      func(fault.Fault) []Rule { return ruleSet },
		Invariants: []Invariant{standInBefore, standInAfter},
	}
}

// casesOf plans every scenario over the faults, in order.
func casesOf(t *testing.T, faults []fault.Fault, scenarios ...Scenario) []Case {
	t.Helper()
	var cases []Case
	for _, s := range scenarios {
		planned, err := s.Cases(faults)
		if err != nil {
			t.Fatalf("Plan returned an error: %v", err)
		}
		cases = append(cases, planned...)
	}
	return cases
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

func TestPlanBuildsOneCasePerCombination(t *testing.T) {
	publishFault := namedFault(t, "RequestLost/Publish")
	readFault := namedFault(t, "RequestLost/Read")
	stallFault := namedFault(t, "Link/Stall")
	all := []fault.Fault{publishFault, readFault, stallFault}

	cases := casesOf(t, all,
		scenarioOf("A", 1, []message.Message{message.Publish}),
		scenarioOf("B", 2, []message.Message{message.Read}))
	if len(cases) != 6 {
		t.Fatalf("Plan built %d cases, want 6 (1 suite × 2 scenarios × 3 faults)", len(cases))
	}
	// scenario A sends Publish only and scenario B sends Read only, so
	// each message fault is skipped in the other's scenario and the
	// link fault applies in both.
	skipped := map[string]bool{}
	for _, c := range cases {
		if c.Skip != nil {
			skipped[strings.Join(c.Path, "/")] = true
			if c.Skip.Text == "" {
				t.Errorf("the skip reason of %v is empty", c.Path)
			}
			if len(c.Checks) != 0 {
				t.Errorf("the skipped case %v carries %d checks, want none", c.Path, len(c.Checks))
			}
		}
	}
	if len(skipped) != 2 || !skipped["P4-1/B/RequestLost/Publish"] || !skipped["P4-1/A/RequestLost/Read"] {
		t.Fatalf("the skipped cases are %v, want exactly the Publish fault in B and the Read fault in A", skipped)
	}
}

func TestPlanOrdersInvariantChecksBeforeRules(t *testing.T) {
	stall := namedFault(t, "Link/Stall")
	cases := casesOf(t, []fault.Fault{stall},
		scenarioOf("A", 1, []message.Message{message.Publish}, standInShall, standInShould))
	if len(cases) != 1 {
		t.Fatalf("Plan built %d cases, want 1", len(cases))
	}
	got := make([]string, 0, len(cases[0].Checks))
	for _, check := range cases[0].Checks {
		got = append(got, check.Name)
	}
	want := []string{
		"StandInBefore", "StandInShall", "StandInShould",
		"HaveFired", "StandInAfter",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("the checks are %v, want %v", got, want)
	}
	for _, check := range cases[0].Checks {
		switch check.Name {
		case "HaveFired", "StandInAfter":
			if check.Phase != AfterClose {
				t.Errorf("%s has phase %d, want AfterClose", check.Name, check.Phase)
			}
		default:
			if check.Phase != BeforeClose {
				t.Errorf("%s has phase %d, want BeforeClose", check.Name, check.Phase)
			}
		}
	}

	t.Run("each check reads the snapshot its phase names", func(t *testing.T) {
		// The runners assert through the package-level Gomega, which
		// InterceptGomegaFailures needs registered.
		gomega.RegisterTestingT(t)
		want := map[string]Phase{
			"StandInBefore": BeforeClose,
			"HaveFired":     AfterClose,
			"StandInAfter":  AfterClose,
		}
		healthy, broken := healthySnapshot(), brokenSnapshot()
		// fails reports whether the assertion fails, so a snapshot that
		// fails every check shows which snapshot a check reads.
		fails := func(assert func()) bool { return len(gomega.InterceptGomegaFailures(assert)) > 0 }
		readsOf := func(assert func(before, after Observed)) []Phase {
			var reads []Phase
			if fails(func() { assert(broken, healthy) }) {
				reads = append(reads, BeforeClose)
			}
			if fails(func() { assert(healthy, broken) }) {
				reads = append(reads, AfterClose)
			}
			return reads
		}
		checks := []Invariant{standInBefore, standInAfter, haveFiredInvariant}
		if len(checks) != len(want) {
			t.Fatalf("the stand-in checks are %d, want exactly the %d pinned here", len(checks), len(want))
		}
		for _, invariant := range checks {
			if _, found := want[invariant.Name]; !found {
				t.Fatalf("the stand-in checks hold no %s", invariant.Name)
			}
			assert := func(before, after Observed) {
				(&Observation{before: before, after: after}).assertInvariant(invariant)
			}
			if fails(func() { assert(healthy, healthy) }) {
				t.Fatalf("%s fails on the healthy snapshot, so the fixture cannot show which snapshot it reads", invariant.Name)
			}
			current := readsOf(assert)
			if !slices.Equal(current, []Phase{want[invariant.Name]}) {
				t.Errorf("the runner's %s reads the snapshots %v, want only %v", invariant.Name, current, want[invariant.Name])
			}
		}
		observed := &Observation{outcome: Outcome{Rules: Context{Value: 7}}}
		if read := observed.Context(); read.Value != 7 {
			t.Errorf("the runner handed a rule the context %+v, want the workload's", read)
		}
	})
}

// healthySnapshot passes every stand-in check and HaveFired: one
// session and one subscription on the connected server, the one
// produced value received once, the sentinel received, and the fault
// fired.
func healthySnapshot() Observed {
	return Observed{
		Produced:            []Produced{{Value: 1, SubscriptionID: 1, SequenceNumber: 1, Reachable: true, SubscriptionInstance: 1}},
		Received:            []int32{1},
		Servers:             []ServerState{{Reachable: true, Connected: true, KnownSessions: 1, ClosingAttempted: 1, LiveSubscriptions: 1}},
		ClientSubscriptions: 1,
		FaultEnd:            time.Unix(1, 0),
		Sentinel:            &Sentinel{Value: 1, AnsweredAt: time.Unix(1, 0), ReceivedAt: time.Unix(2, 0)},
		Fired:               true,
	}
}

// brokenSnapshot fails every stand-in check and HaveFired: two open
// sessions and no live subscription on the connected server, a value
// received twice and out of its produced order, no sentinel, and the
// fault never fired.
func brokenSnapshot() Observed {
	return Observed{
		Produced: []Produced{
			{Value: 1, SubscriptionID: 1, SequenceNumber: 2, Reachable: true, SubscriptionInstance: 1},
			{Value: 2, SubscriptionID: 1, SequenceNumber: 1, Reachable: true, SubscriptionInstance: 1},
		},
		Received:            []int32{1, 2, 2},
		Servers:             []ServerState{{Reachable: true, Connected: true, KnownSessions: 2}},
		ClientSubscriptions: 1,
	}
}

func TestPlanRejectsDuplicateScenarioNames(t *testing.T) {
	// The fixture must not leak into the registrations of later tests
	// in this binary.
	t.Cleanup(func() {
		registeredMu.Lock()
		defer registeredMu.Unlock()
		for _, ordinal := range []int{901, 902, 903} {
			delete(ordinalOwner, ordinal)
		}
		delete(ordinalOf, "P4-1/A")
		delete(ordinalOf, "P4-1/B")
	})
	if err := register(scenarioOf("A", 901, nil)); err != nil {
		t.Fatalf("registering the first scenario returned %v", err)
	}
	err := register(scenarioOf("A", 902, nil))
	if err == nil || !strings.Contains(err.Error(), "A") {
		t.Fatalf("duplicate scenario names returned %v, want an error naming the name", err)
	}
	err = register(scenarioOf("B", 901, nil))
	if err == nil || !strings.Contains(err.Error(), "ordinal 901") {
		t.Fatalf("a duplicate ordinal returned %v, want an error naming it", err)
	}
	if err := register(scenarioOf("A", 901, nil)); err != nil {
		t.Fatalf("registering the same scenario twice returned %v, want none", err)
	}
}

// TestPlanRejectsBadRules asserts the plan rejects the rule shapes a
// clause file must never declare. Each subtest prescribes one shape
// with a scenario that is otherwise well-formed, so one failing
// subtest names the rejection its shape misses.
func TestPlanRejectsBadRules(t *testing.T) {
	stall := namedFault(t, "Link/Stall")
	sends := []message.Message{message.Publish}
	t.Run("rejects a rule with an empty Name", func(t *testing.T) {
		scenario := scenarioOf("Bad", 901, sends, Rule{Name: "", Check: func(Context) {}})
		_, err := scenario.Cases([]fault.Fault{stall})
		if err == nil || !strings.Contains(err.Error(), "empty Name") {
			t.Fatalf("a rule with an empty Name returned %v, want an error naming the empty Name", err)
		}
	})
	t.Run("rejects a rule with no Check", func(t *testing.T) {
		scenario := scenarioOf("Bad", 902, sends, Rule{Name: "StandInUnchecked"})
		_, err := scenario.Cases([]fault.Fault{stall})
		if err == nil || !strings.Contains(err.Error(), "no Check") {
			t.Fatalf("a rule with no Check returned %v, want an error naming the missing Check", err)
		}
	})
	t.Run("rejects a rule that shares a scenario invariant's name", func(t *testing.T) {
		scenario := scenarioOf("Bad", 903, sends,
			Rule{Name: standInBefore.Name, Check: func(Context) {}})
		_, err := scenario.Cases([]fault.Fault{stall})
		if err == nil || !strings.Contains(err.Error(), "invariant's name") {
			t.Fatalf("a rule named as an invariant returned %v, want an error naming the shared name", err)
		}
	})
	t.Run("rejects a rule named HaveFired", func(t *testing.T) {
		scenario := scenarioOf("Bad", 904, sends,
			Rule{Name: haveFiredInvariant.Name, Check: func(Context) {}})
		_, err := scenario.Cases([]fault.Fault{stall})
		if err == nil || !strings.Contains(err.Error(), "invariant's name") {
			t.Fatalf("a rule named HaveFired returned %v, want an error naming the shared name", err)
		}
	})
	t.Run("rejects two rules that share a name", func(t *testing.T) {
		scenario := scenarioOf("Bad", 905, sends,
			Rule{Name: "StandInSame", Check: func(Context) {}},
			Rule{Name: "StandInSame", Check: func(c Context) {}})
		_, err := scenario.Cases([]fault.Fault{stall})
		if err == nil || !strings.Contains(err.Error(), "rules StandInSame of scenario P4-1/Bad under Link/Stall share one name and differ in Check") {
			t.Fatalf("two rules sharing a name returned %v, want an error naming the scenario and the fault", err)
		}
	})
	t.Run("rejects an ordinal left at zero", func(t *testing.T) {
		scenario := scenarioOf("Bad", 0, sends)
		_, err := scenario.Cases([]fault.Fault{stall})
		if err == nil || !strings.Contains(err.Error(), "ordinals count from 1") {
			t.Fatalf("a zero ordinal returned %v, want an error naming the ordinal", err)
		}
	})
}
