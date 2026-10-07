// Package suites holds the failure matrix's scenario suites, one per
// Part 4 clause: each suite's scenarios drive a real client through
// one workload per fault and return the context the clause's rules
// read.
package suites

import (
	"slices"
	"strings"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/matrix"
	"github.com/gopcua/opcua/tests/spec/internal/message"
	"github.com/gopcua/opcua/tests/spec/internal/rules"
	"github.com/gopcua/opcua/ua"
)

// P4_06_07 returns the Part 4 §6.7 suite: three scenarios whose prepare
// step and transport loss decide what the client must re-establish
// after the relay cuts — the session, the session on a second server
// that refuses the transfer, or the subscriptions on a server that
// forgot them.
func P4_06_07() matrix.Suite { return p4_06_07{} }

type p4_06_07 struct{}

func (p4_06_07) Clause() string { return "P4-6.7" }

func (p4_06_07) Scenarios() []matrix.Scenario {
	return []matrix.Scenario{
		scenario06_07{
			name: "SessionSurvives",
			sends: []message.Message{
				message.HEL, message.OpenSecureChannel, message.ActivateSession, message.Read,
				message.Republish, message.Publish, message.CloseSession, message.CloseSecureChannel,
			},
			own:     matrix.SessionSurvives,
			prepare: prepareSessionSurvives,
		},
		scenario06_07{
			name: "SessionLost",
			sends: []message.Message{
				message.HEL, message.OpenSecureChannel, message.ActivateSession, message.CreateSession, message.Read,
				message.TransferSubscriptions, message.CreateSubscription, message.CreateMonitoredItems,
				message.Publish, message.CloseSession, message.CloseSecureChannel,
			},
			own:              matrix.SessionLost,
			prepare:          prepareSessionLost,
			prepareBeforeArm: true,
		},
		scenario06_07{
			name: "SubscriptionsLost",
			sends: []message.Message{
				message.HEL, message.OpenSecureChannel, message.ActivateSession, message.Read,
				message.Republish, message.CreateSubscription, message.CreateMonitoredItems,
				message.Publish, message.CloseSession, message.CloseSecureChannel,
			},
			own:     matrix.SubscriptionsLost,
			prepare: prepareSubscriptionsLost,
		},
	}
}

func (p4_06_07) Rules(c matrix.Category) []rules.Rule {
	switch c {
	case matrix.SessionSurvives:
		return []rules.Rule{
			rules.ReactivatesSession,
			rules.CreatesNoSession,
			rules.RepublishesFromNextSequence,
			rules.SendsNoPublishBeforeNotAvailable,
			rules.KeepsSubscriptionID,
			rules.SendsNoTransferForOwnSubscription,
		}
	case matrix.SessionLost:
		return []rules.Rule{
			rules.CreatesSessionOnlyAfterActivateFailed,
			rules.RecreatesAfterRefusal,
		}
	case matrix.SubscriptionsLost:
		return []rules.Rule{
			rules.RecreatesAfterRefusal,
			rules.RepublishesRecreatedFromOne,
		}
	case matrix.ActivationFailed:
		return []rules.Rule{rules.CreatesSessionAfterActivateTimedOut}
	}
	return nil
}

// scenario06_07 is one §6.7 scenario: the messages a correct client
// sends after the arm point, the category the clause prescribes when
// the fault leaves it to the scenario, and the prepare step that
// stages the server side of the transport loss and names the target
// the client must recover on. prepareBeforeArm says the prepare step
// must run before the arm point — the redirect the server-side faults
// arm behind — instead of after the consumer's burst.
type scenario06_07 struct {
	name             string
	sends            []message.Message
	own              matrix.Category
	prepare          func(env *harness.Environment) target06_07
	prepareBeforeArm bool
}

// target06_07 is what a scenario's prepare step leaves the workload
// with: the server the client must end on, whether it must recreate
// its subscription there, and how the scripted transfer answer
// refuses when the prepare queued one.
type target06_07 struct {
	server          *harness.ScriptedServer
	recreate        bool
	transferRefusal func(harness.ServiceRecord[ua.Response]) bool
}

func (s scenario06_07) Name() string { return s.name }

func (s scenario06_07) Sends() []message.Message { return s.sends }

func (s scenario06_07) Options(f fault.Fault) []harness.Option {
	return []harness.Option{
		harness.WithRetentionQueue(),
		harness.WithPublishingInterval(10 * time.Millisecond),
		harness.WithClientOptions(opcua.RequestTimeout(2 * time.Second)),
		harness.WithFirstValue(s.values(f).first),
	}
}

func (s scenario06_07) Category(f fault.Fault) matrix.Category {
	switch {
	case f.Name() == "DelayAboveTimeout/ActivateSession":
		return matrix.ActivationFailed
	case strings.HasPrefix(f.Name(), "DelayAboveTimeout/"):
		return matrix.Unspecified
	case strings.HasPrefix(f.Name(), "Overload/"):
		return matrix.Unspecified
	}
	return s.own
}

