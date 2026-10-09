package part4

import (
	"fmt"
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/matrix"
	"github.com/onsi/gomega/types"

	. "github.com/onsi/gomega"
)

// subscriptionInvariants are the invariants of a workload that
// publishes on a subscription and answers a sentinel last. The matrix
// adds the HaveFired check itself, so no list here names it.
var subscriptionInvariants = []matrix.Invariant{
	{Name: "ResumePublishing", Phase: matrix.BeforeClose, Assert: func(observed matrix.Observed) {
		Expect(observed).To(resumePublishing(15*time.Second), "the sentinel did not resume publishing within its window")
	}},
	{Name: "KeepOneSessionOpen", Phase: matrix.BeforeClose, Assert: func(observed matrix.Observed) {
		Expect(observed).To(keepOneSessionOpen(), "the connected server holds the wrong session count")
	}},
	{Name: "KeepOneSubscriptionPerClientSubscription", Phase: matrix.BeforeClose, Assert: func(observed matrix.Observed) {
		Expect(observed).To(keepOneSubscriptionPerClientSubscription(), "the live subscriptions do not match the client's")
	}},
	{Name: "DeliverEachValueOnce", Phase: matrix.AfterClose, Assert: func(observed matrix.Observed) {
		Expect(observed).To(deliverEachValueOnce(), "a value was not delivered exactly once")
	}},
	{Name: "DeliverInOrder", Phase: matrix.AfterClose, Assert: func(observed matrix.Observed) {
		Expect(observed).To(deliverInOrder(), "values were delivered out of order")
	}},
	{Name: "CloseEveryKnownSession", Phase: matrix.AfterClose, Assert: func(observed matrix.Observed) {
		Expect(observed).To(closeEveryKnownSession(), "a reachable server still holds a session")
	}},
}

func asObserved(actual any) (matrix.Observed, error) {
	observed, ok := actual.(matrix.Observed)
	if !ok {
		return matrix.Observed{}, fmt.Errorf("want an Observed, got %T", actual)
	}
	return observed, nil
}

func connectedServers(observed matrix.Observed) []matrix.ServerState {
	var connected []matrix.ServerState
	for _, server := range observed.Servers {
		if server.Connected {
			connected = append(connected, server)
		}
	}
	return connected
}

// deliverEachValueOnce says every value a reachable server produced
// was received exactly once, and nothing else was received. Produced
// values are deduplicated by value — a retransmission produces the
// same value twice — received values are not. A value produced under
// a sequence number the server had already sent (Produced.Repeated)
// must not be received at all: Part 4 identifies a notification by its
// sequence number, so a client must not deliver a second notification
// under a number it already received. A value whose subscription the
// server deleted (Produced.Forgotten) is exempt like a value on an
// unreachable server: no correct client can obtain it anymore.
func deliverEachValueOnce() types.GomegaMatcher {
	return &deliverEachValueOnceMatcher{}
}

type deliverEachValueOnceMatcher struct {
	failure string
}

func (m *deliverEachValueOnceMatcher) Match(actual any) (bool, error) {
	observed, err := asObserved(actual)
	if err != nil {
		return false, err
	}
	produced := make(map[int32]bool)
	for _, entry := range observed.Produced {
		produced[entry.Value] = true
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
		if entry.Reachable && !entry.Forgotten && received[entry.Value] == 0 {
			m.failure = fmt.Sprintf("server %d produced %d at sequence %d, but the client never received it", entry.ServerIndex, entry.Value, entry.SequenceNumber)
			return false, nil
		}
	}
	return true, nil
}

func (m *deliverEachValueOnceMatcher) FailureMessage(actual any) string {
	return fmt.Sprintf("Expected the observed snapshot to deliver each value once:\n%s", m.failure)
}

func (m *deliverEachValueOnceMatcher) NegatedFailureMessage(actual any) string {
	return "Expected the observed snapshot not to deliver each value once, but it did"
}

