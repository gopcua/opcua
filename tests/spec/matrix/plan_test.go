package matrix

import (
	"strings"
	"testing"

	"github.com/gopcua/opcua/tests/spec/faults"
	"github.com/gopcua/opcua/tests/spec/message"
	"github.com/gopcua/opcua/tests/spec/rules"
	"github.com/gopcua/opcua/tests/spec/spectest"
)

// fakes for the unit tests: one suite with two scenarios, three faults,
// and defects the cases label.

type fakeScenario struct {
	name  string
	sends []message.Message
}

func (s fakeScenario) Name() string                     { return s.name }
func (s fakeScenario) Sends() []message.Message         { return s.sends }
func (s fakeScenario) Options() []spectest.Option       { return nil }
func (s fakeScenario) Category(f faults.Fault) Category { return SessionSurvives }
func (s fakeScenario) Run(env *spectest.Environment, f faults.Fault) Outcome {
	return Outcome{}
}

type fakeSuite struct {
	clause     string
	scenarios  []Scenario
	byCategory map[Category][]rules.Rule
}

func (s fakeSuite) Clause() string        { return s.clause }
func (s fakeSuite) Scenarios() []Scenario { return s.scenarios }
func (s fakeSuite) Rules(c Category) []rules.Rule {
	if c == Category(0) {
		panic("zero category")
	}
	return s.byCategory[c]
}

