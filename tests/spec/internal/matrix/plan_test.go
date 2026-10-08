package matrix

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/invariants"
	"github.com/gopcua/opcua/tests/spec/internal/message"
	"github.com/onsi/gomega"
)

// standInShall and standInShould stand in for a clause's rules: the
// plan reads only their name, clause and keyword.
var (
	standInShall  = Rule{Name: "StandInShall", Clause: "P4-6.7", Keyword: "shall"}
	standInShould = Rule{Name: "StandInShould", Clause: "P4-6.7", Keyword: "should"}
)

// scenarioOf builds a scenario for the unit tests: clause P4-1, the
// subscription invariants, and the given rules under every fault.
func scenarioOf(name string, ordinal int, sends []message.Message, ruleSet ...Rule) Scenario {
	return Scenario{
		Clause:     "P4-1",
		Name:       name,
		Ordinal:    ordinal,
		Sends:      sends,
		Rules:      func(fault.Fault) []Rule { return ruleSet },
		Invariants: SubscriptionInvariants,
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

func checkNamed(t *testing.T, cases []Case, path []string, name string) Check {
	t.Helper()
	for _, c := range cases {
		if strings.Join(c.Path, "/") != strings.Join(path, "/") {
			continue
		}
		for _, check := range c.Checks {
			if check.Name == name {
				return check
			}
		}
	}
	t.Fatalf("no check %s in case %v", name, path)
	return Check{}
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
		"ResumePublishing", "KeepOneSessionOpen", "KeepOneSubscriptionPerClientSubscription",
		"StandInShall", "StandInShould",
		"HaveFired", "DeliverEachValueOnce", "DeliverInOrder", "CloseEveryKnownSession",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("the checks are %v, want %v", got, want)
	}
	for _, check := range cases[0].Checks {
		switch check.Name {
		case "HaveFired", "DeliverEachValueOnce", "DeliverInOrder", "CloseEveryKnownSession":
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
			"ResumePublishing": BeforeClose, "KeepOneSessionOpen": BeforeClose, "KeepOneSubscriptionPerClientSubscription": BeforeClose,
			"HaveFired": AfterClose, "DeliverEachValueOnce": AfterClose, "DeliverInOrder": AfterClose, "CloseEveryKnownSession": AfterClose,
		}
		healthy, broken := healthySnapshot(), brokenSnapshot()
		// fails reports whether the assertion fails, so a snapshot that
		// fails every invariant shows which snapshot a check reads.
		fails := func(assert func()) bool { return len(gomega.InterceptGomegaFailures(assert)) > 0 }
		readsOf := func(assert func(before, after invariants.Observed)) []Phase {
			var reads []Phase
			if fails(func() { assert(broken, healthy) }) {
				reads = append(reads, BeforeClose)
			}
			if fails(func() { assert(healthy, broken) }) {
				reads = append(reads, AfterClose)
			}
			return reads
		}
		var names []string
		invariantNamed := map[string]Invariant{}
		for _, invariant := range SubscriptionInvariants {
			names = append(names, invariant.Name)
			invariantNamed[invariant.Name] = invariant
		}
		if len(names) != len(want) {
			t.Fatalf("the subscription invariants are %v, want exactly the %d pinned here", names, len(want))
		}
		for name, phase := range want {
			invariant, found := invariantNamed[name]
			if !found {
				t.Fatalf("the subscription invariants hold no %s", name)
			}
			assert := func(before, after invariants.Observed) {
				(&Observation{before: before, after: after}).assertInvariant(invariant)
			}
			if fails(func() { assert(healthy, healthy) }) {
				t.Fatalf("%s fails on the healthy snapshot, so the fixture cannot show which snapshot it reads", name)
			}
			current := readsOf(assert)
			if !slices.Equal(current, []Phase{phase}) {
				t.Errorf("the runner's %s reads the snapshots %v, want only %v", name, current, phase)
			}
		}
		observed := &Observation{outcome: Outcome{Rules: Context{Value: 7}}}
		if read := observed.Context(); read.Value != 7 {
			t.Errorf("the runner handed a rule the context %+v, want the workload's", read)
		}
	})
}

