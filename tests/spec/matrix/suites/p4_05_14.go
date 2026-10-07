package suites

import (
	"context"
	"strings"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/tests/spec/faults"
	"github.com/gopcua/opcua/tests/spec/matrix"
	"github.com/gopcua/opcua/tests/spec/message"
	"github.com/gopcua/opcua/tests/spec/rules"
	"github.com/gopcua/opcua/tests/spec/spectest"
	"github.com/gopcua/opcua/ua"
)

// P4_05_14 returns the Part 4 §5.14 suite: two scenarios whose workloads
// drive a real client through the subscription publishing the clause
// prescribes — a steady stream the fault interrupts, and a subscription
// swap while a Publish is held. The fault is the interruption: the
// workload cuts no relay connection of its own.
func P4_05_14() matrix.Suite { return p4_05_14{} }

type p4_05_14 struct{}

func (p4_05_14) Clause() string { return "P4-5.14" }

func (p4_05_14) Scenarios() []matrix.Scenario {
	return []matrix.Scenario{
		scenario05_14{
			name: "SteadyPublishing",
			sends: []message.Message{
				message.Publish, message.CloseSession, message.CloseSecureChannel,
			},
			own:   matrix.SteadyPublishing,
			drive: driveSteadyPublishing,
		},
		scenario05_14{
			name: "CancelThenSubscribe",
			sends: []message.Message{
				message.DeleteSubscriptions, message.CreateSubscription, message.CreateMonitoredItems,
				message.Publish, message.CloseSession, message.CloseSecureChannel,
			},
			own:   matrix.CancelThenSubscribe,
			drive: driveCancelThenSubscribe,
		},
	}
}

// Rules is the rule set the clause prescribes per category: the steady
// stream must publish again past its own timeout and recover from a
// refusal, the subscription swap must keep publishing through it, and
// both must ask for what the server skipped and publish again after a
// refusal. Unspecified prescribes nothing, so Plan attaches the
// invariants only.
func (p4_05_14) Rules(c matrix.Category) []rules.Rule {
	switch c {
	case matrix.SteadyPublishing:
		return []rules.Rule{
			rules.RepublishesWithinTimeoutAfterPublishTimeout,
			rules.RepublishesSkippedSequence,
			rules.PublishesAgainAfterTooManyPublishRequests,
		}
	case matrix.CancelThenSubscribe:
		return []rules.Rule{
			rules.KeepsPublishingAfterCancelThenSubscribe,
			rules.RepublishesSkippedSequence,
			rules.PublishesAgainAfterTooManyPublishRequests,
		}
	}
	return nil
}

// scenario05_14 is one §5.14 scenario: the messages a correct client
// sends after the arm point, the category the clause prescribes for it,
// and the workload that drives the client.
type scenario05_14 struct {
	name  string
	sends []message.Message
	own   matrix.Category
	drive func(env *spectest.Environment, f faults.Fault, s scenario05_14) matrix.Outcome
}

func (s scenario05_14) Name() string             { return s.name }
func (s scenario05_14) Sends() []message.Message { return s.sends }

func (s scenario05_14) Options(f faults.Fault) []spectest.Option {
	return []spectest.Option{
		spectest.WithRetentionQueue(),
		spectest.WithPublishingInterval(10 * time.Millisecond),
		spectest.WithClientOptions(opcua.RequestTimeout(2 * time.Second)),
		spectest.WithFirstValue(s.values(f).first),
	}
}

// Category is the scenario's own, except for a DelayAboveTimeout or
// Overload fault on a service other than Publish, where Part 4
// prescribes no reaction the rules read: the invariants only.
func (s scenario05_14) Category(f faults.Fault) matrix.Category {
	if (strings.HasPrefix(f.Name(), "DelayAboveTimeout/") || strings.HasPrefix(f.Name(), "Overload/")) && !targetsPublish(f) {
		return matrix.Unspecified
	}
	return s.own
}