// Run drives one §6.7 case. The workload answers v3 and the sentinel
// on the subscription the client publishes with: the one the recreate
// scenarios make it create, or — when the scenario keeps the session —
// whatever live subscription the client holds, because a client that
// recreates its subscription there violates KeepsSubscriptionID and
// the case must still reach its checks. The prepare step runs before
// the arm point when the scenario's faults must arm behind its
// redirect, and after the consumer's burst otherwise — SubscriptionsLost
// forgets the old subscription only after the burst answered on it.
func (s scenario06_07) Run(env *harness.Environment, f fault.Fault) matrix.Outcome {
	values := s.values(f)
	sub := env.Subscription()
	if held, ok := env.Server.TryWaitHeldPublish(15 * time.Second); ok {
		held.Answer(sub, values.v1)
		waitReceived(env, values.v1)
	}
	last := env.LastSequenceNumber()
	sub.Retain(last+1, values.v2)
	var t target06_07
	if s.prepareBeforeArm {
		t = s.prepare(env)
	}
	m := env.Mark()
	injected := f.Inject(env)
	// The arm exchange: a Publish-targeting fault gets a held Publish
	// answered right after the arm, so its armed cut has a response to
	// fire on that exists only because the workload armed first — not
	// the answer of an exchange that raced the arm.
	if targetsPublish(f) {
		if held, ok := env.Server.TryWaitHeldPublish(15 * time.Second); ok {
			held.Answer(sub, values.vArm)
		}
	}
	for i := range consumerBurst(f) {
		if held, ok := env.Server.TryWaitHeldPublish(15 * time.Second); ok {
			held.Answer(sub, values.burst[i])
		}
	}
	if !s.prepareBeforeArm {
		t = s.prepare(env)
	}
	if breaksTransport(f) {
		env.Relay.Cut()
	}
	env.TryWaitUntilReconnected(30 * time.Second)
	faultEnd := time.Now()

	answering := sub
	recreated := harness.Subscription{}
	canAnswer := true
	if t.recreate {
		created, ok := t.server.TryWaitCreatedSubscription(m, 15*time.Second)
		if ok {
			answering = created
			recreated = created
		} else {
			canAnswer = false
		}
	} else if created, ok := t.server.SubscriptionCreatedSince(m); ok {
		answering = created
	}
	sentinel := int32(0)
	var answeredAt time.Time
	if canAnswer {
		if held, ok := t.server.TryWaitHeldPublish(15 * time.Second); ok {
			held.Answer(answering, values.v3)
			if next, ok := t.server.TryWaitHeldPublish(15 * time.Second); ok {
				next.Answer(answering, values.sentinel)
				sentinel = values.sentinel
				answeredAt = time.Now()
			}
		}
	}
	return matrix.Outcome{
		Injected:   injected,
		FaultEnd:   faultEnd,
		Sentinel:   sentinel,
		AnsweredAt: answeredAt,
		Rules: rules.Context{
			Env:             env,
			Mark:            m,
			LastSeq:         last,
			Sub:             sub,
			Recreated:       recreated,
			Server:          t.server,
			TransferRefusal: t.transferRefusal,
		},
	}
}

// prepareSessionSurvives leaves the session and its subscription on
// the one server the client is connected to; the cut after the arm
// point is the transport loss.
func prepareSessionSurvives(env *harness.Environment) target06_07 {
	return target06_07{server: env.Server}
}

// prepareSessionLost starts a second server that inherits the
// retention queue, refuses the next transfer with
// Bad_SubscriptionIdInvalid and sends the client's reconnects to it,
// all before the arm point so server-side faults arm there; the cut
// after the arm point is the transport loss, and the client must
// recreate its session and its subscription on the second server.
func prepareSessionLost(env *harness.Environment) target06_07 {
	second := env.StartServer()
	second.QueueTransferRefusal(ua.StatusBadSubscriptionIDInvalid)
	env.Relay.RedirectTo(second.Address())
	return target06_07{server: second, recreate: true, transferRefusal: refusedBadSubscriptionIDInvalid}
}

// prepareSubscriptionsLost marks every subscription on the server
// deleted: the session survives, but the server answers no
// subscription the client holds; the cut after the arm point is the
// transport loss.
func prepareSubscriptionsLost(env *harness.Environment) target06_07 {
	env.Server.ForgetSubscriptions()
	return target06_07{server: env.Server, recreate: true}
}

