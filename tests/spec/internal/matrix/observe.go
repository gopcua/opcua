package matrix

import (
	"strings"
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/harness"
)

// Produced is one value a server produced, with the subscription and
// sequence number it was enqueued under and the server that holds it.
type Produced struct {
	Value          int32
	SubscriptionID uint32
	SequenceNumber uint32
	ServerIndex    int
	Reachable      bool
	// Repeated is true when the server had already sent that
	// subscription and sequence number before it produced the value.
	Repeated bool
	// SubscriptionInstance is the incarnation ordinal of the
	// subscription on its server: a recreated subscription reuses the
	// wire id of a deleted one while restarting its sequence numbers,
	// so only the instance tells the two apart.
	SubscriptionInstance int
	// Forgotten is true when the server deleted the subscription the
	// value was retained on, so no correct client can obtain it
	// anymore.
	Forgotten bool
}

// ServerState is one server the environment created, with the session
// and subscription counts the harness observed on it.
type ServerState struct {
	Index int
	// Reachable is true when the relay's upstream points at the server
	// or the client's current connection does.
	Reachable bool
	// Connected is true when the client's current connection points
	// at the server.
	Connected     bool
	KnownSessions int
	// ClosingAttempted counts the open sessions for which the recorder
	// saw the client send a CloseSession the network never completed —
	// dropped, held past the client's lifetime, or never answered. The
	// Part 4 exemption such a session carries lives with the matcher
	// that implements it, closeEveryKnownSession in
	// tests/spec/part4/invariants_test.go.
	ClosingAttempted  int
	LiveSubscriptions int
}

// Sentinel is the value the workload answers last, to prove
// publishing resumed.
type Sentinel struct {
	Value      int32
	AnsweredAt time.Time
	// ReceivedAt is zero when the client never received it.
	ReceivedAt time.Time
}

// Observed is a snapshot of what the environment observed, taken
// once before the client closes and once after.
type Observed struct {
	Produced            []Produced
	Received            []int32
	Servers             []ServerState
	ClientSubscriptions int
	FaultEnd            time.Time
	Sentinel            *Sentinel
	Fired               bool

	env *harness.Environment
}

// Observe collects a snapshot of what the environment observed.
func Observe(env *harness.Environment, fault *harness.Injected) Observed {
	servers := env.Servers()
	var states []ServerState
	current := env.Relay.ConnectionCount() - 1
	for index, server := range servers {
		address := strings.TrimPrefix(server.Address(), "opc.tcp://")
		connected := current >= 0 && env.Recorder.UpstreamOf(current) == address
		reachable := connected || env.Relay.Upstream() == address
		states = append(states, ServerState{
			Index:             index,
			Reachable:         reachable,
			Connected:         connected,
			KnownSessions:     server.KnownSessions(),
			ClosingAttempted:  server.SessionsClosingAttempted(),
			LiveSubscriptions: server.LiveSubscriptions(),
		})
	}
	var produced []Produced
	for index, server := range servers {
		for _, entry := range server.Produced() {
			produced = append(produced, Produced{
				Value:                entry.Value,
				SubscriptionID:       entry.SubscriptionID,
				SequenceNumber:       entry.SequenceNumber,
				ServerIndex:          index,
				Reachable:            states[index].Reachable,
				Repeated:             entry.Repeated,
				SubscriptionInstance: entry.SubscriptionInstance,
				Forgotten:            entry.Forgotten,
			})
		}
	}
	return Observed{
		Produced:            produced,
		Received:            env.Received(),
		Servers:             states,
		ClientSubscriptions: len(env.Client.SubscriptionIDs()),
		Fired:               fault.Fired(),
		env:                 env,
	}
}

// WithFaultEnd records the fault's end event the snapshot's
// ResumePublishing measures from.
func (o *Observed) WithFaultEnd(t time.Time) {
	o.FaultEnd = t
}

// WithSentinel records the sentinel value the workload answered and
// when, and fills ReceivedAt from the environment when the client
// received that value.
func (o *Observed) WithSentinel(v int32, answeredAt time.Time) {
	o.Sentinel = &Sentinel{Value: v, AnsweredAt: answeredAt}
	if o.env == nil {
		return
	}
	for _, notification := range o.env.Recorder.Notifications() {
		if notification.Value != v {
			continue
		}
		if _, written, ok := o.env.Recorder.TimesOf(notification.Order); ok {
			o.Sentinel.ReceivedAt = written
		}
	}
}
