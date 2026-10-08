package part4

import (
	"strings"
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/matrix"
	"github.com/gopcua/opcua/ua"

	. "github.com/onsi/gomega"
)

func deleteNamesSubscription(requests []harness.ServiceRecord[ua.Request], orderFloor int, id uint32) bool {
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

func requireNotDeleted(env *harness.Environment, m harness.Mark, sub harness.Subscription, deletedWhen, reason string, orderFloor int) {
	requests := env.Recorder.RequestsSince(m)
	Expect(deleteNamesSubscription(requests, orderFloor, sub.ID())).To(BeFalse(), "client deleted subscription %d %s instead of %s; requests after the first cut: %v", sub.ID(), deletedWhen, reason, requestTypeNames(requests))
}

func requireSubscriptionAlive(env *harness.Environment, m harness.Mark, sub harness.Subscription, deletedWhen string) {
	requireNotDeleted(env, m, sub, deletedWhen, "republishing it", 0)
	requests := env.Recorder.RequestsSince(m)
	Expect(requestsOfType[*ua.CreateSubscriptionRequest](requests)).To(BeEmpty(), "client created a new subscription %s instead of republishing subscription %d; requests after the first cut: %v", deletedWhen, sub.ID(), requestTypeNames(requests))
}

func transferRefusedPerResult(status ua.StatusCode) func(harness.ServiceRecord[ua.Response]) bool {
	return func(answer harness.ServiceRecord[ua.Response]) bool {
		message, decoded := answer.Message()
		if !decoded {
			return false
		}
		response, isTransfer := message.(*ua.TransferSubscriptionsResponse)
		return isTransfer && len(response.Results) == 1 && response.Results[0].StatusCode == status
	}
}

func transferAnsweredWithStatus(status ua.StatusCode) func(harness.ServiceRecord[ua.Response]) bool {
	return func(answer harness.ServiceRecord[ua.Response]) bool {
		answerStatus, decoded := statusOf(answer)
		return decoded && answerStatus == status
	}
}

func requireNoRepublishBetween(env *harness.Environment, m harness.Mark, transferAnswer, createAnswer harness.ServiceRecord[ua.Response]) {
	sentBetween := false
	for _, record := range requestsOfType[*ua.RepublishRequest](env.Recorder.RequestsSince(m)) {
		if record.Order > transferAnswer.Order && record.Order < createAnswer.Order {
			sentBetween = true
		}
	}
	Expect(sentBetween).To(BeFalse(), "client sent a Republish request between the transfer response and the CreateSubscription response; requests after the first cut: %v", requestTypeNames(env.Recorder.RequestsSince(m)))
}

func transferFailedFlow(env *harness.Environment, second *harness.ScriptedServer, m harness.Mark, transferAnswered func(harness.ServiceRecord[ua.Response]) bool, extra func(env *harness.Environment, m harness.Mark, transferAnswer, createAnswer harness.ServiceRecord[ua.Response])) {
	oldID := env.Subscription().ID()
	recreatesAfterRefusal.Check(matrix.Context{Env: env, Mark: m, Sub: env.Subscription(), TransferRefusal: transferAnswered})
	transferAnswer, createRequest, createAnswer := findAnsweredTransferThenNewSubscription(env, m, oldID, transferAnswered)
	extra(env, m, transferAnswer, createAnswer)
	created := second.WaitCreatedSubscription(m)
	requireNotDeleted(env, m, created, "after the cut", "keeping it", createRequest.Order)
	second.WaitHeldPublish().Answer(created, valueAfterReconnect)
	Eventually(func(g Gomega) {
		g.Expect(env.ReceivedSince(m)).To(Equal([]int32{valueAfterReconnect}), "client did not deliver the first notification of the recreated subscription; delivered: %v; errors: %v", env.ReceivedSince(m), env.ReceivedErrorsSince(m))
	}, 15*time.Second).Should(Succeed())
	Expect(second.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", second.UnusedScripts())
}

func transferredFlow(env *harness.Environment, second *harness.ScriptedServer, m harness.Mark, last uint32) {
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

func notificationCarryingValue(env *harness.Environment, value int32) (harness.Notification, bool) {
	for _, notification := range env.Recorder.Notifications() {
		if notification.Value == value {
			return notification, true
		}
	}
	return harness.Notification{}, false
}

// targetsPublish says whether the fault arms on the Publish service:
// the message faults that name it and the overload faults that answer
// it.
func targetsPublish(f fault.Fault) bool {
	parts := strings.Split(f.Name(), "/")
	return len(parts) >= 2 && parts[1] == "Publish"
}