func namedFault(t *testing.T, name string) faults.Fault {
	t.Helper()
	for _, f := range faults.AllFaults {
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
	suite := fakeSuite{
		clause: "P4-1",
		scenarios: []Scenario{
			fakeScenario{name: "A", sends: []message.Message{message.Publish}},
			fakeScenario{name: "B", sends: []message.Message{message.Read}},
		},
		byCategory: map[Category][]rules.Rule{},
	}
	publishFault := namedFault(t, "RequestLost/Publish")
	readFault := namedFault(t, "RequestLost/Read")
	stallFault := namedFault(t, "Link/Stall")
	all := []faults.Fault{publishFault, readFault, stallFault}

	cases, err := Plan([]Suite{suite}, all, nil)
	if err != nil {
		t.Fatalf("Plan returned an error: %v", err)
	}
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
	suite := fakeSuite{
		clause:    "P4-1",
		scenarios: []Scenario{fakeScenario{name: "A", sends: []message.Message{message.Publish}}},
		byCategory: map[Category][]rules.Rule{
			SessionSurvives: {rules.ReactivatesSession, rules.CreatesNoSession},
		},
	}
	stall := namedFault(t, "Link/Stall")
	cases, err := Plan([]Suite{suite}, []faults.Fault{stall}, nil)
	if err != nil {
		t.Fatalf("Plan returned an error: %v", err)
	}
	if len(cases) != 1 {
		t.Fatalf("Plan built %d cases, want 1", len(cases))
	}
	got := make([]string, 0, len(cases[0].Checks))
	for _, check := range cases[0].Checks {
		got = append(got, check.Name)
	}
	want := []string{
		"HaveFired", "ResumePublishing", "KeepOneSessionOpen", "KeepOneSubscriptionPerClientSubscription",
		"ReactivatesSession", "CreatesNoSession",
		"DeliverEachValueOnce", "DeliverInOrder", "CloseEveryKnownSession",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("the checks are %v, want %v", got, want)
	}
	for _, check := range cases[0].Checks {
		switch check.Name {
		case "DeliverEachValueOnce", "DeliverInOrder", "CloseEveryKnownSession":
			if check.Phase != AfterClose {
				t.Errorf("%s has phase %d, want AfterClose", check.Name, check.Phase)
			}
		default:
			if check.Phase != BeforeClose {
				t.Errorf("%s has phase %d, want BeforeClose", check.Name, check.Phase)
			}
		}
	}
}

func TestPlanLabels(t *testing.T) {
	suite := fakeSuite{
		clause:    "P4-1",
		scenarios: []Scenario{fakeScenario{name: "A", sends: []message.Message{message.Publish}}},
		byCategory: map[Category][]rules.Rule{
			SessionSurvives: {rules.ReactivatesSession, rules.SendsNoPublishBeforeNotAvailable},
		},
	}
	publishFault := namedFault(t, "RequestLost/Publish")
	cases, err := Plan([]Suite{suite}, []faults.Fault{publishFault}, nil)
	if err != nil {
		t.Fatalf("Plan returned an error: %v", err)
	}
	check := checkNamed(t, cases, []string{"P4-1", "A", "RequestLost/Publish"}, "SendsNoPublishBeforeNotAvailable")
	if !contains(check.Labels, "P4-1") {
		t.Errorf("a rule check misses its suite's clause label: %v", check.Labels)
	}
	if !contains(check.Labels, "P4-6.7") {
		t.Errorf("a rule check misses the rule's own clause label: %v", check.Labels)
	}
	if !contains(check.Labels, "should") {
		t.Errorf("a should-rule check misses the should label: %v", check.Labels)
	}
	check = checkNamed(t, cases, []string{"P4-1", "A", "RequestLost/Publish"}, "ReactivatesSession")
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
	suite := fakeSuite{
		clause:     "P4-1",
		scenarios:  []Scenario{fakeScenario{name: "A", sends: []message.Message{message.Publish}}},
		byCategory: map[Category][]rules.Rule{},
	}
	publishFault := namedFault(t, "RequestLost/Publish")
	stall := namedFault(t, "Link/Stall")
	defect := KnownDefect{
		Issue: "issue-879",
		Check: "HaveFired",
		Applies: func(scenario string, f faults.Fault) bool {
			return f.Name() == "RequestLost/Publish"
		},
	}
	cases, err := Plan([]Suite{suite}, []faults.Fault{publishFault, stall}, []KnownDefect{defect})
	if err != nil {
		t.Fatalf("Plan returned an error: %v", err)
	}
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
	suite := fakeSuite{
		clause:     "P4-1",
		scenarios:  []Scenario{fakeScenario{name: "A", sends: []message.Message{message.Read}}},
		byCategory: map[Category][]rules.Rule{},
	}
	readFault := namedFault(t, "RequestLost/Read")
	stall := namedFault(t, "Link/Stall")

	// a defect whose Applies is true only on skipped cases
	skippedDefect := KnownDefect{
		Issue: "issue-879",
		Check: "HaveFired",
		Applies: func(scenario string, f faults.Fault) bool {
			return f.Name() == "RequestLost/Publish"
		},
	}
	_, err := Plan([]Suite{suite}, []faults.Fault{readFault, stall, namedFault(t, "RequestLost/Publish")}, []KnownDefect{skippedDefect})
	if err == nil || !strings.Contains(err.Error(), "issue-879") {
		t.Fatalf("a defect matching only skipped cases returned %v, want an error naming it", err)
	}

	// a defect whose Check names no check
	unknownDefect := KnownDefect{
		Issue: "issue-900",
		Check: "NoSuchCheck",
		Applies: func(scenario string, f faults.Fault) bool {
			return true
		},
	}
	_, err = Plan([]Suite{suite}, []faults.Fault{readFault, stall}, []KnownDefect{unknownDefect})
	if err == nil || !strings.Contains(err.Error(), "issue-900") {
		t.Fatalf("a defect matching no check returned %v, want an error naming it", err)
	}
}

func TestPlanRejectsDuplicateScenarioNames(t *testing.T) {
	suite := fakeSuite{
		clause: "P4-1",
		scenarios: []Scenario{
			fakeScenario{name: "A", sends: []message.Message{message.Publish}},
			fakeScenario{name: "A", sends: []message.Message{message.Read}},
		},
		byCategory: map[Category][]rules.Rule{},
	}
	_, err := Plan([]Suite{suite}, []faults.Fault{namedFault(t, "Link/Stall")}, nil)
	if err == nil || !strings.Contains(err.Error(), "A") {
		t.Fatalf("duplicate scenario names returned %v, want an error naming the name", err)
	}
}

func TestUnfiledRecords(t *testing.T) {
	label := Unfiled("the client eats a notification under load")
	if label == "" {
		t.Fatalf("Unfiled returned an empty label")
	}
	listed := UnfiledDefects()
	found := false
	for _, text := range listed {
		if strings.Contains(text, "the client eats a notification under load") && strings.HasPrefix(text, label+": ") {
			found = true
		}
	}
	if !found {
		t.Fatalf("UnfiledDefects is missing the recorded text under its label: %v", listed)
	}
}
