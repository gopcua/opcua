package part4

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/matrix"
	"github.com/gopcua/opcua/ua"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// requestsOfType returns the recorded client requests whose decoded
// message is of the request type T.
func requestsOfType[T ua.Request](records []harness.ServiceRecord[ua.Request]) []harness.ServiceRecord[ua.Request] {
	var matched []harness.ServiceRecord[ua.Request]
	for _, record := range records {
		message, decoded := record.Message()
		if !decoded {
			continue
		}
		if _, is := message.(T); is {
			matched = append(matched, record)
		}
	}
	return matched
}

// recordsOnConnection returns the recorded messages that rode the
// given relay connection.
func recordsOnConnection[M any](records []harness.ServiceRecord[M], connection int) []harness.ServiceRecord[M] {
	var on []harness.ServiceRecord[M]
	for _, record := range records {
		if record.Connection == connection {
			on = append(on, record)
		}
	}
	return on
}

// answerTo returns the recorded response that answers the request,
// paired by connection and request id.
func answerTo(request harness.ServiceRecord[ua.Request], responses []harness.ServiceRecord[ua.Response]) (harness.ServiceRecord[ua.Response], bool) {
	for _, response := range responses {
		if response.Connection == request.Connection && response.RequestID == request.RequestID {
			return response, true
		}
	}
	return harness.ServiceRecord[ua.Response]{}, false
}

// statusOf returns the service result a recorded response carries.
func statusOf(response harness.ServiceRecord[ua.Response]) (ua.StatusCode, bool) {
	message, decoded := response.Message()
	if !decoded {
		return 0, false
	}
	return message.Header().ServiceResult, true
}

// republishForSequence returns the recorded Republish request that
// names the sequence number.
func republishForSequence(records []harness.ServiceRecord[ua.Request], sequenceNumber uint32) (harness.ServiceRecord[ua.Request], bool) {
	for _, record := range requestsOfType[*ua.RepublishRequest](records) {
		message, decoded := record.Message()
		if !decoded {
			continue
		}
		if republish, is := message.(*ua.RepublishRequest); is && republish.RetransmitSequenceNumber == sequenceNumber {
			return record, true
		}
	}
	return harness.ServiceRecord[ua.Request]{}, false
}

// requestTypeNames returns the decoded message type of every recorded
// request, for failure messages.
func requestTypeNames(records []harness.ServiceRecord[ua.Request]) []string {
	names := make([]string, 0, len(records))
	for _, record := range records {
		message, decoded := record.Message()
		if !decoded {
			continue
		}
		names = append(names, fmt.Sprintf("%T", message))
	}
	return names
}

// monitoredNode returns the node the client's CreateMonitoredItems
// requests monitored, the harness node the specs read.
func monitoredNode(env *harness.Environment) *ua.NodeID {
	for _, record := range requestsOfType[*ua.CreateMonitoredItemsRequest](env.Recorder.Requests()) {
		message, decoded := record.Message()
		if !decoded {
			continue
		}
		if request, is := message.(*ua.CreateMonitoredItemsRequest); is && len(request.ItemsToCreate) > 0 {
			return request.ItemsToCreate[0].ItemToMonitor.NodeID
		}
	}
	return nil
}

// preCutSessionToken returns the authentication token the client's
// first ActivateSession request carried, the token the pre-cut session
// was activated with.
func preCutSessionToken(env *harness.Environment) *ua.NodeID {
	for _, record := range requestsOfType[*ua.ActivateSessionRequest](env.Recorder.Requests()) {
		message, decoded := record.Message()
		if decoded {
			return message.Header().AuthenticationToken
		}
	}
	return nil
}

// reactivationAnswer waits for the recorded answer to the client's
// ActivateSession carrying the pre-cut session token, and returns it.
func reactivationAnswer(env *harness.Environment, m harness.Mark) harness.ServiceRecord[ua.Response] {
	sessionToken := preCutSessionToken(env)
	gomega.Expect(sessionToken).NotTo(gomega.BeNil(), "the recorder saw no ActivateSession request before the cut")
	var answer harness.ServiceRecord[ua.Response]
	gomega.Eventually(func(g gomega.Gomega) {
		requests := env.Recorder.RequestsSince(m)
		responses := env.Recorder.ResponsesSince(m)
		reactivated := false
		for _, record := range requestsOfType[*ua.ActivateSessionRequest](requests) {
			message, decoded := record.Message()
			if !decoded || !message.Header().AuthenticationToken.Equal(sessionToken) {
				continue
			}
			candidate, answered := answerTo(record, responses)
			if !answered {
				continue
			}
			if status, decoded := statusOf(candidate); decoded && status == ua.StatusOK {
				answer = candidate
				reactivated = true
				break
			}
		}
		g.Expect(reactivated).To(gomega.BeTrue(), "client sent no ActivateSession request carrying the pre-cut authentication token and answered Good after the reconnect")
	}, 15*time.Second).Should(gomega.Succeed())
	return answer
}

