package part4

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/matrix"
	"github.com/onsi/gomega/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const specWait = 15 * time.Second

func produced(value int32, subscription uint32, sequence uint32, server int, reachable bool) matrix.Produced {
	return matrix.Produced{Value: value, SubscriptionID: subscription, SequenceNumber: sequence, ServerIndex: server, Reachable: reachable, SubscriptionInstance: int(subscription)}
}

func incarnation(value int32, subscription uint32, instance int, sequence uint32, server int, reachable bool) matrix.Produced {
	return matrix.Produced{Value: value, SubscriptionID: subscription, SequenceNumber: sequence, ServerIndex: server, Reachable: reachable, SubscriptionInstance: instance}
}

func repeated(value int32, subscription uint32, sequence uint32, server int, reachable bool) matrix.Produced {
	return matrix.Produced{Value: value, SubscriptionID: subscription, SequenceNumber: sequence, ServerIndex: server, Reachable: reachable, Repeated: true}
}

func forgotten(value int32, subscription uint32, sequence uint32, server int) matrix.Produced {
	return matrix.Produced{Value: value, SubscriptionID: subscription, SequenceNumber: sequence, ServerIndex: server, Reachable: true, Forgotten: true}
}

func server(index int, reachable, connected bool, sessions, subscriptions int) matrix.ServerState {
	return matrix.ServerState{Index: index, Reachable: reachable, Connected: connected, KnownSessions: sessions, LiveSubscriptions: subscriptions}
}

func assertMatcher(t *testing.T, matcher types.GomegaMatcher, observed matrix.Observed, wantPass bool, wantSubstring string) {
	t.Helper()
	success, err := matcher.Match(observed)
	if err != nil {
		t.Fatalf("%T.Match returned an error: %v", matcher, err)
	}
	if success != wantPass {
		t.Fatalf("%T matched %v on %v, want %v", matcher, success, observed, wantPass)
	}
	if wantPass {
		return
	}
	message := matcher.FailureMessage(observed)
	if !strings.Contains(message, wantSubstring) {
		t.Fatalf("%T failed with %q, want it to name %q", matcher, message, wantSubstring)
	}
}

func TestDeliverEachValueOnce(t *testing.T) {
	cases := []struct {
		name       string
		observed   matrix.Observed
		wantPass   bool
		wantSubstr string
	}{
		{"every produced value received once",
			matrix.Observed{
				Produced: []matrix.Produced{produced(101, 1, 1, 0, true), produced(102, 1, 2, 0, true)},
				Received: []int32{101, 102},
			}, true, ""},
		{"one produced value never received",
			matrix.Observed{
				Produced: []matrix.Produced{produced(101, 1, 1, 0, true), produced(102, 1, 2, 0, true)},
				Received: []int32{101},
			}, false, "102"},
		{"one value received twice",
			matrix.Observed{
				Produced: []matrix.Produced{produced(101, 1, 1, 0, true)},
				Received: []int32{101, 101},
			}, false, "101"},
		{"a value received that no server produced",
			matrix.Observed{
				Produced: []matrix.Produced{produced(101, 1, 1, 0, true)},
				Received: []int32{101, 107},
			}, false, "107"},
		{"a missing value whose only producer is unreachable",
			matrix.Observed{
				Produced: []matrix.Produced{produced(101, 1, 1, 0, true), produced(150, 1, 2, 1, false)},
				Received: []int32{101},
			}, true, ""},
		{"the same value produced twice and received once",
			matrix.Observed{
				Produced: []matrix.Produced{produced(101, 1, 3, 0, true), produced(101, 1, 3, 0, true)},
				Received: []int32{101},
			}, true, ""},
		{"a value under an already-sent sequence number is received",
			matrix.Observed{
				Produced: []matrix.Produced{produced(101, 1, 1, 0, true), repeated(107, 1, 1, 0, true)},
				Received: []int32{101, 107},
			}, false, "107, produced at sequence number 1"},
		{"a value under an already-sent sequence number is not received, the rest once",
			matrix.Observed{
				Produced: []matrix.Produced{produced(101, 1, 1, 0, true), repeated(107, 1, 1, 0, true), produced(108, 1, 2, 0, true)},
				Received: []int32{101, 108},
			}, true, ""},
		{"a value retained on a subscription the server forgot is not received",
			matrix.Observed{
				Produced: []matrix.Produced{produced(101, 1, 1, 0, true), forgotten(107, 1, 3, 0), produced(108, 1, 2, 0, true)},
				Received: []int32{101, 108},
			}, true, ""},
		{"a value on a live subscription is still owed",
			matrix.Observed{
				Produced: []matrix.Produced{produced(101, 1, 1, 0, true), produced(107, 1, 3, 0, true), produced(108, 1, 2, 0, true)},
				Received: []int32{101, 108},
			}, false, "107"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertMatcher(t, deliverEachValueOnce(), c.observed, c.wantPass, c.wantSubstr)
		})
	}
}

