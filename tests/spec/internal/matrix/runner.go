package matrix

import (
	"context"
	"strings"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/invariants"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

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
		var opts []harness.Option
		if s.Options != nil {
			opts = append(opts, s.Options(f, c.Block)...)
		}
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
