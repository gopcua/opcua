package matrix

import (
	"context"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/tests/spec/faults"
	"github.com/gopcua/opcua/tests/spec/invariants"
	"github.com/gopcua/opcua/tests/spec/rules"
	"github.com/gopcua/opcua/tests/spec/spectest"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// registered holds the suites RegisterSuites has seen, so the case
// runner can find the scenario and fault a case's path names.
var registered []Suite

// RegisterSuites turns the plan into Ginkgo nodes: ginkgo.Describe(clause) ›
// ginkgo.Describe(scenario) › ginkgo.Describe(fault, ginkgo.Ordered). A skipped case is one
// It that skips with the fault's reason; every check of an applicable
// case is one It with its labels, reading the snapshots the case's
// BeforeAll took. It panics when the plan cannot be built.
func RegisterSuites(suites []Suite, defects []KnownDefect, all ...faults.Fault) {
	registered = append(registered, suites...)
	cases, err := Plan(suites, all, defects)
	if err != nil {
		panic(err)
	}
	for _, c := range cases {
		registerCase(c)
	}
}

func registerCase(c Case) {
	path := c.Path
	scenario := scenarioNamed(path[0], path[1])
	f := faultNamed(path[2])
	if c.Skip != nil {
		ginkgo.Describe(path[0], func() {
			ginkgo.Describe(path[1], func() {
				ginkgo.Describe(path[2], func() {
					ginkgo.It("does not apply: "+c.Skip.Text, func() {
						ginkgo.Skip(c.Skip.Text)
					})
				})
			})
		})
		return
	}
	ginkgo.Describe(path[0], func() {
		ginkgo.Describe(path[1], func() {
			ginkgo.Describe(path[2], ginkgo.Ordered, func() {
				var before invariants.Observed
				var after invariants.Observed
				var outcome Outcome

				ginkgo.BeforeAll(func() {
					opts := append([]spectest.Option{}, scenario.Options()...)
					opts = append(opts, f.Options()...)
					env := spectest.Start(ginkgo.GinkgoT(), opts...)

					outcome = scenario.Run(env, f)

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

					after = invariants.Observe(env, outcome.Injected)
					after.WithFaultEnd(outcome.FaultEnd)
					after.WithSentinel(outcome.Sentinel, outcome.AnsweredAt)
				})

				for _, check := range c.Checks {
					check := check
					ginkgo.It(check.Name, ginkgo.Label(check.Labels...), func() {
						assertCheck(check, before, after, outcome)
					})
				}
			})
		})
	})
}

func assertCheck(check Check, before, after invariants.Observed, outcome Outcome) {
	switch check.Name {
	case "HaveFired":
		gomega.Expect(before.Fired).To(gomega.BeTrue(), "the fault never fired")
	case "ResumePublishing":
		gomega.Expect(before).To(invariants.ResumePublishing(15*time.Second), "the sentinel did not resume publishing within its window")
	case "KeepOneSessionOpen":
		gomega.Expect(before).To(invariants.KeepOneSessionOpen(), "the connected server holds the wrong session count")
	case "KeepOneSubscriptionPerClientSubscription":
		gomega.Expect(before).To(invariants.KeepOneSubscriptionPerClientSubscription(), "the live subscriptions do not match the client's")
	case "DeliverEachValueOnce":
		gomega.Expect(after).To(invariants.DeliverEachValueOnce(), "a value was not delivered exactly once")
	case "DeliverInOrder":
		gomega.Expect(after).To(invariants.DeliverInOrder(), "values were delivered out of order")
	case "CloseEveryKnownSession":
		gomega.Expect(after).To(invariants.CloseEveryKnownSession(), "a reachable server still holds a session")
	default:
		ruleNamed(check.Name).Check(outcome.Rules)
	}
}

// waitForSentinel waits up to 15 s for the client to receive the
// sentinel, so the snapshot's ResumePublishing reads a delivery that
// happened rather than one still in flight.
func waitForSentinel(env *spectest.Environment, sentinel int32) {
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
func waitUntilConnected(env *spectest.Environment) {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		states := env.States()
		if len(states) > 0 && states[len(states)-1] == opcua.Connected {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func scenarioNamed(clause, name string) Scenario {
	for _, suite := range registered {
		if suite.Clause() != clause {
			continue
		}
		for _, scenario := range suite.Scenarios() {
			if scenario.Name() == name {
				return scenario
			}
		}
	}
	ginkgo.Fail("no scenario named " + name + " in suite " + clause)
	return nil
}

func faultNamed(name string) faults.Fault {
	for _, f := range faults.AllFaults {
		if f.Name() == name {
			return f
		}
	}
	panic("no fault named " + name)
}

func ruleNamed(name string) rules.Rule {
	for _, rule := range rules.All() {
		if rule.Name == name {
			return rule
		}
	}
	ginkgo.Fail("no rule named " + name)
	return rules.Rule{}
}
