package part4

import (
	"fmt"
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

func recordsOnConnection[M any](records []spectest.ServiceRecord[M], connection int) []spectest.ServiceRecord[M] {
	var on []spectest.ServiceRecord[M]
	for _, record := range records {
		if record.Connection == connection {
			on = append(on, record)
		}
	}
	return on
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

func deleteNamesSubscription(requests []spectest.ServiceRecord[ua.Request], orderFloor int, id uint32) bool {
	for _, record := range requests {
		if record.Order <= orderFloor {
			continue
		}
		message, decoded := record.Message()
		if !decoded {
			continue
		}
		if request, is := message.(*ua.DeleteSubscriptionsRequest); is {
			for _, named := range request.SubscriptionIDs {
				if named == id {
					return true
				}
			}
		}
	}
	return false
}

func requireNotDeleted(env *spectest.Environment, m spectest.Mark, sub spectest.Subscription, deletedWhen, reason string, orderFloor int) {
	requests := env.Recorder.RequestsSince(m)
	Expect(deleteNamesSubscription(requests, orderFloor, sub.ID())).To(BeFalse(), "client deleted subscription %d %s instead of %s; requests after the first cut: %v", sub.ID(), deletedWhen, reason, requestTypeNames(requests))
}

func requireSubscriptionAlive(env *spectest.Environment, m spectest.Mark, sub spectest.Subscription, deletedWhen string) {
	requireNotDeleted(env, m, sub, deletedWhen, "republishing it", 0)
	requests := env.Recorder.RequestsSince(m)
	Expect(requestsOfType[*ua.CreateSubscriptionRequest](requests)).To(BeEmpty(), "client created a new subscription %s instead of republishing subscription %d; requests after the first cut: %v", deletedWhen, sub.ID(), requestTypeNames(requests))
}

func transferRefusedPerResult(status ua.StatusCode) func(spectest.ServiceRecord[ua.Response]) bool {
	return func(answer spectest.ServiceRecord[ua.Response]) bool {
		message, decoded := answer.Message()
		if !decoded {
			return false
		}
		response, isTransfer := message.(*ua.TransferSubscriptionsResponse)
		return isTransfer && len(response.Results) == 1 && response.Results[0].StatusCode == status
	}
}

func transferAnsweredWithStatus(status ua.StatusCode) func(spectest.ServiceRecord[ua.Response]) bool {
	return func(answer spectest.ServiceRecord[ua.Response]) bool {
		answerStatus, decoded := statusOf(answer)
		return decoded && answerStatus == status
	}
}

func requireTransferAnsweredThenNewSubscription(env *spectest.Environment, m spectest.Mark, oldID uint32, transferAnswered func(spectest.ServiceRecord[ua.Response]) bool) (spectest.ServiceRecord[ua.Response], spectest.ServiceRecord[ua.Request], spectest.ServiceRecord[ua.Response]) {
	var transferAnswer, createAnswer spectest.ServiceRecord[ua.Response]
	var createRequest spectest.ServiceRecord[ua.Request]
	Eventually(func(g Gomega) {
		requests := env.Recorder.RequestsSince(m)
		responses := env.Recorder.ResponsesSince(m)
		complete := false
		for _, record := range requestsOfType[*ua.TransferSubscriptionsRequest](requests) {
			message, decoded := record.Message()
			if !decoded {
				continue
			}
			request, isTransfer := message.(*ua.TransferSubscriptionsRequest)
			if !isTransfer || len(request.SubscriptionIDs) != 1 || request.SubscriptionIDs[0] != oldID {
				continue
			}
			answer, answered := answerTo(record, responses)
			if !answered || !transferAnswered(answer) {
				continue
			}
			for _, createRecord := range requestsOfType[*ua.CreateSubscriptionRequest](requests) {
				if createRecord.Order > answer.Order {
					if response, created := answerTo(createRecord, responses); created {
						transferAnswer = answer
						createRequest = createRecord
						createAnswer = response
						complete = true
					}
					break
				}
			}
			break
		}
		g.Expect(complete).To(BeTrue(), "no TransferSubscriptions request for subscription %d with the scripted answer followed by an answered CreateSubscription request was recorded; requests after the first cut: %v", oldID, requestTypeNames(requests))
	}, 15*time.Second).Should(Succeed())
	return transferAnswer, createRequest, createAnswer
}

func requireNoRepublishBetween(env *spectest.Environment, m spectest.Mark, transferAnswer, createAnswer spectest.ServiceRecord[ua.Response]) {
	sentBetween := false
	for _, record := range requestsOfType[*ua.RepublishRequest](env.Recorder.RequestsSince(m)) {
		if record.Order > transferAnswer.Order && record.Order < createAnswer.Order {
			sentBetween = true
		}
	}
	Expect(sentBetween).To(BeFalse(), "client sent a Republish request between the transfer response and the CreateSubscription response; requests after the first cut: %v", requestTypeNames(env.Recorder.RequestsSince(m)))
}

func noExtraTransferCheck(env *spectest.Environment, m spectest.Mark, transferAnswer, createAnswer spectest.ServiceRecord[ua.Response]) {
}

func transferFailedFlow(env *spectest.Environment, second *spectest.ScriptedServer, m spectest.Mark, transferAnswered func(spectest.ServiceRecord[ua.Response]) bool, extra func(env *spectest.Environment, m spectest.Mark, transferAnswer, createAnswer spectest.ServiceRecord[ua.Response])) {
	oldID := env.Subscription().ID()
	transferAnswer, createRequest, createAnswer := requireTransferAnsweredThenNewSubscription(env, m, oldID, transferAnswered)
	extra(env, m, transferAnswer, createAnswer)
	created := second.WaitCreatedSubscription(m)
	requireNotDeleted(env, m, created, "after the cut", "keeping it", createRequest.Order)
	requireRecreatedMonitoredItem(env, m, created.ID(), createRequest.Order)
	second.WaitHeldPublish().Answer(created, valueAfterReconnect)
	Eventually(func(g Gomega) {
		g.Expect(env.ReceivedSince(m)).To(Equal([]int32{valueAfterReconnect}), "client did not deliver the first notification of the recreated subscription; delivered: %v; errors: %v", env.ReceivedSince(m), env.ReceivedErrorsSince(m))
	}, 15*time.Second).Should(Succeed())
	Expect(second.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", second.UnusedScripts())
}

func transferredFlow(env *spectest.Environment, second *spectest.ScriptedServer, m spectest.Mark, last uint32) {
	Eventually(func(g Gomega) {
		g.Expect(env.ReceivedSince(m)).To(Equal([]int32{valueRetained}), "client did not deliver the retained notification of the transferred subscription; delivered: %v; errors: %v", env.ReceivedSince(m), env.ReceivedErrorsSince(m))
	}, 15*time.Second).Should(Succeed())
	Consistently(func(g Gomega) {
		askedDelivered := false
		for _, record := range requestsOfType[*ua.RepublishRequest](env.Recorder.RequestsSince(m)) {
			message, decoded := record.Message()
			if !decoded {
				continue
			}
			if request, is := message.(*ua.RepublishRequest); is && request.RetransmitSequenceNumber <= last {
				askedDelivered = true
			}
		}
		g.Expect(askedDelivered).To(BeFalse(), "client sent a Republish request for a sequence number at or below %d, the highest it had delivered", last)
	}, 2*time.Second).Should(Succeed())
	Consistently(func(g Gomega) {
		requests := env.Recorder.RequestsSince(m)
		g.Expect(deleteNamesSubscription(requests, 0, env.Subscription().ID())).To(BeFalse(), "client sent a DeleteSubscriptions request naming the transferred subscription %d; requests after the first cut: %v", env.Subscription().ID(), requestTypeNames(requests))
	}, 2*time.Second).Should(Succeed())
	Expect(second.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", second.UnusedScripts())
}

func requestTypeNames(records []spectest.ServiceRecord[ua.Request]) []string {
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
	requireRecreatedMonitoredItem(env, m, createdID, createSubscription.Order)
}

func requireRecreatedMonitoredItem(env *spectest.Environment, m spectest.Mark, id uint32, orderFloor int) {
	node := monitoredNode(env)
	Expect(node).NotTo(BeNil(), "the recorder saw no CreateMonitoredItems request, so the node the client monitors is unknown")
	Eventually(func(g Gomega) {
		requests := env.Recorder.RequestsSince(m)
		sent := false
		for _, record := range requestsOfType[*ua.CreateMonitoredItemsRequest](requests) {
			if record.Order <= orderFloor {
				continue
			}
			message, decoded := record.Message()
			if !decoded {
				continue
			}
			request, is := message.(*ua.CreateMonitoredItemsRequest)
			if !is || request.SubscriptionID != id || len(request.ItemsToCreate) == 0 {
				continue
			}
			item := request.ItemsToCreate[0]
			if item == nil || item.ItemToMonitor == nil || !item.ItemToMonitor.NodeID.Equal(node) {
				continue
			}
			sent = true
			break
		}
		g.Expect(sent).To(BeTrue(), "client sent no CreateMonitoredItems request for the monitored node on subscription %d", id)
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
