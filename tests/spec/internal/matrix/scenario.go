package matrix

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/invariants"
	"github.com/gopcua/opcua/tests/spec/internal/message"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Scenario is one workload a clause file crosses with the fault
// catalogue. A clause file declares it as a value and registers it
// with ginkgo.DescribeTableSubtree, Entries and Run, and writes the
// case's check Its with the Observation that Run returns.
type Scenario struct {
	// Clause is the text of the clause container and the first label
	// of every check, "P4-6.7" for example.
	Clause string
	// Name is the text of the scenario container.
	Name string
	// Ordinal numbers the scenario's value blocks, counting from 1.
	// Two scenarios of one test binary never share an ordinal.
	Ordinal int
	// Sends lists the messages a correct client sends after the arm
	// point. A fault on any other message does not apply.
	Sends []message.Message
	// Options returns the harness options one case needs, none when it
	// is nil. block is the first number of the case's value block.
	Options func(f fault.Fault, block int32) []harness.Option
	// Workload drives the client through one case and injects the
	// fault at its arm point.
	Workload func(env *harness.Environment, f fault.Fault, block int32) Outcome
	// Rules returns the rules the clause prescribes under the fault.
	// None means the case asserts the invariants only.
	Rules func(f fault.Fault) []Rule
	// Invariants are the checks the workload owes under every fault,
	// because of what kind of workload it is.
	Invariants []Invariant
	// KnownDefects are the checks of this scenario measured to fail.
	KnownDefects []KnownDefect
}

// KnownDefect predicts that one check of a scenario fails: the label
// of the issue that names the defect, the check, and the faults under
// which it fails.
type KnownDefect struct {
	Issue   string
	Check   string
	Applies func(f fault.Fault) bool
}

// Invariant is one property a kind of workload owes under every fault:
// its name, the snapshot it reads and the assertion on that snapshot.
type Invariant struct {
	Name   string
	Phase  Phase
	Assert func(observed invariants.Observed)
}

// SubscriptionInvariants are the invariants of a workload that
// publishes on a subscription and answers a sentinel last.
var SubscriptionInvariants = []Invariant{
	{Name: "ResumePublishing", Phase: BeforeClose, Assert: func(observed invariants.Observed) {
		gomega.Expect(observed).To(invariants.ResumePublishing(15*time.Second), "the sentinel did not resume publishing within its window")
	}},
	{Name: "KeepOneSessionOpen", Phase: BeforeClose, Assert: func(observed invariants.Observed) {
		gomega.Expect(observed).To(invariants.KeepOneSessionOpen(), "the connected server holds the wrong session count")
	}},
	{Name: "KeepOneSubscriptionPerClientSubscription", Phase: BeforeClose, Assert: func(observed invariants.Observed) {
		gomega.Expect(observed).To(invariants.KeepOneSubscriptionPerClientSubscription(), "the live subscriptions do not match the client's")
	}},
	{Name: "HaveFired", Phase: AfterClose, Assert: func(observed invariants.Observed) {
		gomega.Expect(observed.Fired).To(gomega.BeTrue(), "the fault never fired")
	}},
	{Name: "DeliverEachValueOnce", Phase: AfterClose, Assert: func(observed invariants.Observed) {
		gomega.Expect(observed).To(invariants.DeliverEachValueOnce(), "a value was not delivered exactly once")
	}},
	{Name: "DeliverInOrder", Phase: AfterClose, Assert: func(observed invariants.Observed) {
		gomega.Expect(observed).To(invariants.DeliverInOrder(), "values were delivered out of order")
	}},
	{Name: "CloseEveryKnownSession", Phase: AfterClose, Assert: func(observed invariants.Observed) {
		gomega.Expect(observed).To(invariants.CloseEveryKnownSession(), "a reachable server still holds a session")
	}},
}

var (
	registeredMu sync.Mutex
	// ordinalOwner maps each ordinal to the clause and name of the
	// scenario that holds it, and ordinalOf the reverse.
	ordinalOwner = make(map[int]string)
	ordinalOf    = make(map[string]int)
)

// register records the scenario's ordinal and name for this test
// binary. It returns an error when another scenario holds the ordinal
// or another scenario of the clause has the name.
func register(s Scenario) error {
	registeredMu.Lock()
	defer registeredMu.Unlock()
	id := s.Clause + "/" + s.Name
	if owner, taken := ordinalOwner[s.Ordinal]; taken && owner != id {
		return fmt.Errorf("scenario %s has ordinal %d, which %s already holds", id, s.Ordinal, owner)
	}
	if ordinal, named := ordinalOf[id]; named && ordinal != s.Ordinal {
		return fmt.Errorf("two scenarios are named %s, with ordinals %d and %d", id, ordinal, s.Ordinal)
	}
	ordinalOwner[s.Ordinal] = id
	ordinalOf[id] = s.Ordinal
	return nil
}

