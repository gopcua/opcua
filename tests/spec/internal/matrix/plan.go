// Package matrix runs the failure matrix: one plan decides, per suite,
// scenario and fault, which checks apply and which known defects
// label them; the registration turns the plan into Ginkgo nodes that
// run a real client through each case and assert the invariants and
// rules against what it observed.
package matrix

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/faults"
	"github.com/gopcua/opcua/tests/spec/internal/message"
	"github.com/gopcua/opcua/tests/spec/internal/rules"
	"github.com/gopcua/opcua/tests/spec/internal/spectest"
)

// Category names the rule set a clause prescribes for one fault in
// one scenario. The zero value is invalid: every category names a
// decision the standard makes.
type Category int

const (
	// categoryInvalid is the zero Category; Plan rejects it.
	categoryInvalid Category = iota
	// SessionSurvives: the transport was lost, the session survived.
	SessionSurvives
	// SessionLost: the session was lost and must be recreated.
	SessionLost
	// SubscriptionsLost: the subscriptions are gone and must be
	// recreated.
	SubscriptionsLost
	// ActivationFailed: the session's activation failed and must be
	// recreated.
	ActivationFailed
	// SteadyPublishing: publishing continues through the fault.
	SteadyPublishing
	// CancelThenSubscribe: the only subscription was cancelled and a
	// new one created while a Publish was held.
	CancelThenSubscribe
	// Unspecified: Part 4 prescribes no reaction; the invariants only.
	Unspecified
)

// Suite is one Part 4 clause with its scenarios and the rules its
// categories prescribe.
type Suite interface {
	Clause() string
	Scenarios() []Scenario
	Rules(Category) []rules.Rule
}

// Scenario is one workload of a suite: the messages a correct client
// sends after the arm point, the Start options it needs per fault, the
// category the clause prescribes per fault, and the Run that drives
// the client and injects the fault at its arm point.
type Scenario interface {
	Name() string
	Sends() []message.Message
	Options(f faults.Fault) []spectest.Option
	Category(faults.Fault) Category
	Run(env *spectest.Environment, f faults.Fault) Outcome
}

// Outcome is what a scenario's Run observed: the armed fault, the
// fault's end event, the sentinel value with the moment its answer
// was sent, and the rules' context.
type Outcome struct {
	Injected   *spectest.Injected
	FaultEnd   time.Time
	Sentinel   int32
	AnsweredAt time.Time
	Rules      rules.Context
}

// KnownDefect is one prediction that a check fails on this tree: the
// issue that names it, the check it labels and where it applies.
type KnownDefect struct {
	Issue   string
	Check   string
	Applies func(scenario string, f faults.Fault) bool
}

// Phase says whether a check runs before or after the client closes.
type Phase int

const (
	// phaseInvalid is the zero Phase; no check carries it.
	phaseInvalid Phase = iota
	// BeforeClose: the check reads the first snapshot, taken while the
	// client is still connected.
	BeforeClose
	// AfterClose: the check reads the second snapshot, taken after the
	// client closed.
	AfterClose
)

// Check is one assertion a case runs: an invariant's or a rule's
// name, the labels it carries and the phase it runs in.
type Check struct {
	Name   string
	Labels []string
	Phase  Phase
}

// Case is one suite × scenario × fault combination the plan decided:
// its path, its skip reason when the fault does not apply, and its
// checks.
type Case struct {
	Path   []string
	Skip   *faults.Reason
	Checks []Check
}

// Plan decides what the failure matrix runs: one case per suite ×
// scenario × fault, each with the checks of its category in order and
// the known defects that label them. It returns an error — never
// panics — for a defect that matches no applicable check, a zero
// category or two scenarios of one suite with the same name.
func Plan(suites []Suite, all []faults.Fault, defects []KnownDefect) ([]Case, error) {
	var cases []Case
	for _, suite := range suites {
		seen := make(map[string]bool)
		for _, scenario := range suite.Scenarios() {
			if seen[scenario.Name()] {
				return nil, fmt.Errorf("suite %s has two scenarios named %s", suite.Clause(), scenario.Name())
			}
			seen[scenario.Name()] = true
		}
		for _, scenario := range suite.Scenarios() {
			for _, f := range all {
				path := []string{suite.Clause(), scenario.Name(), f.Name()}
				if reason := f.Available(scenario.Sends()); reason != nil {
					cases = append(cases, Case{Path: path, Skip: reason})
					continue
				}
				category := scenario.Category(f)
				if category == Category(0) {
					return nil, fmt.Errorf("scenario %s returned a zero category for fault %s", scenario.Name(), f.Name())
				}
				checks, err := checksOf(suite, scenario, category, f, defects)
				if err != nil {
					return nil, err
				}
				cases = append(cases, Case{Path: path, Checks: checks})
			}
		}
	}
	if err := defectsMatch(defects, cases); err != nil {
		return nil, err
	}
	return cases, nil
}

