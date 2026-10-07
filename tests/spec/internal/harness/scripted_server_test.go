package harness

import (
	"context"
	"fmt"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func sendRepublish(env *Environment, subscriptionID, sequenceNumber uint32) (ua.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), specWait)
	defer cancel()
	request := &ua.RepublishRequest{
		SubscriptionID:           subscriptionID,
		RetransmitSequenceNumber: sequenceNumber,
	}
	var response ua.Response
	err := env.Client.Send(ctx, request, func(v ua.Response) error {
		response = v
		return nil
	})
	return response, err
}

var _ = Describe("ScriptedServer Republish", func() {
	It("answers a Republish for an id with no harness subscription with Bad_SubscriptionIdInvalid", func() {
		env := New(GinkgoT())
		sub := env.Subscription()

		response, err := sendRepublish(env, sub.ID()+100, 1)

		Expect(err).To(MatchError(ua.StatusBadSubscriptionIDInvalid),
			"a Republish naming an id with no harness subscription must fail with Bad_SubscriptionIdInvalid")
		fault, isFault := response.(*ua.ServiceFault)
		Expect(isFault).To(BeTrue(), "the Republish answered with a %T, want a ServiceFault", response)
		Expect(fault.ResponseHeader.ServiceResult).To(Equal(ua.StatusBadSubscriptionIDInvalid))
	})

	It("answers a Republish for an unretained sequence number with Bad_MessageNotAvailable", func() {
		env := New(GinkgoT())
		sub := env.Subscription()

		response, err := sendRepublish(env, sub.ID(), 44)

		Expect(err).To(MatchError(ua.StatusBadMessageNotAvailable),
			"a Republish for a sequence number the queue does not hold must fail with Bad_MessageNotAvailable")
		fault, isFault := response.(*ua.ServiceFault)
		Expect(isFault).To(BeTrue(), "the Republish answered with a %T, want a ServiceFault", response)
		Expect(fault.ResponseHeader.ServiceResult).To(Equal(ua.StatusBadMessageNotAvailable))
	})

	It("answers one Republish with a scripted ServiceFault through FailRepublish, and the script is then gone", func() {
		env := New(GinkgoT())
		sub := env.Subscription()
		sub.FailRepublish(7, ua.StatusBadSubscriptionIDInvalid)

		response, err := sendRepublish(env, sub.ID(), 7)

		Expect(err).To(MatchError(ua.StatusBadSubscriptionIDInvalid),
			"a Republish for a scripted sequence number must fail with the scripted status")
		fault, isFault := response.(*ua.ServiceFault)
		Expect(isFault).To(BeTrue(), "the Republish answered with a %T, want a ServiceFault", response)
		Expect(fault.ResponseHeader.ServiceResult).To(Equal(ua.StatusBadSubscriptionIDInvalid))

		requestID, sawRequest := republishRequestID(env)
		Expect(sawRequest).To(BeTrue(), "the recorder saw no Republish request")
		Expect(republishFault(env, requestID, ua.StatusBadSubscriptionIDInvalid)).To(BeTrue(),
			"the recorder saw no ServiceFault with the scripted status for request id %d", requestID)

		_, err = sendRepublish(env, sub.ID(), 7)
		Expect(err).To(MatchError(ua.StatusBadMessageNotAvailable),
			"the FailRepublish script must answer one Republish only, so the second one gets the retransmission queue's answer")
	})

	It("answers two Republish requests for a retained sequence number with the retained message", func() {
		env := New(GinkgoT())
		sub := env.Subscription()
		sub.Retain(6, 7100)

		response, err := sendRepublish(env, sub.ID(), 6)
		Expect(err).NotTo(HaveOccurred())
		first, isRepublish := response.(*ua.RepublishResponse)
		Expect(isRepublish).To(BeTrue(), "the Republish answered with a %T, want a RepublishResponse", response)
		Expect(first.NotificationMessage.SequenceNumber).To(Equal(uint32(6)))
		value, carries := dataChangeValue(first.NotificationMessage)
		Expect(carries).To(BeTrue(), "the retained message carries no data change notification")
		Expect(value).To(Equal(int32(7100)))
		Expect(env.Server.UnusedScripts()).To(BeEmpty(),
			"the retained message was answered by a Republish, so no script stays unused: %v", env.Server.UnusedScripts())

		again, err := sendRepublish(env, sub.ID(), 6)
		Expect(err).NotTo(HaveOccurred())
		second, isRepublish := again.(*ua.RepublishResponse)
		Expect(isRepublish).To(BeTrue(), "the second Republish answered with a %T, want a RepublishResponse", again)
		value, carries = dataChangeValue(second.NotificationMessage)
		Expect(carries).To(BeTrue(), "the retained message carries no data change notification")
		Expect(value).To(Equal(int32(7100)))
	})

	It("records the explicit sequence number of AnswerWithSequenceNumber", func() {
		env := New(GinkgoT())
		sub := env.Subscription()

		env.Server.WaitHeldPublish().AnswerWithSequenceNumber(sub, 9, 7600)

		answered := waitAnsweredPublishes(env, []answeredPublish{
			{sequenceNumber: 1, value: valueBeforeCut},
			{sequenceNumber: 9, value: 7600},
		})
		Expect(answered[len(answered)-1].sequenceNumber).To(Equal(uint32(9)),
			"the recorded PublishResponse does not carry the explicit sequence number")
	})

	It("removes a retained message when the Publish request acknowledges its sequence number", func() {
		env := New(GinkgoT())
		sub := env.Subscription()
		sub.Retain(30, 7500)

		env.Server.WaitHeldPublish().AnswerWithSequenceNumber(sub, 30, 7501)
		env.Server.WaitHeldPublish().Answer(sub, 7502)

		answered := waitAnsweredPublishes(env, []answeredPublish{
			{sequenceNumber: 1, value: valueBeforeCut},
			{sequenceNumber: 30, value: 7501},
			{sequenceNumber: 31, value: 7502},
		})
		Expect(answered[1].availableSequenceNumbers).To(Equal([]uint32{30}),
			"Answer must fill AvailableSequenceNumbers with the retained numbers of the subscription")
		Expect(answered[2].results).To(HaveLen(1),
			"the Publish request acknowledged one sequence number, so the response must carry one result")
		Expect(answered[2].results[0]).To(Equal(ua.StatusOK))
		Expect(answered[2].availableSequenceNumbers).To(BeEmpty(),
			"acknowledging the retained sequence number must remove it from the retransmission queue")

		_, err := sendRepublish(env, sub.ID(), 30)
		Expect(err).To(MatchError(ua.StatusBadMessageNotAvailable),
			"a Republish for an acknowledged sequence number must fail with Bad_MessageNotAvailable")
	})

	It("lists unused FailRepublish scripts and unanswered retained messages", func() {
		env := New(GinkgoT())
		sub := env.Subscription()
		sub.FailRepublish(5, ua.StatusBadSubscriptionIDInvalid)
		sub.Retain(4, 7104)

		Expect(env.Server.UnusedScripts()).To(ConsistOf(
			fmt.Sprintf("FailRepublish(5, BadSubscriptionIDInvalid) for subscription %d", sub.ID()),
			fmt.Sprintf("retained %d for subscription %d", 4, sub.ID()),
		))

		_, err := sendRepublish(env, sub.ID(), 4)
		Expect(err).NotTo(HaveOccurred())
		_, err = sendRepublish(env, sub.ID(), 5)
		Expect(err).To(MatchError(ua.StatusBadSubscriptionIDInvalid))

		Expect(env.Server.UnusedScripts()).To(BeEmpty(),
			"the retained message was answered and the FailRepublish script was used, so no script stays unused: %v", env.Server.UnusedScripts())
	})
})