// caseValues05_14 are the values one §5.14 case answers, unique per
// case: the first one Start answers, the arm exchange's answer, the
// steady stream's values, the cycle values and the sentinel, every one
// derived from the case's own block.
type caseValues05_14 struct {
	first    int32
	vArm     int32
	steady   [11]int32
	cycles   [cancelCycles]int32
	sentinel int32
}

// values returns the value block of one §5.14 case, derived from the
// case's scenario and fault alone so the Start options and the workload
// read the same block.
func (s scenario05_14) values(f faults.Fault) caseValues05_14 {
	base := int32(1000 * caseOrdinal(s.name, f))
	values := caseValues05_14{first: base + 1, vArm: base + 11, sentinel: base + 999}
	for i := range values.steady {
		values.steady[i] = base + int32(21+i)
	}
	for i := range values.cycles {
		values.cycles[i] = base + int32(41+i)
	}
	return values
}

// driveSteadyPublishing arms the fault, answers a steady stream,
// holds one Publish past the client's publish timeout, answers the
// rest, and returns the rules' context. The fault is the interruption:
// no relay connection is cut.
func driveSteadyPublishing(env *spectest.Environment, f faults.Fault, s scenario05_14) matrix.Outcome {
	values := s.values(f)
	sub := env.Subscription()

	m := env.Mark()
	injected := f.Inject(env)
	// The arm exchange a Publish-targeting fault gets: a held Publish
	// answered right after the arm, so the fault has an exchange to fire
	// on that exists only because the workload armed first.
	if targetsPublish(f) {
		if held, ok := env.Server.TryWaitHeldPublish(15 * time.Second); ok {
			held.Answer(sub, values.vArm)
		}
	}
	for _, v := range values.steady[:5] {
		if held, ok := env.Server.TryWaitHeldPublish(15 * time.Second); ok {
			held.Answer(sub, v)
		}
	}
	// One Publish the workload holds unanswered past the client's
	// publish timeout: the stream times out, and the client must
	// publish again. The value answered on it is owed: the server
	// retains it, and the client must ask for it with a Republish.
	heldOrder := 0
	heldAnswerOrder := 0
	if held, ok := env.Server.TryWaitHeldPublish(15 * time.Second); ok {
		if order, recorded := held.Order(); recorded {
			heldOrder = order
		}
		time.Sleep(env.PublishTimeout() + 2*time.Second)
		held.Answer(sub, values.steady[5])
		if answerOrder, answered := held.AnswerOrder(); answered {
			heldAnswerOrder = answerOrder
		}
	}
	for _, v := range values.steady[6:] {
		if held, ok := env.Server.TryWaitHeldPublish(15 * time.Second); ok {
			held.Answer(sub, v)
		}
	}
	faultEnd := time.Now()
	sentinel := int32(0)
	var answeredAt time.Time
	if held, ok := env.Server.TryWaitHeldPublish(15 * time.Second); ok {
		held.Answer(sub, values.sentinel)
		sentinel = values.sentinel
		answeredAt = time.Now()
	}
	return matrix.Outcome{
		Injected:   injected,
		FaultEnd:   faultEnd,
		Sentinel:   sentinel,
		AnsweredAt: answeredAt,
		Rules: rules.Context{
			Env:             env,
			Mark:            m,
			Sub:             sub,
			Server:          env.Server,
			Value:           values.steady[5],
			HeldOrder:       heldOrder,
			HeldAnswerOrder: heldAnswerOrder,
		},
	}
}

// cancelCycles is how many cancel-then-subscribe cycles the workload
// drives: on main the client's publish loop picks the pause or the
// resume signal at random, so a cycle has half a chance to park the
// loop, and twenty cycles make a buggy client fail with probability
// 1 − 2⁻²⁰ inside one case.
const cancelCycles = 20

