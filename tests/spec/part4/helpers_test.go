package part4

import (
	"time"

	"github.com/gopcua/opcua/tests/spec/spectest"
	"github.com/gopcua/opcua/ua"

	. "github.com/onsi/gomega"
)

func requestsOfType[T ua.Request](records []spectest.ServiceRecord[ua.Request]) []spectest.ServiceRecord[ua.Request] {
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

func answerTo(request spectest.ServiceRecord[ua.Request], responses []spectest.ServiceRecord[ua.Response]) (spectest.ServiceRecord[ua.Response], bool) {
	for _, response := range responses {
		if response.Connection == request.Connection && response.RequestID == request.RequestID {
			return response, true
		}
	}
	return spectest.ServiceRecord[ua.Response]{}, false
}

func statusOf(response spectest.ServiceRecord[ua.Response]) (ua.StatusCode, bool) {
	message, decoded := response.Message()
	if !decoded {
		return 0, false
	}
	return message.Header().ServiceResult, true
}

func republishForSequence(records []spectest.ServiceRecord[ua.Request], sequenceNumber uint32) (spectest.ServiceRecord[ua.Request], bool) {
	for _, record := range requestsOfType[*ua.RepublishRequest](records) {
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

func preCutSessionToken(env *spectest.Environment) *ua.NodeID {
	for _, record := range requestsOfType[*ua.ActivateSessionRequest](env.Recorder.Requests()) {
		message, decoded := record.Message()
		if decoded {
			return message.Header().AuthenticationToken
		}
	}
	return nil
}

func notificationCarryingValue(env *spectest.Environment, value int32) (spectest.Notification, bool) {
	for _, notification := range env.Recorder.Notifications() {
		if notification.Value == value {
			return notification, true
		}
	}
	return spectest.Notification{}, false
}

func monitoredNode(env *spectest.Environment) *ua.NodeID {
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

func waitSubscriptionRecreated(env *spectest.Environment, m spectest.Mark, last uint32) {
	var republishAnswer spectest.ServiceRecord[ua.Response]
	Eventually(func(g Gomega) {
		requests := env.Recorder.RequestsSince(m)
		responses := env.Recorder.ResponsesSince(m)
		complete := false
		if republish, sent := republishForSequence(requests, last+1); sent {
			if answer, answered := answerTo(republish, responses); answered {
				if status, decoded := statusOf(answer); decoded && status == ua.StatusBadSubscriptionIDInvalid {
					republishAnswer = answer
					complete = true
				}
			}
		}
		g.Expect(complete).To(BeTrue(), "no Republish request for sequence number %d answered Bad_SubscriptionIdInvalid was recorded after the reconnect", last+1)
	}, 15*time.Second).Should(Succeed())
	node := monitoredNode(env)
	Expect(node).NotTo(BeNil(), "the recorder saw no CreateMonitoredItems request, so the node the client monitors is unknown")
	var createSubscription spectest.ServiceRecord[ua.Request]
	var createdID uint32
	Eventually(func(g Gomega) {
		requests := env.Recorder.RequestsSince(m)
		responses := env.Recorder.ResponsesSince(m)
		sent := false
		for _, request := range requestsOfType[*ua.CreateSubscriptionRequest](requests) {
			if request.Order > republishAnswer.Order {
				if answer, answered := answerTo(request, responses); answered {
					answerMessage, answerDecoded := answer.Message()
					if response, isCreate := answerMessage.(*ua.CreateSubscriptionResponse); answerDecoded && isCreate {
						createSubscription = request
						createdID = response.SubscriptionID
						sent = true
					}
				}
				break
			}
		}
		g.Expect(sent).To(BeTrue(), "client sent no CreateSubscription request answered with a subscription id after the Republish was answered Bad_SubscriptionIdInvalid")
	}, 15*time.Second).Should(Succeed())
	Eventually(func(g Gomega) {
		requests := env.Recorder.RequestsSince(m)
		sent := false
		for _, record := range requestsOfType[*ua.CreateMonitoredItemsRequest](requests) {
			if record.Order <= createSubscription.Order {
				continue
			}
			message, decoded := record.Message()
			if !decoded {
				continue
			}
			request, is := message.(*ua.CreateMonitoredItemsRequest)
			if !is || request.SubscriptionID != createdID || len(request.ItemsToCreate) == 0 {
				continue
			}
			item := request.ItemsToCreate[0]
			if item == nil || item.ItemToMonitor == nil || !item.ItemToMonitor.NodeID.Equal(node) {
				continue
			}
			sent = true
			break
		}
		g.Expect(sent).To(BeTrue(), "client sent no CreateMonitoredItems request for the monitored node on the recreated subscription")
	}, 15*time.Second).Should(Succeed())
}

func badMessageNotAvailableAnswer(env *spectest.Environment, m spectest.Mark) (spectest.ServiceRecord[ua.Response], bool) {
	responses := env.Recorder.ResponsesSince(m)
	for _, republish := range requestsOfType[*ua.RepublishRequest](env.Recorder.RequestsSince(m)) {
		if answer, answered := answerTo(republish, responses); answered {
			if status, decoded := statusOf(answer); decoded && status == ua.StatusBadMessageNotAvailable {
				return answer, true
			}
		}
	}
	return spectest.ServiceRecord[ua.Response]{}, false
}

func waitAnsweredBadMessageNotAvailable(env *spectest.Environment, m spectest.Mark) spectest.ServiceRecord[ua.Response] {
	var answer spectest.ServiceRecord[ua.Response]
	Eventually(func(g Gomega) {
		var answered bool
		answer, answered = badMessageNotAvailableAnswer(env, m)
		g.Expect(answered).To(BeTrue(), "client sent no Republish request that the server answered Bad_MessageNotAvailable")
	}, 15*time.Second).Should(Succeed())
	return answer
}