// badMessageNotAvailableAnswer returns the recorded answer that the
// server answered Bad_MessageNotAvailable to a Republish request.
func badMessageNotAvailableAnswer(env *harness.Environment, m harness.Mark) (harness.ServiceRecord[ua.Response], bool) {
	responses := env.Recorder.ResponsesSince(m)
	for _, republish := range requestsOfType[*ua.RepublishRequest](env.Recorder.RequestsSince(m)) {
		if answer, answered := answerTo(republish, responses); answered {
			if status, decoded := statusOf(answer); decoded && status == ua.StatusBadMessageNotAvailable {
				return answer, true
			}
		}
	}
	return harness.ServiceRecord[ua.Response]{}, false
}

// waitAnsweredBadMessageNotAvailable waits for the client's Republish
// request the server answered Bad_MessageNotAvailable and returns the
// recorded answer.
func waitAnsweredBadMessageNotAvailable(env *harness.Environment, m harness.Mark) harness.ServiceRecord[ua.Response] {
	var answer harness.ServiceRecord[ua.Response]
	gomega.Eventually(func(g gomega.Gomega) {
		var answered bool
		answer, answered = badMessageNotAvailableAnswer(env, m)
		g.Expect(answered).To(gomega.BeTrue(), "client sent no Republish request that the server answered Bad_MessageNotAvailable")
	}, 15*time.Second).Should(gomega.Succeed())
	return answer
}

// answeredTransferThenNewSubscription says whether the recorded
// traffic already holds a TransferSubscriptions request for oldID
// answered per transferRefusal, followed by an answered
// CreateSubscription request whose answer the relay forwarded to the
// client, and returns their records.
func answeredTransferThenNewSubscription(env *harness.Environment, m harness.Mark, oldID uint32, transferRefusal func(harness.ServiceRecord[ua.Response]) bool) (transferAnswer harness.ServiceRecord[ua.Response], createRequest harness.ServiceRecord[ua.Request], createAnswer harness.ServiceRecord[ua.Response], complete bool) {
	requests := env.Recorder.RequestsSince(m)
	responses := env.Recorder.ResponsesSince(m)
	for _, record := range requestsOfType[*ua.TransferSubscriptionsRequest](requests) {
		message, decoded := record.Message()
		if !decoded {
			continue
		}
		if request, isTransfer := message.(*ua.TransferSubscriptionsRequest); !isTransfer || len(request.SubscriptionIDs) != 1 || request.SubscriptionIDs[0] != oldID {
			continue
		}
		answer, answered := answerTo(record, responses)
		if !answered || !transferRefusal(answer) {
			continue
		}
		for _, create := range requestsOfType[*ua.CreateSubscriptionRequest](requests) {
			if create.Order <= answer.Order {
				continue
			}
			createAnswer, created := answerTo(create, responses)
			if !created {
				continue
			}
			if createAnswer.Fate != harness.Forwarded {
				continue
			}
			if _, isCreate := createAnswer.Message(); isCreate {
				return answer, create, createAnswer, true
			}
		}
		break
	}
	return harness.ServiceRecord[ua.Response]{}, harness.ServiceRecord[ua.Request]{}, harness.ServiceRecord[ua.Response]{}, false
}

// findAnsweredTransferThenNewSubscription returns the records of a
// TransferSubscriptions request for oldID answered per
// transferRefusal and the CreateSubscription request answered after
// it, without asserting they exist; a caller whose rule already
// proved them uses it to pass the records on.
func findAnsweredTransferThenNewSubscription(env *harness.Environment, m harness.Mark, oldID uint32, transferRefusal func(harness.ServiceRecord[ua.Response]) bool) (harness.ServiceRecord[ua.Response], harness.ServiceRecord[ua.Request], harness.ServiceRecord[ua.Response]) {
	transferAnswer, createRequest, createAnswer, _ := answeredTransferThenNewSubscription(env, m, oldID, transferRefusal)
	return transferAnswer, createRequest, createAnswer
}

// receivedNotification is one data change notification a Forwarded
// response carried to the client: its wire order, the subscription it
// belongs to, its sequence number and its value.
type receivedNotification struct {
	order          int
	subscriptionID uint32
	sequenceNumber uint32
	value          int32
}

