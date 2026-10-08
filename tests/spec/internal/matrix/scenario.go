package matrix

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/invariants"
	"github.com/gopcua/opcua/tests/spec/internal/message"
	"github.com/gopcua/opcua/tests/spec/internal/rules"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Scenario is one workload a clause file crosses with the fault
// catalogue. A clause file declares it as a value and registers it
// with ginkgo.DescribeTableSubtree, Entries and Run.
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
	// Options returns the harness options one case needs. block is
	// the first number of the case's value block.
	Options func(f fault.Fault, block int32) []harness.Option
	// Workload drives the client through one case and injects the
	// fault at its arm point.
	Workload func(env *harness.Environment, f fault.Fault, block int32) Outcome
	// Rules returns the rules the clause prescribes under the fault.
	// None means the case asserts the invariants only.
	Rules func(f fault.Fault) []rules.Rule
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
	ordinalsMu sync.Mutex
	ordinals   = make(map[int]string)
)

// Entries returns one ginkgo.Entry per fault for the scenario's
// ginkgo.DescribeTableSubtree. The entry's text is the fault's name.
// An entry whose fault applies is Ordered and ContinueOnFailure, so
// its checks share one run of the workload, and one failing check
// retires only itself. It panics when the scenario's cases cannot be
// planned or when another scenario already holds its ordinal.
func Entries(s Scenario, faults ...fault.Fault) []ginkgo.TableEntry {
	if _, err := s.Cases(faults); err != nil {
		panic(err)
	}
	ordinalsMu.Lock()
	owner, taken := ordinals[s.Ordinal]
	if taken && owner != s.Clause+"/"+s.Name {
		ordinalsMu.Unlock()
		panic(fmt.Sprintf("scenario %s/%s has ordinal %d, which %s already holds", s.Clause, s.Name, s.Ordinal, owner))
	}
	ordinals[s.Ordinal] = s.Clause + "/" + s.Name
	ordinalsMu.Unlock()
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

// Run registers one case of the scenario: when the fault does not
// apply, one It that skips with the reason; otherwise a BeforeAll that
// runs the workload and takes a snapshot before and after the client
// closes, and one It per check with its labels.
func Run(s Scenario, f fault.Fault) {
	c := s.caseOf(f)
	if c.Skip != nil {
		ginkgo.It("does not apply: "+c.Skip.Text, func() {
			ginkgo.Skip(c.Skip.Text)
		})
		return
	}
	var before invariants.Observed
	var after invariants.Observed
	var outcome Outcome

	ginkgo.BeforeAll(func() {
		opts := append([]harness.Option{}, s.Options(f, c.Block)...)
		opts = append(opts, f.Options()...)
		env := harness.New(ginkgo.GinkgoT(), opts...)

		outcome = s.Workload(env, f, c.Block)

		// Settle: the sentinel reaches the client before the
		// snapshot reads it, and the case ends with the client
		// reporting Connected, so the teardown closes a live
		// client instead of racing its reconnect Dial (#883).
		waitForSentinel(env, outcome.Sentinel)
		waitUntilConnected(env)

		before = invariants.Observe(env, outcome.Injected)
		before.WithFaultEnd(outcome.FaultEnd)
		before.WithSentinel(outcome.Sentinel, outcome.AnsweredAt)

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		gomega.Expect(env.Client.Close(ctx)).To(gomega.Succeed(), "closing the client failed")

		// A delayed request is released only after Close
		// returns: the after-close snapshot must wait for
		// the hold, or a fault on the held message reads
		// as never fired.
		if isHoldFault(f) {
			deadline := time.Now().Add(10 * time.Second)
			for !outcome.Injected.Fired() && time.Now().Before(deadline) {
				time.Sleep(50 * time.Millisecond)
			}
		}

		after = invariants.Observe(env, outcome.Injected)
		after.WithFaultEnd(outcome.FaultEnd)
		after.WithSentinel(outcome.Sentinel, outcome.AnsweredAt)
	})

	for _, check := range c.Checks {
		assert := s.assertion(check.Name, f)
		ginkgo.It(check.Name, ginkgo.Label(check.Labels...), func() {
			assert(before, after, outcome)
		})
	}
}

// assertion returns what the check named name asserts: an invariant
// reads the snapshot its phase names, a rule reads the context the
// workload returned.
func (s Scenario) assertion(name string, f fault.Fault) func(before, after invariants.Observed, outcome Outcome) {
	for _, invariant := range s.Invariants {
		if invariant.Name != name {
			continue
		}
		if invariant.Phase == BeforeClose {
			return func(before, _ invariants.Observed, _ Outcome) { invariant.Assert(before) }
		}
		return func(_, after invariants.Observed, _ Outcome) { invariant.Assert(after) }
	}
	for _, rule := range s.rulesUnder(f) {
		if rule.Name == name {
			return func(_, _ invariants.Observed, outcome Outcome) { rule.Check(outcome.Rules) }
		}
	}
	panic("scenario " + s.Name + " has no check named " + name)
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
func (s Scenario) rulesUnder(f fault.Fault) []rules.Rule {
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
