package part4

import (
	"time"

	"github.com/gopcua/opcua/tests/spec/rules"
	"github.com/gopcua/opcua/tests/spec/spectest"
	"github.com/gopcua/opcua/ua"

	. "github.com/onsi/gomega"
)

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
	Expect(deleteNamesSubscription(requests, orderFloor, sub.ID())).To(BeFalse(), "client deleted subscription %d %s instead of %s; requests after the first cut: %v", sub.ID(), deletedWhen, reason, rules.RequestTypeNames(requests))
}

func requireSubscriptionAlive(env *spectest.Environment, m spectest.Mark, sub spectest.Subscription, deletedWhen string) {
	requireNotDeleted(env, m, sub, deletedWhen, "republishing it", 0)
	requests := env.Recorder.RequestsSince(m)
	Expect(rules.RequestsOfType[*ua.CreateSubscriptionRequest](requests)).To(BeEmpty(), "client created a new subscription %s instead of republishing subscription %d; requests after the first cut: %v", deletedWhen, sub.ID(), rules.RequestTypeNames(requests))
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
		answerStatus, decoded := rules.StatusOf(answer)
		return decoded && answerStatus == status
	}
}

func requireNoRepublishBetween(env *spectest.Environment, m spectest.Mark, transferAnswer, createAnswer spectest.ServiceRecord[ua.Response]) {
	sentBetween := false
	for _, record := range rules.RequestsOfType[*ua.RepublishRequest](env.Recorder.RequestsSince(m)) {
		if record.Order > transferAnswer.Order && record.Order < createAnswer.Order {
			sentBetween = true
		}
	}
	Expect(sentBetween).To(BeFalse(), "client sent a Republish request between the transfer response and the CreateSubscription response; requests after the first cut: %v", rules.RequestTypeNames(env.Recorder.RequestsSince(m)))
}

func transferFailedFlow(env *spectest.Environment, second *spectest.ScriptedServer, m spectest.Mark, transferAnswered func(spectest.ServiceRecord[ua.Response]) bool, extra func(env *spectest.Environment, m spectest.Mark, transferAnswer, createAnswer spectest.ServiceRecord[ua.Response])) {
	oldID := env.Subscription().ID()
	rules.RecreatesAfterRefusal.Check(rules.Context{Env: env, Mark: m, Sub: env.Subscription(), TransferRefusal: transferAnswered})
	transferAnswer, createRequest, createAnswer := rules.FindAnsweredTransferThenNewSubscription(env, m, oldID, transferAnswered)
	extra(env, m, transferAnswer, createAnswer)
	created := second.WaitCreatedSubscription(m)
	requireNotDeleted(env, m, created, "after the cut", "keeping it", createRequest.Order)
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
		for _, record := range rules.RequestsOfType[*ua.RepublishRequest](env.Recorder.RequestsSince(m)) {
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
		g.Expect(deleteNamesSubscription(requests, 0, env.Subscription().ID())).To(BeFalse(), "client sent a DeleteSubscriptions request naming the transferred subscription %d; requests after the first cut: %v", env.Subscription().ID(), rules.RequestTypeNames(requests))
	}, 2*time.Second).Should(Succeed())
	Expect(second.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", second.UnusedScripts())
}

func notificationCarryingValue(env *spectest.Environment, value int32) (spectest.Notification, bool) {
	for _, notification := range env.Recorder.Notifications() {
		if notification.Value == value {
			return notification, true
		}
	}
	return spectest.Notification{}, false
}