// receivedNotifications returns the data change notifications the
// relay forwarded to the client: Publish responses name their
// subscription, and a Republish response belongs to the subscription
// its paired Republish request named.
func receivedNotifications(requests []harness.ServiceRecord[ua.Request], responses []harness.ServiceRecord[ua.Response]) []receivedNotification {
	var notifications []receivedNotification
	for _, record := range responses {
		if record.Fate != harness.Forwarded {
			continue
		}
		message, decoded := record.Message()
		if !decoded {
			continue
		}
		var subscriptionID uint32
		var notification *ua.NotificationMessage
		switch response := message.(type) {
		case *ua.PublishResponse:
			subscriptionID = response.SubscriptionID
			notification = response.NotificationMessage
		case *ua.RepublishResponse:
			subscriptionID = republishSubscriptionID(requests, record)
			notification = response.NotificationMessage
		default:
			continue
		}
		value, carries := notificationValue(notification)
		if !carries {
			continue
		}
		notifications = append(notifications, receivedNotification{
			order:          record.Order,
			subscriptionID: subscriptionID,
			sequenceNumber: notification.SequenceNumber,
			value:          value,
		})
	}
	return notifications
}

// republishSubscriptionID returns the subscription id the Republish
// request paired with the response named.
func republishSubscriptionID(requests []harness.ServiceRecord[ua.Request], response harness.ServiceRecord[ua.Response]) uint32 {
	for _, record := range requests {
		if record.Connection != response.Connection || record.RequestID != response.RequestID {
			continue
		}
		message, decoded := record.Message()
		if !decoded {
			continue
		}
		if request, isRepublish := message.(*ua.RepublishRequest); isRepublish {
			return request.SubscriptionID
		}
	}
	return 0
}

// notificationValue returns the int32 data change value a notification
// message carries, and whether it carries one.
func notificationValue(message *ua.NotificationMessage) (int32, bool) {
	if message == nil {
		return 0, false
	}
	for _, data := range message.NotificationData {
		if data == nil || data.Value == nil {
			continue
		}
		change, isDataChange := data.Value.(*ua.DataChangeNotification)
		if !isDataChange || len(change.MonitoredItems) == 0 || change.MonitoredItems[0].Value == nil {
			continue
		}
		value := change.MonitoredItems[0].Value.Value.Value()
		if number, isInt32 := value.(int32); isInt32 {
			return number, true
		}
	}
	return 0, false
}

// skippedSequenceNumbers returns every sequence number the server
// skipped on a subscription: a number missing between two consecutive
// notifications the client received on it, in wire order.
func skippedSequenceNumbers(notifications []receivedNotification) []uint32 {
	bySubscription := map[uint32][]receivedNotification{}
	for _, notification := range notifications {
		bySubscription[notification.subscriptionID] = append(bySubscription[notification.subscriptionID], notification)
	}
	var skipped []uint32
	for _, group := range bySubscription {
		for i := 1; i < len(group); i++ {
			for missing := group[i-1].sequenceNumber + 1; missing < group[i].sequenceNumber; missing++ {
				skipped = append(skipped, missing)
			}
		}
	}
	slices.Sort(skipped)
	return skipped
}

// publishAnsweredTooMany is one Publish request the server answered
// Bad_TooManyPublishRequests, with its answer.
type publishAnsweredTooMany struct {
	request harness.ServiceRecord[ua.Request]
	answer  harness.ServiceRecord[ua.Response]
}

// publishesAnsweredTooMany returns every recorded Publish request the
// server answered Bad_TooManyPublishRequests, with its answer.
func publishesAnsweredTooMany(requests []harness.ServiceRecord[ua.Request], responses []harness.ServiceRecord[ua.Response]) []publishAnsweredTooMany {
	var refused []publishAnsweredTooMany
	for _, request := range requestsOfType[*ua.PublishRequest](requests) {
		answer, answered := answerTo(request, responses)
		if !answered {
			continue
		}
		if status, decoded := statusOf(answer); decoded && status == ua.StatusBadTooManyPublishRequests {
			refused = append(refused, publishAnsweredTooMany{request: request, answer: answer})
		}
	}
	return refused
}

// publishRequest builds a Publish request a hand-built environment
// holds: a request on connection 0 with the given order and request
// id, sent on the session the token names. A hand-built environment
// is harness.RecordedEnvironment fed records these tests construct
// directly, instead of a live harness.New environment.
func publishRequest(order int, requestID uint32, token *ua.NodeID) harness.ServiceRecord[ua.Request] {
	return harness.RecordedRequest(order, 0, requestID, harness.Forwarded, &ua.PublishRequest{
		RequestHeader: &ua.RequestHeader{AuthenticationToken: token},
	})
}

// answeredWithValue builds a Publish response a hand-built environment
// holds: an answer on connection 0 to the given request id, carrying
// one data change notification of v at the sequence number, for the
// given subscription.
func answeredWithValue(order int, requestID uint32, subscriptionID, sequenceNumber uint32, v int32) harness.ServiceRecord[ua.Response] {
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
	// Check asserts through the package-global Gomega, so the collector
	// above replaced the suite's fail handler; restore it, or a later
	// test's assertion would collect silently into a dead slice.
	defer gomega.RegisterFailHandler(Fail)
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
	defer gomega.RegisterFailHandler(Fail)
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