// driveCancelThenSubscribe arms the fault, then drives cancelCycles
// cycles of cancelling the only subscription and creating a new one
// while a Publish is held, answering one value per cycle and stopping
// at the first cycle whose value the client did not deliver. The fault
// is the interruption: no relay connection is cut.
func driveCancelThenSubscribe(env *spectest.Environment, f faults.Fault, s scenario05_14) matrix.Outcome {
	values := s.values(f)
	sub := env.Subscription()

	m := env.Mark()
	injected := f.Inject(env)
	if targetsPublish(f) {
		if held, ok := env.Server.TryWaitHeldPublish(15 * time.Second); ok {
			held.Answer(sub, values.vArm)
		}
	}
	completed := 0
	current := env.ClientSubscription()
cycle:
	for i := 0; i < cancelCycles; i++ {
		held, heldOK := env.Server.TryWaitHeldPublish(15 * time.Second)
		if !heldOK {
			break cycle
		}
		cancelCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cancelErr := current.Cancel(cancelCtx)
		cancel()
		if cancelErr != nil {
			break cycle
		}
		subscribeCtx, subscribe := context.WithTimeout(context.Background(), 5*time.Second)
		next, subscribeErr := env.Client.Subscribe(subscribeCtx, &opcua.SubscriptionParameters{
			Interval:          100 * time.Millisecond,
			LifetimeCount:     1_000_000,
			MaxKeepAliveCount: 1000,
		}, current.Notifs)
		subscribe()
		if subscribeErr != nil {
			break cycle
		}
		monitorCtx, monitor := context.WithTimeout(context.Background(), 5*time.Second)
		_, monitorErr := next.Monitor(monitorCtx, ua.TimestampsToReturnBoth,
			opcua.NewMonitoredItemCreateRequestWithDefaults(env.Server.Node(), ua.AttributeIDValue, 1))
		monitor()
		if monitorErr != nil {
			break cycle
		}
		created, createdOK := env.Server.TryWaitCreatedSubscription(m, 15*time.Second)
		if !createdOK {
			break cycle
		}
		held.Answer(created, values.cycles[i])
		deadline := time.Now().Add(5 * time.Second)
		delivered := false
		for time.Now().Before(deadline) {
			for _, received := range env.ReceivedSince(m) {
				if received == values.cycles[i] {
					delivered = true
					break
				}
			}
			if delivered {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if !delivered {
			break cycle
		}
		current = next
		completed++
	}
	faultEnd := time.Now()
	sentinel := int32(0)
	var answeredAt time.Time
	if held, ok := env.Server.TryWaitHeldPublish(15 * time.Second); ok {
		held.Answer(env.Subscription(), values.sentinel)
		sentinel = values.sentinel
		answeredAt = time.Now()
	}
	lastValue := int32(0)
	if completed > 0 {
		lastValue = values.cycles[completed-1]
	}
	return matrix.Outcome{
		Injected:   injected,
		FaultEnd:   faultEnd,
		Sentinel:   sentinel,
		AnsweredAt: answeredAt,
		Rules: rules.Context{
			Env:             env,
			Mark:            m,
			Sub:             sub,
			Server:          env.Server,
			Value:           lastValue,
			CyclesCompleted: completed,
			CyclesWanted:    cancelCycles,
		},
	}
}

// Run drives one §5.14 case: the scenario's workload, with the values
// of the case's own block.
func (s scenario05_14) Run(env *spectest.Environment, f faults.Fault) matrix.Outcome {
	return s.drive(env, f, s)
}

// caseValuesOf05_14 returns every value of one §5.14 case's block, for
// the test that pins the blocks apart.
func caseValuesOf05_14(scenario matrix.Scenario, f faults.Fault) []int32 {
	s := scenario.(scenario05_14)
	values := s.values(f)
	all := []int32{values.first, values.vArm, values.sentinel}
	all = append(all, values.steady[:]...)
	all = append(all, values.cycles[:]...)
	return all
}
