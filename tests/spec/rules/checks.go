// Package rules holds the Part 4 rules the existing specs assert and
// the failure matrix's scenarios will assert: each rule is one
// behaviour with its clause label and its keyword, and its Check runs
// the assertions for it against the context the caller observed. The
// helpers here search the recorder's records; every rule and every
// spec that still asserts on its own uses them, so no search exists
// twice.
package rules

import (
	"fmt"
	"time"

	"github.com/gopcua/opcua/tests/spec/spectest"
	"github.com/gopcua/opcua/ua"

	"github.com/onsi/gomega"
)

// RequestsOfType returns the recorded client requests whose decoded
// message is of the request type T.
func RequestsOfType[T ua.Request](records []spectest.ServiceRecord[ua.Request]) []spectest.ServiceRecord[ua.Request] {
	var matched []spectest.ServiceRecord[ua.Request]
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

// RecordsOnConnection returns the recorded messages that rode the
// given relay connection.
func RecordsOnConnection[M any](records []spectest.ServiceRecord[M], connection int) []spectest.ServiceRecord[M] {
	var on []spectest.ServiceRecord[M]
	for _, record := range records {
		if record.Connection == connection {
			on = append(on, record)
		}
	}
	return on
}

// AnswerTo returns the recorded response that answers the request,
// paired by connection and request id.
func AnswerTo(request spectest.ServiceRecord[ua.Request], responses []spectest.ServiceRecord[ua.Response]) (spectest.ServiceRecord[ua.Response], bool) {
	for _, response := range responses {
		if response.Connection == request.Connection && response.RequestID == request.RequestID {
			return response, true
		}
	}
	return spectest.ServiceRecord[ua.Response]{}, false
}

// StatusOf returns the service result a recorded response carries.
func StatusOf(response spectest.ServiceRecord[ua.Response]) (ua.StatusCode, bool) {
	message, decoded := response.Message()
	if !decoded {
		return 0, false
	}
	return message.Header().ServiceResult, true
}

// RepublishForSequence returns the recorded Republish request that
// names the sequence number.
func RepublishForSequence(records []spectest.ServiceRecord[ua.Request], sequenceNumber uint32) (spectest.ServiceRecord[ua.Request], bool) {
	for _, record := range RequestsOfType[*ua.RepublishRequest](records) {
		message, decoded := record.Message()
		if !decoded {
			continue
		}
		if republish, is := message.(*ua.RepublishRequest); is && republish.RetransmitSequenceNumber == sequenceNumber {
			return record, true
		}
	}
	return spectest.ServiceRecord[ua.Request]{}, false
}

// RequestTypeNames returns the decoded message type of every recorded
// request, for failure messages.
func RequestTypeNames(records []spectest.ServiceRecord[ua.Request]) []string {
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

// MonitoredNode returns the node the client's CreateMonitoredItems
// requests monitored, the harness node the specs read.
func MonitoredNode(env *spectest.Environment) *ua.NodeID {
	for _, record := range RequestsOfType[*ua.CreateMonitoredItemsRequest](env.Recorder.Requests()) {
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

// PreCutSessionToken returns the authentication token the client's
// first ActivateSession request carried, the token the pre-cut session
// was activated with.
func PreCutSessionToken(env *spectest.Environment) *ua.NodeID {
	for _, record := range RequestsOfType[*ua.ActivateSessionRequest](env.Recorder.Requests()) {
		message, decoded := record.Message()
		if decoded {
			return message.Header().AuthenticationToken
		}
	}
	return nil
}

// reactivationAnswer waits for the recorded answer to the client's
// ActivateSession carrying the pre-cut session token, and returns it.
func reactivationAnswer(env *spectest.Environment, m spectest.Mark) spectest.ServiceRecord[ua.Response] {
	sessionToken := PreCutSessionToken(env)
	gomega.Expect(sessionToken).NotTo(gomega.BeNil(), "the recorder saw no ActivateSession request before the cut")
	var answer spectest.ServiceRecord[ua.Response]
	gomega.Eventually(func(g gomega.Gomega) {
		requests := env.Recorder.RequestsSince(m)
		responses := env.Recorder.ResponsesSince(m)
		reactivated := false
		for _, record := range RequestsOfType[*ua.ActivateSessionRequest](requests) {
			message, decoded := record.Message()
			if !decoded || !message.Header().AuthenticationToken.Equal(sessionToken) {
				continue
			}
			candidate, answered := AnswerTo(record, responses)
			if !answered {
				continue
			}
			if status, decoded := StatusOf(candidate); decoded && status == ua.StatusOK {
				answer = candidate
				reactivated = true
				break
			}
		}
		g.Expect(reactivated).To(gomega.BeTrue(), "client sent no ActivateSession request carrying the pre-cut authentication token and answered Good after the reconnect")
	}, 15*time.Second).Should(gomega.Succeed())
	return answer
}

// BadMessageNotAvailableAnswer returns the recorded answer that the
// server answered Bad_MessageNotAvailable to a Republish request.
func BadMessageNotAvailableAnswer(env *spectest.Environment, m spectest.Mark) (spectest.ServiceRecord[ua.Response], bool) {
	responses := env.Recorder.ResponsesSince(m)
	for _, republish := range RequestsOfType[*ua.RepublishRequest](env.Recorder.RequestsSince(m)) {
		if answer, answered := AnswerTo(republish, responses); answered {
			if status, decoded := StatusOf(answer); decoded && status == ua.StatusBadMessageNotAvailable {
				return answer, true
			}
		}
	}
	return spectest.ServiceRecord[ua.Response]{}, false
}

// WaitAnsweredBadMessageNotAvailable waits for the client's Republish
// request the server answered Bad_MessageNotAvailable and returns the
// recorded answer.
func WaitAnsweredBadMessageNotAvailable(env *spectest.Environment, m spectest.Mark) spectest.ServiceRecord[ua.Response] {
	var answer spectest.ServiceRecord[ua.Response]
	gomega.Eventually(func(g gomega.Gomega) {
		var answered bool
		answer, answered = BadMessageNotAvailableAnswer(env, m)
		g.Expect(answered).To(gomega.BeTrue(), "client sent no Republish request that the server answered Bad_MessageNotAvailable")
	}, 15*time.Second).Should(gomega.Succeed())
	return answer
}

// answeredTransferThenNewSubscription says whether the recorded
// traffic already holds a TransferSubscriptions request for oldID
// answered per transferRefusal, followed by an answered
// CreateSubscription request whose answer the relay forwarded to the
// client, and returns their records.
func answeredTransferThenNewSubscription(env *spectest.Environment, m spectest.Mark, oldID uint32, transferRefusal func(spectest.ServiceRecord[ua.Response]) bool) (transferAnswer spectest.ServiceRecord[ua.Response], createRequest spectest.ServiceRecord[ua.Request], createAnswer spectest.ServiceRecord[ua.Response], complete bool) {
	requests := env.Recorder.RequestsSince(m)
	responses := env.Recorder.ResponsesSince(m)
	for _, record := range RequestsOfType[*ua.TransferSubscriptionsRequest](requests) {
		message, decoded := record.Message()
		if !decoded {
			continue
		}
		if request, isTransfer := message.(*ua.TransferSubscriptionsRequest); !isTransfer || len(request.SubscriptionIDs) != 1 || request.SubscriptionIDs[0] != oldID {
			continue
		}
		answer, answered := AnswerTo(record, responses)
		if !answered || !transferRefusal(answer) {
			continue
		}
		for _, create := range RequestsOfType[*ua.CreateSubscriptionRequest](requests) {
			if create.Order <= answer.Order {
				continue
			}
			createAnswer, created := AnswerTo(create, responses)
			if !created {
				continue
			}
			if createAnswer.Fate != spectest.Forwarded {
				continue
			}
			if _, isCreate := createAnswer.Message(); isCreate {
				return answer, create, createAnswer, true
			}
		}
		break
	}
	return spectest.ServiceRecord[ua.Response]{}, spectest.ServiceRecord[ua.Request]{}, spectest.ServiceRecord[ua.Response]{}, false
}

// FindAnsweredTransferThenNewSubscription returns the records of a
// TransferSubscriptions request for oldID answered per
// transferRefusal and the CreateSubscription request answered after
// it, without asserting they exist; a caller whose rule already
// proved them uses it to pass the records on.
func FindAnsweredTransferThenNewSubscription(env *spectest.Environment, m spectest.Mark, oldID uint32, transferRefusal func(spectest.ServiceRecord[ua.Response]) bool) (spectest.ServiceRecord[ua.Response], spectest.ServiceRecord[ua.Request], spectest.ServiceRecord[ua.Response]) {
	transferAnswer, createRequest, createAnswer, _ := answeredTransferThenNewSubscription(env, m, oldID, transferRefusal)
	return transferAnswer, createRequest, createAnswer
}