// healthySnapshot passes every invariant: one session and one
// subscription on the connected server, the one produced value
// received once, the sentinel received, and the fault fired.
func healthySnapshot() invariants.Observed {
	return invariants.Observed{
		Produced:            []invariants.Produced{{Value: 1, SubscriptionID: 1, SequenceNumber: 1, Reachable: true, SubscriptionInstance: 1}},
		Received:            []int32{1},
		Servers:             []invariants.ServerState{{Reachable: true, Connected: true, KnownSessions: 1, ClosingAttempted: 1, LiveSubscriptions: 1}},
		ClientSubscriptions: 1,
		FaultEnd:            time.Unix(1, 0),
		Sentinel:            &invariants.Sentinel{Value: 1, AnsweredAt: time.Unix(1, 0), ReceivedAt: time.Unix(2, 0)},
		Fired:               true,
	}
}

// brokenSnapshot fails every invariant: two open sessions and no live
// subscription on the connected server, a value received twice and
// out of its produced order, no sentinel, and the fault never fired.
func brokenSnapshot() invariants.Observed {
	return invariants.Observed{
		Produced: []invariants.Produced{
			{Value: 1, SubscriptionID: 1, SequenceNumber: 2, Reachable: true, SubscriptionInstance: 1},
			{Value: 2, SubscriptionID: 1, SequenceNumber: 1, Reachable: true, SubscriptionInstance: 1},
		},
		Received:            []int32{1, 2, 2},
		Servers:             []invariants.ServerState{{Reachable: true, Connected: true, KnownSessions: 2}},
		ClientSubscriptions: 1,
	}
}

func TestPlanLabels(t *testing.T) {
	publishFault := namedFault(t, "RequestLost/Publish")
	cases := casesOf(t, []fault.Fault{publishFault},
		scenarioOf("A", 1, []message.Message{message.Publish}, standInShall, standInShould))
	check := checkNamed(t, cases, []string{"P4-1", "A", "RequestLost/Publish"}, "StandInShould")
	if !contains(check.Labels, "P4-1") {
		t.Errorf("a rule check misses its suite's clause label: %v", check.Labels)
	}
	if !contains(check.Labels, "P4-6.7") {
		t.Errorf("a rule check misses the rule's own clause label: %v", check.Labels)
	}
	if !contains(check.Labels, "should") {
		t.Errorf("a should-rule check misses the should label: %v", check.Labels)
	}
	check = checkNamed(t, cases, []string{"P4-1", "A", "RequestLost/Publish"}, "StandInShall")
	if contains(check.Labels, "should") {
		t.Errorf("a shall-rule check carries the should label: %v", check.Labels)
	}
	check = checkNamed(t, cases, []string{"P4-1", "A", "RequestLost/Publish"}, "HaveFired")
	if !contains(check.Labels, "invariant") {
		t.Errorf("an invariant check misses the invariant label: %v", check.Labels)
	}
	if !contains(check.Labels, "fault-message") {
		t.Errorf("a message fault's check misses fault-message: %v", check.Labels)
	}
	if !contains(check.Labels, "message-Publish") {
		t.Errorf("a Publish fault's check misses message-Publish: %v", check.Labels)
	}
}

func contains(labels []string, want string) bool {
	for _, label := range labels {
		if label == want {
			return true
		}
	}
	return false
}

func TestPlanLabelsOnlyMatchingDefects(t *testing.T) {
	publishFault := namedFault(t, "RequestLost/Publish")
	stall := namedFault(t, "Link/Stall")
	scenario := scenarioOf("A", 1, []message.Message{message.Publish})
	scenario.KnownDefects = []KnownDefect{{Issue: "issue-879", Check: "HaveFired", Applies: FaultsNamed("RequestLost/Publish")}}
	cases := casesOf(t, []fault.Fault{publishFault, stall}, scenario)
	for _, c := range cases {
		for _, check := range c.Checks {
			if check.Name != "HaveFired" {
				if contains(check.Labels, "known-defect") {
					t.Errorf("the defect labelled check %s of case %v", check.Name, c.Path)
				}
				continue
			}
			if strings.Join(c.Path, "/") == "P4-1/A/RequestLost/Publish" {
				if !contains(check.Labels, "known-defect") || !contains(check.Labels, "issue-879") {
					t.Errorf("the defect's labels are missing on the matching check: %v", check.Labels)
				}
			} else if contains(check.Labels, "known-defect") {
				t.Errorf("the defect labelled the same check in case %v", c.Path)
			}
		}
	}
}