func checksOf(suite Suite, scenario Scenario, category Category, f faults.Fault, defects []KnownDefect) ([]Check, error) {
	_ = categoryInvalid
	_ = phaseInvalid
	var checks []Check
	before := func(name string, labels ...string) {
		checks = append(checks, Check{Name: name, Labels: labels, Phase: BeforeClose})
	}
	after := func(name string, labels ...string) {
		checks = append(checks, Check{Name: name, Labels: labels, Phase: AfterClose})
	}
	base := []string{suite.Clause(), "fault-" + faultGroup(f)}
	if target, named := faultTarget(f); named {
		base = append(base, "message-"+target)
	}

	before("ResumePublishing", withLabels(base, "invariant")...)
	before("KeepOneSessionOpen", withLabels(base, "invariant")...)
	before("KeepOneSubscriptionPerClientSubscription", withLabels(base, "invariant")...)

	for _, rule := range suite.Rules(category) {
		labels := append(withLabels(base, rule.Clause), rule.Name)
		if rule.Keyword == "should" {
			labels = append(labels, "should")
		}
		checks = append(checks, Check{Name: rule.Name, Labels: labels, Phase: BeforeClose})
	}

	after("HaveFired", withLabels(base, "invariant")...)
	after("DeliverEachValueOnce", withLabels(base, "invariant")...)
	after("DeliverInOrder", withLabels(base, "invariant")...)
	after("CloseEveryKnownSession", withLabels(base, "invariant")...)

	for i := range checks {
		for _, defect := range defects {
			if defect.Check != checks[i].Name || !defect.Applies(scenario.Name(), f) {
				continue
			}
			checks[i].Labels = append(checks[i].Labels, "known-defect", defect.Issue)
		}
	}
	return checks, nil
}

func withLabels(base []string, extra ...string) []string {
	return append(append([]string{}, base...), extra...)
}

func faultGroup(f faults.Fault) string {
	group := f.Name()
	if index := strings.Index(group, "/"); index >= 0 {
		group = group[:index]
	}
	switch group {
	case "RequestLost", "ResponseLost", "CutAfterResponse", "DelayBelowTimeout", "DelayAboveTimeout":
		return "message"
	}
	return strings.ToLower(group)
}

// faultTarget returns the message name of a message-targeting fault
// and whether it names one.
func faultTarget(f faults.Fault) (string, bool) {
	name := f.Name()
	parts := strings.Split(name, "/")
	switch parts[0] {
	case "Overload", "DelayBelowTimeout", "DelayAboveTimeout", "RequestLost", "ResponseLost", "CutAfterResponse":
		if len(parts) < 2 {
			return "", false
		}
		return parts[1], true
	}
	return "", false
}

// defectsMatch verifies every defect labels at least one check of an
// applicable, non-skipped case.
func defectsMatch(defects []KnownDefect, cases []Case) error {
	for _, defect := range defects {
		matched := false
		for _, c := range cases {
			if c.Skip != nil {
				continue
			}
			if !defect.Applies(c.Path[1], namedFaultFor(c.Path[2])) {
				continue
			}
			for _, check := range c.Checks {
				if check.Name == defect.Check {
					matched = true
				}
			}
		}
		if !matched {
			return fmt.Errorf("known defect %s (check %s) matches no applicable check", defect.Issue, defect.Check)
		}
	}
	return nil
}

func namedFaultFor(name string) faults.Fault {
	for _, f := range faults.AllFaults {
		if f.Name() == name {
			return f
		}
	}
	return nil
}

var (
	unfiledMu    sync.Mutex
	unfiledTexts = make(map[string]string)
)

// slugPattern says which slugs Unfiled accepts.
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Unfiled records a finding no issue names yet and returns the label
// its defect carries, so the label names the defect and survives
// moving it to another file. The PR body lists UnfiledDefects so each
// one gets an issue or an explanation.
func Unfiled(slug, text string) string {
	unfiledMu.Lock()
	defer unfiledMu.Unlock()
	if !slugPattern.MatchString(slug) {
		panic("Unfiled: slug " + slug + " does not match " + slugPattern.String())
	}
	label := "unfiled-" + slug
	if old, recorded := unfiledTexts[label]; recorded && old != text {
		panic("Unfiled: slug " + slug + " already records " + old + ", not " + text)
	}
	unfiledTexts[label] = text
	return label
}

// UnfiledDefects lists every text Unfiled recorded, each paired with
// the label its defect carries, sorted by label.
func UnfiledDefects() []string {
	unfiledMu.Lock()
	defer unfiledMu.Unlock()
	var texts []string
	for _, label := range slices.Sorted(maps.Keys(unfiledTexts)) {
		texts = append(texts, label+": "+unfiledTexts[label])
	}
	return texts
}
