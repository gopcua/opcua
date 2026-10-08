package rules

import (
	"strings"
	"testing"

	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/matrix"
	"github.com/gopcua/opcua/ua"
	"github.com/onsi/gomega"
)

// publishValue builds a Publish response a hand-built environment holds:
// an answer on connection 0 to the given request id, carrying one data
// change notification of v at the sequence number, for the given
// subscription.
func publishValue(order int, requestID uint32, subscriptionID, sequenceNumber uint32, v int32) harness.ServiceRecord[ua.Response] {
	change := &ua.DataChangeNotification{
		MonitoredItems: []*ua.MonitoredItemNotification{{
			Value: server.DataValueFromValue(v),
		}},
	}
	return harness.RecordedResponse(order, 0, requestID, harness.Forwarded, &ua.PublishResponse{
		ResponseHeader: &ua.ResponseHeader{ServiceResult: ua.StatusOK},
		SubscriptionID: subscriptionID,
		NotificationMessage: &ua.NotificationMessage{
			SequenceNumber:   sequenceNumber,
			NotificationData: []*ua.ExtensionObject{ua.NewExtensionObject(change)},
		},
	})
}

// publishRequest builds a Publish request a hand-built environment
// holds: a request on connection 0 with the given order and request id,
// sent on the session the token names.
func publishRequest(order int, requestID uint32, token *ua.NodeID) harness.ServiceRecord[ua.Request] {
	return harness.RecordedRequest(order, 0, requestID, harness.Forwarded, &ua.PublishRequest{
		RequestHeader: &ua.RequestHeader{AuthenticationToken: token},
	})
}

// answeredTooMany builds a Publish response a hand-built environment
// holds: an answer on connection 0 that refuses the request with
// Bad_TooManyPublishRequests.
func answeredTooMany(order int, requestID uint32) harness.ServiceRecord[ua.Response] {
	return harness.RecordedResponse(order, 0, requestID, harness.Forwarded, &ua.PublishResponse{
		ResponseHeader: &ua.ResponseHeader{ServiceResult: ua.StatusBadTooManyPublishRequests},
	})
}

// republishFor builds a Republish request a hand-built environment
// holds: a request for the given subscription and sequence number.
func republishFor(order int, subscriptionID, sequenceNumber uint32) harness.ServiceRecord[ua.Request] {
	return harness.RecordedRequest(order, 0, 0, harness.Forwarded, &ua.RepublishRequest{
		SubscriptionID:           subscriptionID,
		RetransmitSequenceNumber: sequenceNumber,
	})
}

// checkPasses asserts a rule's Check passes against the context the
// hand-built records define.
func checkPasses(t *testing.T, rule matrix.Rule, c matrix.Context) {
	t.Helper()
	var failures []string
	gomega.RegisterFailHandler(func(message string, _ ...int) { failures = append(failures, message) })
	rule.Check(c)
	if len(failures) != 0 {
		t.Fatalf("%s failed against records it must accept: %s", rule.Name, strings.Join(failures, "; "))
	}
}

// checkFails asserts a rule's Check fails against the context the
// hand-built records define, with a message naming want.
func checkFails(t *testing.T, rule matrix.Rule, c matrix.Context, want string) {
	t.Helper()
	var failures []string
	gomega.RegisterFailHandler(func(message string, _ ...int) { failures = append(failures, message) })
	rule.Check(c)
	if len(failures) == 0 {
		t.Fatalf("%s passed against records it must reject", rule.Name)
	}
	for _, message := range failures {
		if !strings.Contains(message, want) {
			t.Errorf("%s failed with %q, want a message naming %q", rule.Name, message, want)
		}
	}
}