func TestPlanRejectsBadDefects(t *testing.T) {
	readFault := namedFault(t, "RequestLost/Read")
	stall := namedFault(t, "Link/Stall")

	// a defect whose Applies is true only on skipped cases
	scenario := scenarioOf("A", 1, []message.Message{message.Read})
	scenario.KnownDefects = []KnownDefect{{Issue: "issue-879", Check: "HaveFired", Applies: FaultsNamed("RequestLost/Publish")}}
	_, err := scenario.Cases([]fault.Fault{readFault, stall, namedFault(t, "RequestLost/Publish")})
	if err == nil || !strings.Contains(err.Error(), "issue-879") {
		t.Fatalf("a defect matching only skipped cases returned %v, want an error naming it", err)
	}

	// a defect whose Check names no check
	scenario.KnownDefects = []KnownDefect{{Issue: "issue-900", Check: "NoSuchCheck", Applies: EveryFault}}
	_, err = scenario.Cases([]fault.Fault{readFault, stall})
	if err == nil || !strings.Contains(err.Error(), "issue-900") {
		t.Fatalf("a defect matching no check returned %v, want an error naming it", err)
	}

	// an ordinal left at zero
	scenario = scenarioOf("A", 0, []message.Message{message.Read})
	_, err = scenario.Cases([]fault.Fault{stall})
	if err == nil || !strings.Contains(err.Error(), "ordinal 0") {
		t.Fatalf("a zero ordinal returned %v, want an error naming it", err)
	}
}

func TestPlanRejectsDuplicateScenarioNames(t *testing.T) {
	// The fixture must not leak into the registrations of later tests
	// in this binary: the self-spec registers its own scenarios.
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

func TestUnfiledRecords(t *testing.T) {
	const slug = "eats-notification-under-load"
	const text = "the client eats a notification under load"

	t.Run("derives the label from the slug", func(t *testing.T) {
		if label := Unfiled(slug, text); label != "unfiled-"+slug {
			t.Fatalf("Unfiled returned %q, want %q", label, "unfiled-"+slug)
		}
	})
	// The fixture must not leak into the listings of later tests in this
	// binary: UnfiledDefects is the PR body's source of real defects.
	t.Cleanup(func() {
		unfiledMu.Lock()
		delete(unfiledTexts, "unfiled-"+slug)
		unfiledMu.Unlock()
	})
	t.Run("rejects a malformed slug", func(t *testing.T) {
		g := gomega.NewWithT(t)
		for _, bad := range []string{"Eats", "", "-eats", "double--hyphen", "under_score", "has space"} {
			g.Expect(func() { Unfiled(bad, text) }).To(gomega.Panic(), "Unfiled accepted the malformed slug %q", bad)
		}
	})
	t.Run("re-registers one slug and text without panicking", func(t *testing.T) {
		if label := Unfiled(slug, text); label != "unfiled-"+slug {
			t.Fatalf("re-registration returned %q, want the same label %q", label, "unfiled-"+slug)
		}
	})
	t.Run("rejects one slug under a second text", func(t *testing.T) {
		g := gomega.NewWithT(t)
		Unfiled(slug, text)
		g.Expect(func() { Unfiled(slug, text+" again") }).To(gomega.Panic(), "Unfiled re-registered a second text under the slug %q", slug)
	})
	t.Run("lists the recorded texts sorted by label", func(t *testing.T) {
		listed := UnfiledDefects()
		var labels []string
		for _, entry := range listed {
			label, _, found := strings.Cut(entry, ": ")
			if !found {
				t.Fatalf("UnfiledDefects listed an entry without a label: %q", entry)
			}
			labels = append(labels, label)
		}
		if !slices.IsSorted(labels) {
			t.Fatalf("UnfiledDefects is not sorted by label: %v", listed)
		}
		if !slices.Contains(listed, "unfiled-"+slug+": "+text) {
			t.Fatalf("UnfiledDefects is missing the recorded text under its label: %v", listed)
		}
	})
}
