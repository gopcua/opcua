package part4

import (
	"context"
	"strings"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/matrix"
	"github.com/gopcua/opcua/tests/spec/internal/message"
	"github.com/gopcua/opcua/tests/spec/internal/rules"
	"github.com/gopcua/opcua/ua"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	timeoutPublishingInterval = 10 * time.Millisecond
	publishTimeoutWait        = 20 * time.Second
)

var _ = Describe("when the publishing interval is 10 ms", func() {
	It("times out a held Publish on its first subscription", Label("P4-5.14.1.2"), func() {
		env := harness.New(GinkgoT(), harness.WithPublishingInterval(timeoutPublishingInterval))
		held := env.Server.WaitHeldPublish()
		heldOrder, recorded := held.Order()
		Expect(recorded).To(BeTrue(), "the held Publish request was never recorded, so its wire order is unknown")
		// Hold the Publish unanswered past the client's publish
		// timeout: the request times out on the client, which must
		// publish again while the first stays unanswered.
		time.Sleep(env.PublishTimeout() + 2*time.Second)
		rules.RepublishesWithinTimeoutAfterPublishTimeout.Check(matrix.Context{Env: env, Server: env.Server, HeldOrder: heldOrder})
	})

	It("times out a held Publish after recreating the subscription", Label("P4-5.14.1.2", "known-defect"), func() {
		env := harness.New(GinkgoT(), harness.WithPublishingInterval(timeoutPublishingInterval))
		second := env.StartServer()
		env.Relay.RedirectTo(second.Address())
		second.QueueTransferRefusal(ua.StatusBadSubscriptionIDInvalid)
		m := env.Mark()
		env.Relay.Cut()
		env.WaitUntilReconnected()
		second.WaitCreatedSubscription(m)
		held := second.WaitHeldPublish()
		heldOrder, recorded := held.Order()
		Expect(recorded).To(BeTrue(), "the held Publish request was never recorded, so its wire order is unknown")
		time.Sleep(env.PublishTimeout() + 2*time.Second)
		rules.RepublishesWithinTimeoutAfterPublishTimeout.Check(matrix.Context{Env: env, Server: second, HeldOrder: heldOrder})
		Expect(second.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", second.UnusedScripts())
	})
})

var _ = Describe("when the only subscription is cancelled and a new one created while a Publish is held", func() {
	It("keeps publishing", Label("P4-5.14.1.2", "issue-895", "known-defect"), MustPassRepeatedly(20), func() {
		env := harness.New(GinkgoT())
		old := env.ClientSubscription()
		held := env.Server.WaitHeldPublish()
		m := env.Mark()
		cancelCtx, cancelCancel := context.WithTimeout(context.Background(), secondSubscribeWait)
		cancelErr := old.Cancel(cancelCtx)
		cancelCancel()
		Expect(cancelErr).NotTo(HaveOccurred(), "the client cancelled no subscription: %v", cancelErr)
		requireSubscriptionDeleted(env, m, old.SubscriptionID)
		subscribeCtx, subscribeCancel := context.WithTimeout(context.Background(), secondSubscribeWait)
		newSubscription, subscribeErr := env.Client.Subscribe(subscribeCtx, &opcua.SubscriptionParameters{
			Interval:          secondSubscribeInterval,
			LifetimeCount:     secondSubscribeLifetimeCount,
			MaxKeepAliveCount: secondSubscribeKeepAliveCount,
		}, old.Notifs)
		subscribeCancel()
		Expect(subscribeErr).NotTo(HaveOccurred(), "the client created no new subscription: %v", subscribeErr)
		created := env.Server.WaitCreatedSubscription(m)
		node := rules.MonitoredNode(env)
		Expect(node).NotTo(BeNil(), "the recorder saw no CreateMonitoredItems request, so the node the client monitors is unknown")
		monitorCtx, monitorCancel := context.WithTimeout(context.Background(), secondSubscribeWait)
		_, monitorErr := newSubscription.Monitor(monitorCtx, ua.TimestampsToReturnBoth,
			opcua.NewMonitoredItemCreateRequestWithDefaults(node, ua.AttributeIDValue, secondMonitorClientHandle))
		monitorCancel()
		Expect(monitorErr).NotTo(HaveOccurred(), "the client monitored no node on the new subscription: %v", monitorErr)
		held.Answer(created, valueAfterReconnect)
		rules.KeepsPublishingAfterCancelThenSubscribe.Check(matrix.Context{Env: env, Mark: m, Value: valueAfterReconnect})
		Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
	})
})

func requireSubscriptionDeleted(env *harness.Environment, m harness.Mark, id uint32) {
	responses := env.Recorder.ResponsesSince(m)
	deleted := false
	for _, record := range rules.RequestsOfType[*ua.DeleteSubscriptionsRequest](env.Recorder.RequestsSince(m)) {
		message, decoded := record.Message()
		if !decoded {
			continue
		}
		if request, is := message.(*ua.DeleteSubscriptionsRequest); is && len(request.SubscriptionIDs) == 1 && request.SubscriptionIDs[0] == id {
			if answer, answered := rules.AnswerTo(record, responses); answered {
				answerMessage, answerDecoded := answer.Message()
				if response, isDelete := answerMessage.(*ua.DeleteSubscriptionsResponse); answerDecoded && isDelete && len(response.Results) == 1 && response.Results[0] == ua.StatusOK {
					deleted = true
				}
			}
		}
	}
	Expect(deleted).To(BeTrue(),
		"the client deleted no subscription %d before creating the new one; requests since the mark: %v", id, rules.RequestTypeNames(env.Recorder.RequestsSince(m)))
}

// The §5.14 failure matrix: scenarios whose workloads drive a real
// client through the subscription publishing the clause prescribes — a
// steady stream the fault interrupts, and a subscription swap while a
// Publish is held. The fault is the interruption: the workload cuts no
// relay connection of its own.
var _ = Describe("P4-5.14", func() {
	DescribeTableSubtree(steadyPublishing.Name, func(f fault.Fault) {
		obs := matrix.Run(steadyPublishing, f)
		obs.BeforeCloseInvariants()
		if obs.Applies(rules.RepublishesWithinTimeoutAfterPublishTimeout) {
			It("RepublishesWithinTimeoutAfterPublishTimeout", obs.Labels("RepublishesWithinTimeoutAfterPublishTimeout"), func() {
				rules.RepublishesWithinTimeoutAfterPublishTimeout.Check(obs.Context())
			})
		}
		if obs.Applies(rules.RepublishesSkippedSequence) {
			It("RepublishesSkippedSequence", obs.Labels("RepublishesSkippedSequence"), func() {
				rules.RepublishesSkippedSequence.Check(obs.Context())
			})
		}
		if obs.Applies(rules.PublishesAgainAfterTooManyPublishRequests) {
			It("PublishesAgainAfterTooManyPublishRequests", obs.Labels("PublishesAgainAfterTooManyPublishRequests"), func() {
				rules.PublishesAgainAfterTooManyPublishRequests.Check(obs.Context())
			})
		}
		obs.AfterCloseInvariants()
	}, matrix.Entries(steadyPublishing, fault.AllFaults...))

	DescribeTableSubtree(cancelThenSubscribe.Name, func(f fault.Fault) {
		obs := matrix.Run(cancelThenSubscribe, f)
		obs.BeforeCloseInvariants()
		if obs.Applies(rules.KeepsPublishingAfterCancelThenSubscribe) {
			It("KeepsPublishingAfterCancelThenSubscribe", obs.Labels("KeepsPublishingAfterCancelThenSubscribe"), func() {
				rules.KeepsPublishingAfterCancelThenSubscribe.Check(obs.Context())
			})
		}
		if obs.Applies(rules.RepublishesSkippedSequence) {
			It("RepublishesSkippedSequence", obs.Labels("RepublishesSkippedSequence"), func() {
				rules.RepublishesSkippedSequence.Check(obs.Context())
			})
		}
		if obs.Applies(rules.PublishesAgainAfterTooManyPublishRequests) {
			It("PublishesAgainAfterTooManyPublishRequests", obs.Labels("PublishesAgainAfterTooManyPublishRequests"), func() {
				rules.PublishesAgainAfterTooManyPublishRequests.Check(obs.Context())
			})
		}
		obs.AfterCloseInvariants()
	}, matrix.Entries(cancelThenSubscribe, fault.AllFaults...))
})

var steadyPublishing = matrix.Scenario{
	Clause:  "P4-5.14",
	Name:    "SteadyPublishing",
	Ordinal: 4,
	Sends: []message.Message{
		message.Publish, message.CloseSession, message.CloseSecureChannel,
	},
	Options:  publishingOptions,
	Workload: driveSteadyPublishing,
	// The steady stream must publish again past its own timeout, ask
	// for what the server skipped and publish again after a refusal.
	Rules: func(f fault.Fault) []matrix.Rule {
		if publishingUnspecified(f) {
			return nil
		}
		return []matrix.Rule{
			rules.RepublishesWithinTimeoutAfterPublishTimeout,
			rules.RepublishesSkippedSequence,
			rules.PublishesAgainAfterTooManyPublishRequests,
		}
	},
	Invariants: matrix.SubscriptionInvariants,
}

var cancelThenSubscribe = matrix.Scenario{
	Clause:  "P4-5.14",
	Name:    "CancelThenSubscribe",
	Ordinal: 5,
	Sends: []message.Message{
		message.DeleteSubscriptions, message.CreateSubscription, message.CreateMonitoredItems,
		message.Publish, message.CloseSession, message.CloseSecureChannel,
	},
	Options:  publishingOptions,
	Workload: driveCancelThenSubscribe,
	// The subscription swap must keep publishing through it, ask for
	// what the server skipped and publish again after a refusal.
	Rules: func(f fault.Fault) []matrix.Rule {
		if publishingUnspecified(f) {
			return nil
		}
		return []matrix.Rule{
			rules.KeepsPublishingAfterCancelThenSubscribe,
			rules.RepublishesSkippedSequence,
			rules.PublishesAgainAfterTooManyPublishRequests,
		}
	},
	Invariants: matrix.SubscriptionInvariants,
	// The parked publish loop is the #895 mechanism, so every fault
	// fails the cycle count and the sentinel both.
	KnownDefects: []matrix.KnownDefect{
		{Issue: "issue-895", Check: "KeepsPublishingAfterCancelThenSubscribe", Applies: matrix.EveryFault},
		{Issue: "issue-895", Check: "ResumePublishing", Applies: matrix.EveryFault},
	},
}

// publishingUnspecified says Part 4 prescribes no reaction the rules
// read: a DelayAboveTimeout or Overload fault on a service other than
// Publish leaves the invariants only.
func publishingUnspecified(f fault.Fault) bool {
	return (strings.HasPrefix(f.Name(), "DelayAboveTimeout/") || strings.HasPrefix(f.Name(), "Overload/")) && !targetsPublish(f)
}

func publishingOptions(f fault.Fault, block int32) []harness.Option {
	return []harness.Option{
		harness.WithRetentionQueue(),
		harness.WithPublishingInterval(10 * time.Millisecond),
		harness.WithClientOptions(opcua.RequestTimeout(2 * time.Second)),
		harness.WithFirstValue(publishingValuesOf(block).first),
	}
}

// publishingValues are the values one §5.14 case answers, unique per
// case: the first one New answers, the arm exchange's answer, the
// steady stream's values, the cycle values and the sentinel, every one
// derived from the case's own block.
type publishingValues struct {
	first    int32
	vArm     int32
	steady   [11]int32
	cycles   [cancelCycles]int32
	sentinel int32
}

// publishingValuesOf returns the value block of one §5.14 case, derived
// from the case's block alone so the New options and the workload read
// the same values.
func publishingValuesOf(base int32) publishingValues {
	values := publishingValues{first: base + 1, vArm: base + 11, sentinel: base + 999}
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
func driveSteadyPublishing(env *harness.Environment, f fault.Fault, block int32) matrix.Outcome {
	values := publishingValuesOf(block)
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
		Rules: matrix.Context{
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
func driveCancelThenSubscribe(env *harness.Environment, f fault.Fault, block int32) matrix.Outcome {
	values := publishingValuesOf(block)
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
		Rules: matrix.Context{
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
