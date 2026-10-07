package invariants

import (
	"strings"
	"time"

	"github.com/gopcua/opcua/tests/spec/spectest"
)

// Observe collects a snapshot of what the environment observed: the
// values every server produced with their subscription and sequence
// number, the values the client received, the state of every server
// — reachable when the relay's upstream points at it or the client's
// current connection does, connected when the client's current
// connection does — the client's subscription count, and whether the
// fault fired. WithFaultEnd and WithSentinel fill the fields only the
// caller knows; WithSentinel fills the sentinel's ReceivedAt from
// the environment when the client received the value.
func Observe(env *spectest.Environment, fault *spectest.Injected) Observed {
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