// Entries returns one ginkgo.Entry per fault for the scenario's
// ginkgo.DescribeTableSubtree. The entry's text is the fault's name.
// An entry whose fault applies is Ordered and ContinueOnFailure, so
// its checks share one run of the workload, and one failing check
// retires only itself. It panics when the scenario's cases cannot be
// planned, when another scenario already holds its ordinal, or when
// another scenario of the clause has its name.
func Entries(s Scenario, faults ...fault.Fault) []ginkgo.TableEntry {
	if _, err := s.Cases(faults); err != nil {
		panic(err)
	}
	if err := register(s); err != nil {
		panic(err)
	}
	var entries []ginkgo.TableEntry
	for _, f := range faults {
		if f.Available(s.Sends) != nil {
			entries = append(entries, ginkgo.Entry(f.Name(), f))
			continue
		}
		entries = append(entries, ginkgo.Entry(f.Name(), f, ginkgo.Ordered, ginkgo.ContinueOnFailure))
	}
	return entries
}

// Cases plans the scenario over the faults: one case per fault, each
// with its value block and its checks in the order they run. It
// returns an error when the ordinal is not positive or a known defect
// labels no check of an applicable case.
func (s Scenario) Cases(faults []fault.Fault) ([]Case, error) {
	if s.Ordinal < 1 {
		return nil, fmt.Errorf("scenario %s/%s has ordinal %d; ordinals count from 1", s.Clause, s.Name, s.Ordinal)
	}
	var cases []Case
	for _, f := range faults {
		cases = append(cases, s.caseOf(f))
	}
	for _, defect := range s.KnownDefects {
		matched := false
		for i, c := range cases {
			if c.Skip != nil || !defect.Applies(faults[i]) {
				continue
			}
			if slices.ContainsFunc(c.Checks, func(check Check) bool { return check.Name == defect.Check }) {
				matched = true
			}
		}
		if !matched {
			return nil, fmt.Errorf("known defect %s (check %s) matches no applicable check", defect.Issue, defect.Check)
		}
	}
	return cases, nil
}

// caseOf plans one case: its path, its value block, and either the
// reason the fault does not apply or its checks. The invariants read
// before the client closes come first, then the rules, then the
// invariants read after it closes.
func (s Scenario) caseOf(f fault.Fault) Case {
	c := Case{Path: []string{s.Clause, s.Name, f.Name()}, Block: s.block(f)}
	if reason := f.Available(s.Sends); reason != nil {
		c.Skip = reason
		return c
	}
	base := []string{s.Clause, "fault-" + faultGroup(f)}
	if target, named := faultTarget(f); named {
		base = append(base, "message-"+target)
	}
	invariantsIn := func(phase Phase) {
		for _, invariant := range s.Invariants {
			if invariant.Phase == phase {
				c.Checks = append(c.Checks, Check{Name: invariant.Name, Labels: withLabels(base, "invariant"), Phase: phase})
			}
		}
	}
	invariantsIn(BeforeClose)
	for _, rule := range s.rulesUnder(f) {
		labels := append(withLabels(base, rule.Clause), rule.Name)
		if rule.Keyword == "should" {
			labels = append(labels, "should")
		}
		c.Checks = append(c.Checks, Check{Name: rule.Name, Labels: labels, Phase: BeforeClose})
	}
	invariantsIn(AfterClose)
	for i := range c.Checks {
		for _, defect := range s.KnownDefects {
			if defect.Check == c.Checks[i].Name && defect.Applies(f) {
				c.Checks[i].Labels = append(c.Checks[i].Labels, "known-defect", defect.Issue)
			}
		}
	}
	return c
}

// rulesUnder returns the rules the clause prescribes under the fault,
// none when the scenario declares no Rules.
func (s Scenario) rulesUnder(f fault.Fault) []Rule {
	if s.Rules == nil {
		return nil
	}
	return s.Rules(f)
}

// block returns the first number of the case's value block: 1000 times
// the case's ordinal among every scenario × fault pair, so every value
// a case answers is its block plus an offset below 1000, and no two
// cases share a value.
func (s Scenario) block(f fault.Fault) int32 {
	faultOrdinal := slices.IndexFunc(fault.AllFaults, func(candidate fault.Fault) bool {
		return candidate.Name() == f.Name()
	})
	return int32(1000 * ((s.Ordinal-1)*len(fault.AllFaults) + faultOrdinal + 1))
}

// EveryFault says a known defect applies under every fault.
func EveryFault(fault.Fault) bool { return true }

// EveryFaultExcept says a known defect applies under every fault
// except the named ones.
func EveryFaultExcept(names ...string) func(fault.Fault) bool {
	return func(f fault.Fault) bool {
		return !slices.Contains(names, f.Name())
	}
}

// FaultsNamed says a known defect applies under the named faults.
func FaultsNamed(names ...string) func(fault.Fault) bool {
	return func(f fault.Fault) bool {
		return slices.Contains(names, f.Name())
	}
}

// FaultsTargetingExcept says a known defect applies under every
// message or overload fault that targets one of the named services,
// except the named faults.
func FaultsTargetingExcept(except []string, services ...string) func(fault.Fault) bool {
	return func(f fault.Fault) bool {
		if slices.Contains(except, f.Name()) {
			return false
		}
		parts := strings.Split(f.Name(), "/")
		if len(parts) < 2 {
			return false
		}
		switch parts[0] {
		case "RequestLost", "ResponseLost", "CutAfterResponse", "DelayBelowTimeout", "DelayAboveTimeout", "Overload":
			return slices.Contains(services, parts[1])
		}
		return false
	}
}