var _ = Describe("ScriptedServer TryWaitHeldPublish", func() {
	It("returns the next held Publish when one arrives within the timeout", func() {
		env := New(GinkgoT())
		held, ok := env.Server.TryWaitHeldPublish(specWait)
		Expect(ok).To(BeTrue(), "the client sent no Publish request within %s", specWait)
		Expect(held.Connection()).To(Equal(0),
			"the held Publish did not arrive on the live connection")
		held.Answer(env.Subscription(), 7301)
		Eventually(func() []int32 { return env.Received() }).WithTimeout(specWait).
			Should(ContainElement(int32(7301)),
				"the client delivered no value the returned held Publish was answered with")
	})

	It("reports no held Publish on a server the client never connects to", func() {
		env := New(GinkgoT())
		second := env.StartServer()
		_, ok := second.TryWaitHeldPublish(200 * time.Millisecond)
		Expect(ok).To(BeFalse(),
			"a server the client never connected to held a Publish request")
	})
})

var _ = Describe("ScriptedServer SubscriptionCreatedSince", func() {
	It("returns a subscription the client created after the mark, without waiting", func() {
		env := New(GinkgoT())
		m := env.Mark()
		_, ok := env.Server.SubscriptionCreatedSince(m)
		Expect(ok).To(BeFalse(), "SubscriptionCreatedSince reported a subscription although the client created none")

		notifications := make(chan *opcua.PublishNotificationData, notificationBuffer)
		subscribeCtx, subscribeCancel := context.WithTimeout(context.Background(), specWait)
		defer subscribeCancel()
		_, err := env.Client.Subscribe(subscribeCtx, &opcua.SubscriptionParameters{
			Interval:          100 * time.Millisecond,
			LifetimeCount:     lifetimeCount,
			MaxKeepAliveCount: maxKeepAliveCount,
		}, notifications)
		Expect(err).NotTo(HaveOccurred(), "the client created no second subscription: %v", err)
		created, ok := env.Server.SubscriptionCreatedSince(m)
		Expect(ok).To(BeTrue(), "SubscriptionCreatedSince reported no subscription although the client created one")
		Expect(created.ID()).NotTo(Equal(env.Subscription().ID()),
			"the returned subscription is the one created before the mark")
	})

	It("reports no subscription on a server the client never connects to", func() {
		env := New(GinkgoT())
		second := env.StartServer()
		_, ok := second.SubscriptionCreatedSince(env.Mark())
		Expect(ok).To(BeFalse(), "a server the client never connected to created a subscription")
	})
})

