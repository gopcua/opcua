// Package invariants holds the Gomega matchers the failure matrix
// asserts its observed snapshots against: each value delivered
// exactly once and in order, publishing resumed within its window,
// one session kept open and every other one closed, one live
// subscription per client subscription, and the fault fired.
package invariants

import (
	"fmt"
	"time"

	"github.com/gopcua/opcua/tests/spec/spectest"
	"github.com/onsi/gomega/types"
)

// Produced is one value a server produced, with the subscription and
// sequence number it was enqueued under and the server that holds it.
// Repeated is true when the server had already sent that subscription
// and sequence number before it produced the value. SubscriptionInstance
// is the incarnation ordinal of the subscription on its server: a
// recreated subscription reuses the wire id of a deleted one while
// restarting its sequence numbers, so only the instance tells the two
// apart.
type Produced struct {
	Value                int32
	SubscriptionID       uint32
	SequenceNumber       uint32
	ServerIndex          int
	Reachable            bool
	Repeated             bool
	SubscriptionInstance int
}

// ServerState is one server the environment created, with the session
// and subscription counts the harness observed on it.
// ClosingAttempted counts the open sessions for which the recorder saw
// the client send a CloseSession the network never completed — dropped,
// held past the client's lifetime, or never answered — which Part 4
// leaves to the server's session timeout.
type ServerState struct {
	Index             int
	Reachable         bool
	Connected         bool
	KnownSessions     int
	ClosingAttempted  int
	LiveSubscriptions int
}

// Sentinel is the value the workload answers last, to prove
// publishing resumed; ReceivedAt is zero when the client never
// received it.
type Sentinel struct {
	Value      int32
	AnsweredAt time.Time
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

	env *spectest.Environment
}

func asObserved(actual any) (Observed, error) {
	observed, ok := actual.(Observed)
	if !ok {
		return Observed{}, fmt.Errorf("want an Observed, got %T", actual)
	}
	return observed, nil
}

func connectedServers(observed Observed) []ServerState {
	var connected []ServerState
	for _, server := range observed.Servers {
		if server.Connected {
			connected = append(connected, server)
		}
	}
	return connected
}

// DeliverEachValueOnce says every value a reachable server produced
// was received exactly once, and nothing else was received. Produced
// values are deduplicated by value — a retransmission produces the
// same value twice — received values are not. A value produced under
// a sequence number the server had already sent (Produced.Repeated)
// must not be received at all: Part 4 identifies a notification by its
// sequence number, so a client must not deliver a second notification
// under a number it already received.
func DeliverEachValueOnce() types.GomegaMatcher {
	return &deliverEachValueOnce{}
}

type deliverEachValueOnce struct {
	failure string
}

func (m *deliverEachValueOnce) Match(actual any) (bool, error) {
	observed, err := asObserved(actual)
	if err != nil {
		return false, err
	}
	reachableProduced := make(map[int32]bool)
	produced := make(map[int32]bool)
	for _, entry := range observed.Produced {
		produced[entry.Value] = true
		if entry.Reachable {
			reachableProduced[entry.Value] = true
		}
	}
	received := make(map[int32]int)
	for _, value := range observed.Received {
		if !produced[value] {
			m.failure = fmt.Sprintf("the client received %d, which no server produced", value)
			return false, nil
		}
		received[value]++
		if received[value] > 1 {
			m.failure = fmt.Sprintf("the client received %d more than once", value)
			return false, nil
		}
	}
	for _, entry := range observed.Produced {
		if entry.Repeated {
			if received[entry.Value] > 0 {
				m.failure = fmt.Sprintf("the client received %d, produced at sequence number %d the server had already sent; a repeated notification must not be delivered", entry.Value, entry.SequenceNumber)
				return false, nil
			}
			continue
		}
		if entry.Reachable && received[entry.Value] == 0 {
			m.failure = fmt.Sprintf("server %d produced %d at sequence %d, but the client never received it", entry.ServerIndex, entry.Value, entry.SequenceNumber)
			return false, nil
		}
	}
	return true, nil
}