func TestRepublishesSkippedSequence(t *testing.T) {
	// No gap: no notification carries a sequence number missing between
	// two the client received on one subscription, so the rule holds
	// vacuously.
	checkPasses(t, RepublishesSkippedSequence, matrix.Context{Env: harness.RecordedEnvironment(t,
		[]harness.ServiceRecord[ua.Request]{
			publishRequest(1, 10, ua.NewTwoByteNodeID(1)),
		},
		[]harness.ServiceRecord[ua.Response]{
			publishValue(2, 10, 5, 1, 101),
			publishValue(3, 11, 5, 2, 102),
		},
		nil)})

	// A gap the client closes with a Republish for the missing number.
	checkPasses(t, RepublishesSkippedSequence, matrix.Context{Env: harness.RecordedEnvironment(t,
		[]harness.ServiceRecord[ua.Request]{
			publishRequest(1, 10, ua.NewTwoByteNodeID(1)),
			republishFor(4, 5, 2),
		},
		[]harness.ServiceRecord[ua.Response]{
			publishValue(2, 10, 5, 1, 101),
			publishValue(3, 11, 5, 3, 102),
		},
		nil)})

	// A gap no Republish closes: the rule fails naming the skipped
	// number.
	checkFails(t, RepublishesSkippedSequence, matrix.Context{Env: harness.RecordedEnvironment(t,
		[]harness.ServiceRecord[ua.Request]{
			publishRequest(1, 10, ua.NewTwoByteNodeID(1)),
			republishFor(4, 5, 8),
		},
		[]harness.ServiceRecord[ua.Response]{
			publishValue(2, 10, 5, 1, 101),
			publishValue(3, 11, 5, 3, 102),
		},
		nil)}, "sequence number 2")
}

func TestPublishesAgainAfterTooManyPublishRequests(t *testing.T) {
	// No Publish was answered Bad_TooManyPublishRequests, so the rule
	// holds vacuously.
	checkPasses(t, PublishesAgainAfterTooManyPublishRequests, matrix.Context{Env: harness.RecordedEnvironment(t,
		[]harness.ServiceRecord[ua.Request]{
			publishRequest(1, 10, ua.NewTwoByteNodeID(1)),
		},
		[]harness.ServiceRecord[ua.Response]{
			publishValue(2, 10, 5, 1, 101),
		},
		[]int32{101})})

	// One was, and the client sent another Publish on the same session,
	// and a value answered after it was delivered.
	checkPasses(t, PublishesAgainAfterTooManyPublishRequests, matrix.Context{Env: harness.RecordedEnvironment(t,
		[]harness.ServiceRecord[ua.Request]{
			publishRequest(1, 10, ua.NewTwoByteNodeID(1)),
			publishRequest(3, 11, ua.NewTwoByteNodeID(1)),
		},
		[]harness.ServiceRecord[ua.Response]{
			answeredTooMany(2, 10),
			publishValue(4, 11, 5, 1, 42),
		},
		[]int32{42})})

	// One was, and the client sent no Publish after it: the rule fails
	// naming the refused request.
	checkFails(t, PublishesAgainAfterTooManyPublishRequests, matrix.Context{Env: harness.RecordedEnvironment(t,
		[]harness.ServiceRecord[ua.Request]{
			publishRequest(1, 10, ua.NewTwoByteNodeID(1)),
			publishRequest(3, 11, ua.NewTwoByteNodeID(2)),
		},
		[]harness.ServiceRecord[ua.Response]{
			answeredTooMany(2, 10),
			publishValue(4, 11, 5, 1, 42),
		},
		[]int32{42})}, "request id 10")

	// One was, another Publish followed on the same session, but no
	// value answered after it was delivered: the rule fails naming the
	// refused request.
	checkFails(t, PublishesAgainAfterTooManyPublishRequests, matrix.Context{Env: harness.RecordedEnvironment(t,
		[]harness.ServiceRecord[ua.Request]{
			publishRequest(1, 10, ua.NewTwoByteNodeID(1)),
			publishRequest(3, 11, ua.NewTwoByteNodeID(1)),
		},
		[]harness.ServiceRecord[ua.Response]{
			answeredTooMany(2, 10),
			publishValue(4, 11, 5, 1, 42),
		},
		nil)}, "request id 10")
}