func TestDeliverInOrder(t *testing.T) {
	cases := []struct {
		name       string
		observed   matrix.Observed
		wantPass   bool
		wantSubstr string
	}{
		{"sequence three received before two",
			matrix.Observed{
				Produced: []matrix.Produced{produced(102, 1, 2, 0, true), produced(103, 1, 3, 0, true)},
				Received: []int32{103, 102},
			}, false, "103"},
		{"two subscriptions interleaved, each in order",
			matrix.Observed{
				Produced: []matrix.Produced{produced(201, 2, 1, 0, true), produced(101, 1, 1, 0, true), produced(202, 2, 2, 0, true), produced(102, 1, 2, 0, true)},
				Received: []int32{201, 101, 202, 102},
			}, true, ""},
		{"a recreated subscription reuses the id and restarts at one",
			matrix.Observed{
				Produced: []matrix.Produced{
					incarnation(101, 1, 1, 1, 0, true),
					incarnation(102, 1, 1, 2, 0, true),
					incarnation(103, 1, 2, 1, 0, true),
					incarnation(104, 1, 2, 2, 0, true),
				},
				Received: []int32{101, 102, 103, 104},
			}, true, ""},
		{"the same subscription on two servers, each in order",
			matrix.Observed{
				Produced: []matrix.Produced{
					incarnation(101, 1, 1, 2, 0, true),
					incarnation(102, 1, 1, 3, 0, true),
					incarnation(201, 1, 1, 1, 1, true),
					incarnation(202, 1, 1, 2, 1, true),
				},
				Received: []int32{101, 102, 201, 202},
			}, true, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertMatcher(t, deliverInOrder(), c.observed, c.wantPass, c.wantSubstr)
		})
	}
}

func TestResumePublishing(t *testing.T) {
	base := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	sentinelAt := func(delay time.Duration) *matrix.Sentinel {
		return &matrix.Sentinel{Value: 199, AnsweredAt: base, ReceivedAt: base.Add(delay)}
	}
	cases := []struct {
		name       string
		observed   matrix.Observed
		within     time.Duration
		wantPass   bool
		wantSubstr string
	}{
		{"no sentinel staged",
			matrix.Observed{FaultEnd: base, Sentinel: nil}, 15 * time.Second, false, "sentinel"},
		{"sentinel received sixteen seconds after the fault ended",
			matrix.Observed{FaultEnd: base, Sentinel: sentinelAt(16 * time.Second)}, 15 * time.Second, false, "16"},
		{"sentinel received sixteen seconds after the fault ended names the window",
			matrix.Observed{FaultEnd: base, Sentinel: sentinelAt(16 * time.Second)}, 15 * time.Second, false, "15s"},
		{"sentinel received two seconds after the fault ended",
			matrix.Observed{FaultEnd: base, Sentinel: sentinelAt(2 * time.Second)}, 15 * time.Second, true, ""},
		{"sentinel never received",
			matrix.Observed{FaultEnd: base, Sentinel: &matrix.Sentinel{Value: 199, AnsweredAt: base}}, 15 * time.Second, false, "received"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertMatcher(t, resumePublishing(c.within), c.observed, c.wantPass, c.wantSubstr)
		})
	}
}

func TestKeepOneSessionOpen(t *testing.T) {
	cases := []struct {
		name       string
		observed   matrix.Observed
		wantPass   bool
		wantSubstr string
	}{
		{"connected server holds two known sessions",
			matrix.Observed{Servers: []matrix.ServerState{server(0, true, true, 2, 1)}}, false, "2"},
		{"connected server holds one known session",
			matrix.Observed{Servers: []matrix.ServerState{server(0, true, true, 1, 1)}}, true, ""},
		{"an unreachable server's session does not count",
			matrix.Observed{Servers: []matrix.ServerState{server(1, false, false, 1, 0), server(0, true, true, 1, 1)}}, true, ""},
		{"no connected server",
			matrix.Observed{Servers: []matrix.ServerState{server(0, true, false, 1, 1)}}, false, "connected"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertMatcher(t, keepOneSessionOpen(), c.observed, c.wantPass, c.wantSubstr)
		})
	}
}

