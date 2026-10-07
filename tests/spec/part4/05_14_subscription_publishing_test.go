package part4

import (
	"context"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/tests/spec/rules"
	"github.com/gopcua/opcua/tests/spec/spectest"
	"github.com/gopcua/opcua/ua"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	timeoutPublishingInterval = 10 * time.Millisecond
	publishTimeoutWait        = 20 * time.Second
)

var _ = Describe("when the publishing interval is 10 ms", func() {
	It("times out a held Publish on its first subscription", Label("P4-5.14.1.2"), func() {
		env := spectest.Start(GinkgoT(), spectest.WithPublishingInterval(timeoutPublishingInterval))
		held := env.Server.WaitHeldPublish()
		heldOrder, recorded := held.Order()
		Expect(recorded).To(BeTrue(), "the held Publish request was never recorded, so its wire order is unknown")
		// Hold the Publish unanswered past the client's publish
		// timeout: the request times out on the client, which must
		// publish again while the first stays unanswered.
		time.Sleep(env.PublishTimeout() + 2*time.Second)
		rules.RepublishesWithinTimeoutAfterPublishTimeout.Check(rules.Context{Env: env, Server: env.Server, HeldOrder: heldOrder})
	})

	It("times out a held Publish after recreating the subscription", Label("P4-5.14.1.2", "known-defect"), func() {
		env := spectest.Start(GinkgoT(), spectest.WithPublishingInterval(timeoutPublishingInterval))
		second := env.StartServer()
		env.Relay.RedirectTo(second.Address())
		second.QueueTransferRefusal(ua.StatusBadSubscriptionIDInvalid)
		m := env.Mark()
		env.Relay.Cut()
		env.WaitUntilReconnected()
		second.WaitCreatedSubscription(m)
		held := second.WaitHeldPublish()
		heldOrder, recorded := held.Order()
		Expect(recorded).To(BeTrue(), "the held Publish request was never recorded, so its wire order is unknown")
		time.Sleep(env.PublishTimeout() + 2*time.Second)
		rules.RepublishesWithinTimeoutAfterPublishTimeout.Check(rules.Context{Env: env, Server: second, HeldOrder: heldOrder})
		Expect(second.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", second.UnusedScripts())
	})
})

var _ = Describe("when the only subscription is cancelled and a new one created while a Publish is held", func() {
	It("keeps publishing", Label("P4-5.14.1.2", "issue-895", "known-defect"), MustPassRepeatedly(20), func() {
		env := spectest.Start(GinkgoT())
		old := env.ClientSubscription()
		held := env.Server.WaitHeldPublish()
		m := env.Mark()
		cancelCtx, cancelCancel := context.WithTimeout(context.Background(), secondSubscribeWait)
		cancelErr := old.Cancel(cancelCtx)
		cancelCancel()
		Expect(cancelErr).NotTo(HaveOccurred(), "the client cancelled no subscription: %v", cancelErr)
		requireSubscriptionDeleted(env, m, old.SubscriptionID)
		subscribeCtx, subscribeCancel := context.WithTimeout(context.Background(), secondSubscribeWait)
		newSubscription, subscribeErr := env.Client.Subscribe(subscribeCtx, &opcua.SubscriptionParameters{
			Interval:          secondSubscribeInterval,
			LifetimeCount:     secondSubscribeLifetimeCount,
			MaxKeepAliveCount: secondSubscribeKeepAliveCount,
		}, old.Notifs)
		subscribeCancel()
		Expect(subscribeErr).NotTo(HaveOccurred(), "the client created no new subscription: %v", subscribeErr)
		created := env.Server.WaitCreatedSubscription(m)
		node := rules.MonitoredNode(env)
		Expect(node).NotTo(BeNil(), "the recorder saw no CreateMonitoredItems request, so the node the client monitors is unknown")
		monitorCtx, monitorCancel := context.WithTimeout(context.Background(), secondSubscribeWait)
		_, monitorErr := newSubscription.Monitor(monitorCtx, ua.TimestampsToReturnBoth,
			opcua.NewMonitoredItemCreateRequestWithDefaults(node, ua.AttributeIDValue, secondMonitorClientHandle))
		monitorCancel()
		Expect(monitorErr).NotTo(HaveOccurred(), "the client monitored no node on the new subscription: %v", monitorErr)
		held.Answer(created, valueAfterReconnect)
		rules.KeepsPublishingAfterCancelThenSubscribe.Check(rules.Context{Env: env, Mark: m, Value: valueAfterReconnect})
		Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
	})
})

func requireSubscriptionDeleted(env *spectest.Environment, m spectest.Mark, id uint32) {
	responses := env.Recorder.ResponsesSince(m)
	deleted := false
	for _, record := range rules.RequestsOfType[*ua.DeleteSubscriptionsRequest](env.Recorder.RequestsSince(m)) {
		message, decoded := record.Message()
		if !decoded {
			continue
		}
		if request, is := message.(*ua.DeleteSubscriptionsRequest); is && len(request.SubscriptionIDs) == 1 && request.SubscriptionIDs[0] == id {
			if answer, answered := rules.AnswerTo(record, responses); answered {
				answerMessage, answerDecoded := answer.Message()
				if response, isDelete := answerMessage.(*ua.DeleteSubscriptionsResponse); answerDecoded && isDelete && len(response.Results) == 1 && response.Results[0] == ua.StatusOK {
					deleted = true
				}
			}
		}
	}
	Expect(deleted).To(BeTrue(),
		"the client deleted no subscription %d before creating the new one; requests since the mark: %v", id, rules.RequestTypeNames(env.Recorder.RequestsSince(m)))
}