var _ = Describe("ScriptedServer TryWaitCreatedSubscription", func() {
	It("returns a subscription the client created after the mark", func() {
		env := New(GinkgoT())
		m := env.Mark()
		notifications := make(chan *opcua.PublishNotificationData, notificationBuffer)
		subscribeCtx, subscribeCancel := context.WithTimeout(context.Background(), specWait)
		defer subscribeCancel()
		_, err := env.Client.Subscribe(subscribeCtx, &opcua.SubscriptionParameters{
			Interval:          100 * time.Millisecond,
			LifetimeCount:     lifetimeCount,
			MaxKeepAliveCount: maxKeepAliveCount,
		}, notifications)
		Expect(err).NotTo(HaveOccurred(), "the client created no second subscription: %v", err)
		created, ok := env.Server.TryWaitCreatedSubscription(m, specWait)
		Expect(ok).To(BeTrue(),
			"TryWaitCreatedSubscription reported no subscription although the client created one")
		Expect(created.ID()).NotTo(Equal(env.Subscription().ID()),
			"the returned subscription is the one created before the mark")
	})

	It("reports no subscription on a server the client never connects to", func() {
		env := New(GinkgoT())
		second := env.StartServer()
		_, ok := second.TryWaitCreatedSubscription(env.Mark(), 200*time.Millisecond)
		Expect(ok).To(BeFalse(),
			"a server the client never connected to created a subscription")
	})
})

func republishRequestID(env *Environment) (requestID uint32, found bool) {
	for _, record := range env.Recorder.Requests() {
		message, forwarded := record.Message()
		if !forwarded {
			continue
		}
		if _, isRepublish := message.(*ua.RepublishRequest); isRepublish {
			return record.RequestID, true
		}
	}
	return 0, false
}

func republishFault(env *Environment, requestID uint32, status ua.StatusCode) bool {
	for _, record := range env.Recorder.Responses() {
		if record.RequestID != requestID {
			continue
		}
		message, forwarded := record.Message()
		if !forwarded {
			continue
		}
		if fault, isFault := message.(*ua.ServiceFault); isFault && fault.ResponseHeader.ServiceResult == status {
			return true
		}
	}
	return false
}