func (m *deliverEachValueOnce) FailureMessage(actual any) string {
	return fmt.Sprintf("Expected the observed snapshot to deliver each value once:\n%s", m.failure)
}

func (m *deliverEachValueOnce) NegatedFailureMessage(actual any) string {
	return "Expected the observed snapshot not to deliver each value once, but it did"
}

// DeliverInOrder says the received values of one subscription
// incarnation appear in the sequence order their server produced them
// under.
func DeliverInOrder() types.GomegaMatcher {
	return &deliverInOrder{}
}

type deliverInOrder struct {
	failure string
}

type subscriptionIdentity struct {
	server   int
	instance int
}

func (m *deliverInOrder) Match(actual any) (bool, error) {
	observed, err := asObserved(actual)
	if err != nil {
		return false, err
	}
	sequenceOf := make(map[int32]uint32)
	subscriptionOf := make(map[int32]subscriptionIdentity)
	idOf := make(map[int32]uint32)
	for _, entry := range observed.Produced {
		if _, seen := sequenceOf[entry.Value]; !seen {
			sequenceOf[entry.Value] = entry.SequenceNumber
			subscriptionOf[entry.Value] = subscriptionIdentity{server: entry.ServerIndex, instance: entry.SubscriptionInstance}
			idOf[entry.Value] = entry.SubscriptionID
		}
	}
	type lastSeen struct {
		value    int32
		sequence uint32
	}
	lastOf := make(map[subscriptionIdentity]lastSeen)
	for _, value := range observed.Received {
		sequence, produced := sequenceOf[value]
		if !produced {
			continue
		}
		subscription := subscriptionOf[value]
		if previous, seen := lastOf[subscription]; seen && sequence < previous.sequence {
			m.failure = fmt.Sprintf("the client received %d (sequence %d) before %d (sequence %d) of subscription %d, against their produced order", previous.value, previous.sequence, value, sequence, idOf[value])
			return false, nil
		}
		lastOf[subscription] = lastSeen{value: value, sequence: sequence}
	}
	return true, nil
}

func (m *deliverInOrder) NegatedFailureMessage(actual any) string {
	return "Expected the observed snapshot not to deliver in order, but it did"
}

func (m *deliverInOrder) FailureMessage(actual any) string {
	return fmt.Sprintf("Expected the observed snapshot to deliver in order:\n%s", m.failure)
}

// ResumePublishing says the sentinel reached the client within the
// window after the fault ended.
func ResumePublishing(within time.Duration) types.GomegaMatcher {
	return &resumePublishing{within: within}
}

type resumePublishing struct {
	within  time.Duration
	failure string
}

func (m *resumePublishing) Match(actual any) (bool, error) {
	observed, err := asObserved(actual)
	if err != nil {
		return false, err
	}
	if observed.Sentinel == nil {
		m.failure = "no sentinel was staged, so publishing resumption cannot be checked"
		return false, nil
	}
	if observed.Sentinel.ReceivedAt.IsZero() {
		m.failure = fmt.Sprintf("the sentinel %d was never received", observed.Sentinel.Value)
		return false, nil
	}
	delay := observed.Sentinel.ReceivedAt.Sub(observed.FaultEnd)
	if delay > m.within {
		m.failure = fmt.Sprintf("the sentinel %d arrived %s after the fault ended, want within %s", observed.Sentinel.Value, delay, m.within)
		return false, nil
	}
	return true, nil
}

func (m *resumePublishing) FailureMessage(actual any) string {
	return fmt.Sprintf("Expected the observed snapshot to resume publishing:\n%s", m.failure)
}

func (m *resumePublishing) NegatedFailureMessage(actual any) string {
	return "Expected the observed snapshot not to resume publishing, but it did"
}

// KeepOneSessionOpen says the server the client is connected to
// holds exactly one known session. Unreachable servers are exempt,
// and no connected server at all is a failure.
func KeepOneSessionOpen() types.GomegaMatcher {
	return &keepOneSessionOpen{}
}

type keepOneSessionOpen struct {
	failure string
}