// deliverInOrder says the received values of one subscription
// incarnation appear in the sequence order their server produced them
// under.
func deliverInOrder() types.GomegaMatcher {
	return &deliverInOrderMatcher{}
}

type deliverInOrderMatcher struct {
	failure string
}

type subscriptionIdentity struct {
	server   int
	instance int
}

func (m *deliverInOrderMatcher) Match(actual any) (bool, error) {
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

func (m *deliverInOrderMatcher) NegatedFailureMessage(actual any) string {
	return "Expected the observed snapshot not to deliver in order, but it did"
}

func (m *deliverInOrderMatcher) FailureMessage(actual any) string {
	return fmt.Sprintf("Expected the observed snapshot to deliver in order:\n%s", m.failure)
}

// resumePublishing says the sentinel reached the client within the
// window after the fault ended.
func resumePublishing(within time.Duration) types.GomegaMatcher {
	return &resumePublishingMatcher{within: within}
}

type resumePublishingMatcher struct {
	within  time.Duration
	failure string
}

func (m *resumePublishingMatcher) Match(actual any) (bool, error) {
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

func (m *resumePublishingMatcher) FailureMessage(actual any) string {
	return fmt.Sprintf("Expected the observed snapshot to resume publishing:\n%s", m.failure)
}

func (m *resumePublishingMatcher) NegatedFailureMessage(actual any) string {
	return "Expected the observed snapshot not to resume publishing, but it did"
}

// keepOneSessionOpen says the server the client is connected to
// holds exactly one known session. Unreachable servers are exempt,
// and no connected server at all is a failure.
func keepOneSessionOpen() types.GomegaMatcher {
	return &keepOneSessionOpenMatcher{}
}

type keepOneSessionOpenMatcher struct {
	failure string
}

func (m *keepOneSessionOpenMatcher) Match(actual any) (bool, error) {
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

func (m *keepOneSessionOpenMatcher) FailureMessage(actual any) string {
	return fmt.Sprintf("Expected the observed snapshot to keep one session open:\n%s", m.failure)
}

func (m *keepOneSessionOpenMatcher) NegatedFailureMessage(actual any) string {
	return "Expected the observed snapshot not to keep one session open, but it did"
}

// closeEveryKnownSession says no reachable server still holds a
// known session. Unreachable servers are exempt: the client cannot
// close what it can no longer reach. A session whose CloseSession the
// client sent but the network never completed is exempt too: Part 4
// leaves such a session to the server's session timeout.
func closeEveryKnownSession() types.GomegaMatcher {
	return &closeEveryKnownSessionMatcher{}
}

type closeEveryKnownSessionMatcher struct {
	failure string
}

func (m *closeEveryKnownSessionMatcher) Match(actual any) (bool, error) {
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

func (m *closeEveryKnownSessionMatcher) FailureMessage(actual any) string {
	return fmt.Sprintf("Expected the observed snapshot to close every known session:\n%s", m.failure)
}

func (m *closeEveryKnownSessionMatcher) NegatedFailureMessage(actual any) string {
	return "Expected the observed snapshot not to close every known session, but it did"
}

// keepOneSubscriptionPerClientSubscription says the server the client
// is connected to holds exactly one live subscription per
// subscription the client holds.
func keepOneSubscriptionPerClientSubscription() types.GomegaMatcher {
	return &keepOneSubscriptionPerClientSubscriptionMatcher{}
}

type keepOneSubscriptionPerClientSubscriptionMatcher struct {
	failure string
}

func (m *keepOneSubscriptionPerClientSubscriptionMatcher) Match(actual any) (bool, error) {
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

func (m *keepOneSubscriptionPerClientSubscriptionMatcher) FailureMessage(actual any) string {
	return fmt.Sprintf("Expected the observed snapshot to keep one subscription per client subscription:\n%s", m.failure)
}

func (m *keepOneSubscriptionPerClientSubscriptionMatcher) NegatedFailureMessage(actual any) string {
	return "Expected the observed snapshot not to keep one subscription per client subscription, but it did"
}
