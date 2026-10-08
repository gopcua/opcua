package matrix

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Observation is one case of a scenario as the clause file registers
// it: the planned case, and what the case's BeforeAll observed, for
// the check Its to read.
type Observation struct {
	scenario Scenario
	fault    fault.Fault
	planned  Case
	before   Observed
	after    Observed
	outcome  Outcome
	// written lists the checks whose Its asked for their labels, in
	// the order they registered.
	written []string
}

// Run registers one case of the scenario: when the fault does not
// apply, one It that skips with the reason; otherwise a BeforeAll that
// runs the workload and takes a snapshot before and after the client
// closes. The clause file then registers the case's check Its with
// the returned Observation: BeforeCloseInvariants, one It per rule
// that Applies, then AfterCloseInvariants.
func Run(s Scenario, f fault.Fault) *Observation {
	o := &Observation{scenario: s, fault: f, planned: s.caseOf(f)}
	c := o.planned
	if c.Skip != nil {
		ginkgo.It("does not apply: "+c.Skip.Text, func() {
			ginkgo.Skip(c.Skip.Text)
		})
		return o
	}

	ginkgo.BeforeAll(func() {
		var opts []harness.Option
		if s.Options != nil {
			opts = append(opts, s.Options(f, c.Block)...)
		}
		opts = append(opts, f.Options()...)
		env := harness.New(ginkgo.GinkgoT(), opts...)

		o.outcome = s.Workload(env, f, c.Block)

		// Settle: the sentinel reaches the client before the
		// snapshot reads it, and the case ends with the client
		// reporting Connected, so the teardown closes a live
		// client instead of racing its reconnect Dial (#883).
		waitForSentinel(env, o.outcome.Sentinel)
		waitUntilConnected(env)

		o.before = Observe(env, o.outcome.Injected)
		o.before.WithFaultEnd(o.outcome.FaultEnd)
		o.before.WithSentinel(o.outcome.Sentinel, o.outcome.AnsweredAt)

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		gomega.Expect(env.Client.Close(ctx)).To(gomega.Succeed(), "closing the client failed")

		// A delayed request is released only after Close
		// returns: the after-close snapshot must wait for
		// the hold, or a fault on the held message reads
		// as never fired.
		if isHoldFault(f) {
			deadline := time.Now().Add(10 * time.Second)
			for !o.outcome.Injected.Fired() && time.Now().Before(deadline) {
				time.Sleep(50 * time.Millisecond)
			}
		}

		o.after = Observe(env, o.outcome.Injected)
		o.after.WithFaultEnd(o.outcome.FaultEnd)
		o.after.WithSentinel(o.outcome.Sentinel, o.outcome.AnsweredAt)
	})
	return o
}

// BeforeCloseInvariants registers one It per invariant of the
// scenario that reads the snapshot taken before the client closes.
func (o *Observation) BeforeCloseInvariants() {
	o.invariantsIn(BeforeClose)
}

// haveFiredInvariant asserts on the after-close snapshot that the
// fault armed for the case fired.
var haveFiredInvariant = Invariant{
	Name:  "HaveFired",
	Phase: AfterClose,
	Assert: func(observed Observed) {
		gomega.Expect(observed.Fired).To(gomega.BeTrue(), "the fault never fired")
	},
}

// AfterCloseInvariants registers the HaveFired check, then one It per
// invariant of the scenario that reads the snapshot taken after the
// client closes. It is the case's last call, so it panics when the Its
// registered so far differ from the case's planned checks, which carry
// the known-defect labels.
func (o *Observation) AfterCloseInvariants() {
	if o.planned.Skip == nil {
		o.invariant(haveFiredInvariant)
	}
	o.invariantsIn(AfterClose)
	var planned []string
	for _, check := range o.planned.Checks {
		planned = append(planned, check.Name)
	}
	if !slices.Equal(o.written, planned) {
		panic(fmt.Sprintf("scenario %s under %s registers the checks %v, but plans %v",
			strings.Join(o.planned.Path[:2], "/"), o.planned.Path[2], o.written, planned))
	}
}

func (o *Observation) invariantsIn(phase Phase) {
	if o.planned.Skip != nil {
		return
	}
	for _, invariant := range o.scenario.Invariants {
		if invariant.Phase == phase {
			o.invariant(invariant)
		}
	}
}

// invariant registers one invariant's It, on the snapshot its phase
// names.
func (o *Observation) invariant(invariant Invariant) {
	ginkgo.It(invariant.Name, o.Labels(invariant.Name), func() {
		o.assertInvariant(invariant)
	})
}

// assertInvariant asserts the invariant on the snapshot its phase
// names.
func (o *Observation) assertInvariant(invariant Invariant) {
	if invariant.Phase == BeforeClose {
		invariant.Assert(o.before)
		return
	}
	invariant.Assert(o.after)
}

// Applies says whether the scenario's Rules prescribe the rule under
// the case's fault, so the clause file registers its It.
func (o *Observation) Applies(rule Rule) bool {
	if o.planned.Skip != nil {
		return false
	}
	return slices.ContainsFunc(o.scenario.rulesUnder(o.fault), func(r Rule) bool { return r.Name == rule.Name })
}

// Labels returns the labels of the case's check named name: its
// clause, fault and message labels, and the known-defect labels the
// scenario predicts for it. It panics when the case plans no such
// check.
func (o *Observation) Labels(name string) ginkgo.Labels {
	for _, check := range o.planned.Checks {
		if check.Name == name {
			o.written = append(o.written, name)
			return ginkgo.Label(check.Labels...)
		}
	}
	panic(fmt.Sprintf("scenario %s under %s plans no check named %s",
		strings.Join(o.planned.Path[:2], "/"), o.planned.Path[2], name))
}

// Context returns the rules' context the workload returned. A rule's
// It reads it, after the BeforeAll ran.
func (o *Observation) Context() Context {
	return o.outcome.Rules
}

// waitForSentinel waits up to 15 s for the client to receive the
// sentinel, so the snapshot's ResumePublishing reads a delivery that
// happened rather than one still in flight.
func waitForSentinel(env *harness.Environment, sentinel int32) {
	if sentinel == 0 {
		return
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		for _, value := range env.Received() {
			if value == sentinel {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// waitUntilConnected waits up to 15 s for the client to report
// Connected, as the case runner's settle step.
func waitUntilConnected(env *harness.Environment) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		states := env.States()
		if len(states) > 0 && states[len(states)-1] == opcua.Connected {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// isHoldFault says whether the fault holds a message past the moment
// Close returns, so the after-close snapshot must wait for its
// release.
func isHoldFault(f fault.Fault) bool {
	return strings.HasPrefix(f.Name(), "DelayAboveTimeout/") || strings.HasPrefix(f.Name(), "DelayBelowTimeout/")
}