func (m *keepOneSessionOpen) Match(actual any) (bool, error) {
	observed, err := asObserved(actual)
	if err != nil {
		return false, err
	}
	connected := connectedServers(observed)
	if len(connected) == 0 {
		m.failure = "the client is connected to no server"
		return false, nil
	}
	for _, server := range connected {
		if server.KnownSessions != 1 {
			m.failure = fmt.Sprintf("the connected server %d holds %d known sessions, want 1", server.Index, server.KnownSessions)
			return false, nil
		}
	}
	return true, nil
}

func (m *keepOneSessionOpen) FailureMessage(actual any) string {
	return fmt.Sprintf("Expected the observed snapshot to keep one session open:\n%s", m.failure)
}

func (m *keepOneSessionOpen) NegatedFailureMessage(actual any) string {
	return "Expected the observed snapshot not to keep one session open, but it did"
}

// CloseEveryKnownSession says no reachable server still holds a
// known session. Unreachable servers are exempt: the client cannot
// close what it can no longer reach. A session whose CloseSession the
// client sent but the network never completed is exempt too: Part 4
// leaves such a session to the server's session timeout.
func CloseEveryKnownSession() types.GomegaMatcher {
	return &closeEveryKnownSession{}
}

type closeEveryKnownSession struct {
	failure string
}

func (m *closeEveryKnownSession) Match(actual any) (bool, error) {
	observed, err := asObserved(actual)
	if err != nil {
		return false, err
	}
	for _, server := range observed.Servers {
		if server.Reachable && server.KnownSessions-server.ClosingAttempted > 0 {
			m.failure = fmt.Sprintf("server %d still holds %d known sessions", server.Index, server.KnownSessions-server.ClosingAttempted)
			return false, nil
		}
	}
	return true, nil
}

func (m *closeEveryKnownSession) FailureMessage(actual any) string {
	return fmt.Sprintf("Expected the observed snapshot to close every known session:\n%s", m.failure)
}

func (m *closeEveryKnownSession) NegatedFailureMessage(actual any) string {
	return "Expected the observed snapshot not to close every known session, but it did"
}

// KeepOneSubscriptionPerClientSubscription says the server the client
// is connected to holds exactly one live subscription per
// subscription the client holds.
func KeepOneSubscriptionPerClientSubscription() types.GomegaMatcher {
	return &keepOneSubscriptionPerClientSubscription{}
}

type keepOneSubscriptionPerClientSubscription struct {
	failure string
}

func (m *keepOneSubscriptionPerClientSubscription) Match(actual any) (bool, error) {
	observed, err := asObserved(actual)
	if err != nil {
		return false, err
	}
	connected := connectedServers(observed)
	if len(connected) == 0 {
		m.failure = "the client is connected to no server"
		return false, nil
	}
	for _, server := range connected {
		if server.LiveSubscriptions != observed.ClientSubscriptions {
			m.failure = fmt.Sprintf("the connected server %d holds %d live subscriptions for %d client subscriptions", server.Index, server.LiveSubscriptions, observed.ClientSubscriptions)
			return false, nil
		}
	}
	return true, nil
}

func (m *keepOneSubscriptionPerClientSubscription) FailureMessage(actual any) string {
	return fmt.Sprintf("Expected the observed snapshot to keep one subscription per client subscription:\n%s", m.failure)
}

func (m *keepOneSubscriptionPerClientSubscription) NegatedFailureMessage(actual any) string {
	return "Expected the observed snapshot not to keep one subscription per client subscription, but it did"
}

// HaveFired says the fault the case armed fired.
func HaveFired() types.GomegaMatcher {
	return &haveFired{}
}

type haveFired struct{}

func (m *haveFired) Match(actual any) (bool, error) {
	observed, err := asObserved(actual)
	if err != nil {
		return false, err
	}
	return observed.Fired, nil
}

func (m *haveFired) FailureMessage(actual any) string {
	return "Expected the fault to have fired, but it never did"
}

func (m *haveFired) NegatedFailureMessage(actual any) string {
	return "Expected the fault not to have fired, but it did"
}
