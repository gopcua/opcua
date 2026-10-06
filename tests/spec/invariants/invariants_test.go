package invariants

import (
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega/types"
)

func produced(value int32, subscription uint32, sequence uint32, server int, reachable bool) Produced {
	return Produced{Value: value, SubscriptionID: subscription, SequenceNumber: sequence, ServerIndex: server, Reachable: reachable}
}

func server(index int, reachable, connected bool, sessions, subscriptions int) ServerState {
	return ServerState{Index: index, Reachable: reachable, Connected: connected, KnownSessions: sessions, LiveSubscriptions: subscriptions}
}

func assertMatcher(t *testing.T, matcher types.GomegaMatcher, observed Observed, wantPass bool, wantSubstring string) {
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
		observed   Observed
		wantPass   bool
		wantSubstr string
	}{
		{"every produced value received once",
			Observed{
				Produced: []Produced{produced(101, 1, 1, 0, true), produced(102, 1, 2, 0, true)},
				Received: []int32{101, 102},
			}, true, ""},
		{"one produced value never received",
			Observed{
				Produced: []Produced{produced(101, 1, 1, 0, true), produced(102, 1, 2, 0, true)},
				Received: []int32{101},
			}, false, "102"},
		{"one value received twice",
			Observed{
				Produced: []Produced{produced(101, 1, 1, 0, true)},
				Received: []int32{101, 101},
			}, false, "101"},
		{"a value received that no server produced",
			Observed{
				Produced: []Produced{produced(101, 1, 1, 0, true)},
				Received: []int32{101, 107},
			}, false, "107"},
		{"a missing value whose only producer is unreachable",
			Observed{
				Produced: []Produced{produced(101, 1, 1, 0, true), produced(150, 1, 2, 1, false)},
				Received: []int32{101},
			}, true, ""},
		{"the same value produced twice and received once",
			Observed{
				Produced: []Produced{produced(101, 1, 3, 0, true), produced(101, 1, 3, 0, true)},
				Received: []int32{101},
			}, true, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertMatcher(t, DeliverEachValueOnce(), c.observed, c.wantPass, c.wantSubstr)
		})
	}
}

func TestDeliverInOrder(t *testing.T) {
	cases := []struct {
		name       string
		observed   Observed
		wantPass   bool
		wantSubstr string
	}{
		{"sequence three received before two",
			Observed{
				Produced: []Produced{produced(102, 1, 2, 0, true), produced(103, 1, 3, 0, true)},
				Received: []int32{103, 102},
			}, false, "103"},
		{"two subscriptions interleaved, each in order",
			Observed{
				Produced: []Produced{produced(201, 2, 1, 0, true), produced(101, 1, 1, 0, true), produced(202, 2, 2, 0, true), produced(102, 1, 2, 0, true)},
				Received: []int32{201, 101, 202, 102},
			}, true, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertMatcher(t, DeliverInOrder(), c.observed, c.wantPass, c.wantSubstr)
		})
	}
}

func TestResumePublishing(t *testing.T) {
	base := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	sentinelAt := func(delay time.Duration) *Sentinel {
		return &Sentinel{Value: 199, AnsweredAt: base, ReceivedAt: base.Add(delay)}
	}
	cases := []struct {
		name       string
		observed   Observed
		within     time.Duration
		wantPass   bool
		wantSubstr string
	}{
		{"no sentinel staged",
			Observed{FaultEnd: base, Sentinel: nil}, 15 * time.Second, false, "sentinel"},
		{"sentinel received sixteen seconds after the fault ended",
			Observed{FaultEnd: base, Sentinel: sentinelAt(16 * time.Second)}, 15 * time.Second, false, "16"},
		{"sentinel received sixteen seconds after the fault ended names the window",
			Observed{FaultEnd: base, Sentinel: sentinelAt(16 * time.Second)}, 15 * time.Second, false, "15s"},
		{"sentinel received two seconds after the fault ended",
			Observed{FaultEnd: base, Sentinel: sentinelAt(2 * time.Second)}, 15 * time.Second, true, ""},
		{"sentinel never received",
			Observed{FaultEnd: base, Sentinel: &Sentinel{Value: 199, AnsweredAt: base}}, 15 * time.Second, false, "received"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertMatcher(t, ResumePublishing(c.within), c.observed, c.wantPass, c.wantSubstr)
		})
	}
}

func TestKeepOneSessionOpen(t *testing.T) {
	cases := []struct {
		name       string
		observed   Observed
		wantPass   bool
		wantSubstr string
	}{
		{"connected server holds two known sessions",
			Observed{Servers: []ServerState{server(0, true, true, 2, 1)}}, false, "2"},
		{"connected server holds one known session",
			Observed{Servers: []ServerState{server(0, true, true, 1, 1)}}, true, ""},
		{"an unreachable server's session does not count",
			Observed{Servers: []ServerState{server(1, false, false, 1, 0), server(0, true, true, 1, 1)}}, true, ""},
		{"no connected server",
			Observed{Servers: []ServerState{server(0, true, false, 1, 1)}}, false, "connected"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertMatcher(t, KeepOneSessionOpen(), c.observed, c.wantPass, c.wantSubstr)
		})
	}
}

func TestCloseEveryKnownSession(t *testing.T) {
	cases := []struct {
		name       string
		observed   Observed
		wantPass   bool
		wantSubstr string
	}{
		{"a reachable server still holds a known session",
			Observed{Servers: []ServerState{server(0, true, false, 1, 0)}}, false, "1"},
		{"only an unreachable server still holds a known session",
			Observed{Servers: []ServerState{server(1, false, false, 1, 0), server(0, true, false, 0, 0)}}, true, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertMatcher(t, CloseEveryKnownSession(), c.observed, c.wantPass, c.wantSubstr)
		})
	}
}

func TestKeepOneSubscriptionPerClientSubscription(t *testing.T) {
	cases := []struct {
		name       string
		observed   Observed
		wantPass   bool
		wantSubstr string
	}{
		{"the server holds two live subscriptions for one client subscription",
			Observed{Servers: []ServerState{server(0, true, true, 1, 2)}, ClientSubscriptions: 1}, false, "2"},
		{"live subscriptions equal the client's",
			Observed{Servers: []ServerState{server(0, true, true, 1, 1)}, ClientSubscriptions: 1}, true, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertMatcher(t, KeepOneSubscriptionPerClientSubscription(), c.observed, c.wantPass, c.wantSubstr)
		})
	}
}

func TestHaveFired(t *testing.T) {
	assertMatcher(t, HaveFired(), Observed{Fired: true}, true, "")
	assertMatcher(t, HaveFired(), Observed{Fired: false}, false, "fired")
}