func TestCloseEveryKnownSession(t *testing.T) {
	cases := []struct {
		name       string
		observed   matrix.Observed
		wantPass   bool
		wantSubstr string
	}{
		{"a reachable server still holds a known session",
			matrix.Observed{Servers: []matrix.ServerState{server(0, true, false, 1, 0)}}, false, "1"},
		{"only an unreachable server still holds a known session",
			matrix.Observed{Servers: []matrix.ServerState{server(1, false, false, 1, 0), server(0, true, false, 0, 0)}}, true, ""},
		{"a session whose CloseSession the client sent but the network lost",
			matrix.Observed{Servers: []matrix.ServerState{{Index: 0, Reachable: true, KnownSessions: 1, ClosingAttempted: 1}}}, true, ""},
		{"one of two sessions closing-attempted still fails for the other",
			matrix.Observed{Servers: []matrix.ServerState{{Index: 0, Reachable: true, KnownSessions: 2, ClosingAttempted: 1}}}, false, "1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertMatcher(t, closeEveryKnownSession(), c.observed, c.wantPass, c.wantSubstr)
		})
	}
}

func TestKeepOneSubscriptionPerClientSubscription(t *testing.T) {
	cases := []struct {
		name       string
		observed   matrix.Observed
		wantPass   bool
		wantSubstr string
	}{
		{"the server holds two live subscriptions for one client subscription",
			matrix.Observed{Servers: []matrix.ServerState{server(0, true, true, 1, 2)}, ClientSubscriptions: 1}, false, "2"},
		{"live subscriptions equal the client's",
			matrix.Observed{Servers: []matrix.ServerState{server(0, true, true, 1, 1)}, ClientSubscriptions: 1}, true, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertMatcher(t, keepOneSubscriptionPerClientSubscription(), c.observed, c.wantPass, c.wantSubstr)
		})
	}
}

func closeClient(env *harness.Environment) {
	ctx, cancel := context.WithTimeout(context.Background(), specWait)
	defer cancel()
	Expect(env.Client.Close(ctx)).To(Succeed(), "closing the client failed")
}

var _ = Describe("Observe", func() {
	It("matches every invariant on an undisturbed environment", func() {
		env := harness.New(GinkgoT(), harness.WithRetentionQueue())
		sub := env.Subscription()
		caseStart := time.Now()
		env.Server.WaitHeldPublish().Answer(sub, 101)
		env.Server.WaitHeldPublish().Answer(sub, 102)
		env.Server.WaitHeldPublish().Answer(sub, 103)
		sentinelAnswered := time.Now()
		env.Server.WaitHeldPublish().Answer(sub, 199)
		Eventually(func() []int32 { return env.Received() }, specWait).Should(HaveLen(5),
			"the client never received the sentinel along with the answered values")

		observed := matrix.Observe(env, nil)
		observed.WithFaultEnd(caseStart)
		observed.WithSentinel(199, sentinelAnswered)

		Expect(observed).To(deliverEachValueOnce())
		Expect(observed).To(deliverInOrder())
		Expect(observed).To(resumePublishing(15 * time.Second))
		Expect(observed).To(keepOneSessionOpen())
		Expect(observed).To(keepOneSubscriptionPerClientSubscription())

		closeClient(env)
		Expect(matrix.Observe(env, nil)).To(closeEveryKnownSession())
	})

	It("names a planted loss", func() {
		env := harness.New(GinkgoT(), harness.WithRetentionQueue())
		sub := env.Subscription()
		caseStart := time.Now()
		env.Server.WaitHeldPublish().Answer(sub, 101)
		env.Server.WaitHeldPublish().Answer(sub, 102)
		env.Server.WaitHeldPublish().Answer(sub, 103)
		sentinelAnswered := time.Now()
		env.Server.WaitHeldPublish().Answer(sub, 199)
		Eventually(func() []int32 { return env.Received() }, specWait).Should(HaveLen(5),
			"the client never received the sentinel along with the answered values")
		sub.Retain(9099, 150)

		observed := matrix.Observe(env, nil)
		observed.WithFaultEnd(caseStart)
		observed.WithSentinel(199, sentinelAnswered)

		Expect(observed).To(deliverInOrder())
		Expect(observed).To(resumePublishing(15 * time.Second))
		Expect(observed).To(keepOneSessionOpen())
		Expect(observed).To(keepOneSubscriptionPerClientSubscription())
		matcher := deliverEachValueOnce()
		success, err := matcher.Match(observed)
		Expect(err).NotTo(HaveOccurred(), "matching the snapshot failed")
		Expect(success).To(BeFalse(), "the planted loss 150 passed DeliverEachValueOnce")
		Expect(matcher.FailureMessage(observed)).To(ContainSubstring("150"),
			"the failure does not name the planted loss")
	})
})