// breaksTransport says whether the workload cuts the relay after the
// fault is armed: a stalled link is its own transport loss — the link
// goes silent instead of closing — and so is a Publish fault that cuts
// the connection itself, because the arm exchange answers a held
// Publish whose request or response fires the armed cut.
func breaksTransport(f fault.Fault) bool {
	if f.Name() == "Link/Stall" {
		return false
	}
	if targetsPublish(f) {
		switch strings.Split(f.Name(), "/")[0] {
		case "RequestLost", "ResponseLost", "CutAfterResponse":
			return false
		}
	}
	return true
}

// targetsPublish says whether the fault arms on the Publish service:
// the message faults that name it and the overload faults that answer
// it.
func targetsPublish(f fault.Fault) bool {
	parts := strings.Split(f.Name(), "/")
	return len(parts) >= 2 && parts[1] == "Publish"
}

// waitReceived waits until the client has delivered v: a fault armed
// while the relay still holds the answer the client has not seen can
// fire on that answer's own response, and whether it does is a race
// between the relay's pump and the arm.
func waitReceived(env *harness.Environment, v int32) {
	deadline := time.Now().Add(15 * time.Second)
	for !slices.Contains(env.Received(), v) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
}

// consumerBurst says how many values the workload answers in a row
// right after the arm point: the slow consumer's channel, four deep,
// fills only when a burst outruns its 200 ms drain.
func consumerBurst(f fault.Fault) int {
	if f.Name() == "Consumer/Slow" {
		return 8
	}
	return 0
}

// refusedBadSubscriptionIDInvalid recognizes the transfer answer the
// SessionLost base action scripts: one result, Bad_SubscriptionIdInvalid.
func refusedBadSubscriptionIDInvalid(answer harness.ServiceRecord[ua.Response]) bool {
	message, decoded := answer.Message()
	if !decoded {
		return false
	}
	response, isTransfer := message.(*ua.TransferSubscriptionsResponse)
	return isTransfer && len(response.Results) == 1 && response.Results[0].StatusCode == ua.StatusBadSubscriptionIDInvalid
}

// caseValues are the values one case answers, unique per case: every
// value, the first one Start answers included, derives from the case's
// own block — 1000 times the case's ordinal among every scenario ×
// fault pair plus a per-value offset — so a received value matches the
// notification that carried it by value and never a value another case
// answered. The burst is the eight values the slow consumer's case
// answers right after arming. vArm is answered on the held Publish
// right after the arm point, so a Publish-targeting fault has an
// exchange to fire on that exists only because the workload armed.
type caseValues struct {
	first    int32
	v1       int32
	v2       int32
	v3       int32
	vArm     int32
	sentinel int32
	burst    [8]int32
}

// values returns the value block of one case, derived from the case's
// scenario and fault alone so the Start options and the workload read
// the same block.
func (s scenario06_07) values(f fault.Fault) caseValues {
	base := int32(1000 * caseOrdinal(s.name, f))
	values := caseValues{first: base + 1, v1: base + 2, v2: base + 3, v3: base + 4, vArm: base + 5, sentinel: base + 999}
	for i := range values.burst {
		values.burst[i] = base + int32(11+i)
	}
	return values
}

// ArmValueOf returns the value the workload answers on the held
// Publish right after the fault is armed, so a spec can tell the arm
// exchange's answer apart from the values the client received before
// the arm.
func ArmValueOf(scenario matrix.Scenario, f fault.Fault) int32 {
	return scenario.(scenario06_07).values(f).vArm
}

// caseOrdinal returns the case's ordinal among every scenario × fault
// pair: the scenario's place among the suite's scenarios times the
// fault catalogue's size, plus the fault's place in the catalogue.
func caseOrdinal(scenario string, f fault.Fault) int {
	faultOrdinal := 0
	for i, candidate := range fault.AllFaults {
		if candidate.Name() == f.Name() {
			faultOrdinal = i
			break
		}
	}
	return scenarioOrdinals[scenario]*len(fault.AllFaults) + faultOrdinal + 1
}

var scenarioOrdinals = map[string]int{
	"SessionSurvives":     0,
	"SessionLost":         1,
	"SubscriptionsLost":   2,
	"SteadyPublishing":    3,
	"CancelThenSubscribe": 4,
}

// caseValuesOf returns every value of one case's block, for the test
// that pins the blocks apart.
func caseValuesOf(scenario matrix.Scenario, f fault.Fault) []int32 {
	s := scenario.(scenario06_07)
	values := s.values(f)
	all := []int32{values.first, values.v1, values.v2, values.v3, values.vArm, values.sentinel}
	return append(all, values.burst[:]...)
}

// caseValuesOfAny returns every value of one case's block whatever
// suite its scenario belongs to, for the test that pins every suite's
// blocks apart.
func caseValuesOfAny(scenario matrix.Scenario, f fault.Fault) []int32 {
	if _, is := scenario.(scenario06_07); is {
		return caseValuesOf(scenario, f)
	}
	return caseValuesOf05_14(scenario, f)
}
