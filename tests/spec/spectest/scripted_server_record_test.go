package spectest

import (
	"github.com/gopcua/opcua/ua"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const valueRecorded1 int32 = 8101
const valueRecorded2 int32 = 8102
const valueDuplicateAnswer int32 = 8103
const valueSkippedAnswer int32 = 8104
const valueStaged int32 = 8105

func publishResponseCarrying(env *Environment, sequenceNumber uint32) *ua.PublishResponse {
	for _, record := range env.Recorder.Responses() {
		message, ok := record.Message()
		if !ok {
			continue
		}
		response, isPublish := message.(*ua.PublishResponse)
		if !isPublish || response.NotificationMessage == nil {
			continue
		}
		if response.NotificationMessage.SequenceNumber == sequenceNumber {
			return response
		}
	}
	return nil
}

func notificationCarryingValue(env *Environment, value int32) (Notification, bool) {
	for _, notification := range env.Recorder.Notifications() {
		if notification.Value == value {
			return notification, true
		}
	}
	return Notification{}, false
}

func producedEntry(produced []Produced, value int32) (Produced, bool) {
	for _, entry := range produced {
		if entry.Value == value {
			return entry, true
		}
	}
	return Produced{}, false
}

var _ = Describe("ScriptedServer record", func() {
	It("keeps an answered value in the retransmission queue until a Publish acknowledges it", func() {
		env := Start(GinkgoT(), WithRetentionQueue())
		sub := env.Subscription()
		held := env.Server.WaitHeldPublish()
		held.Answer(sub, valueRecorded1)

		var answer Notification
		Eventually(func(g Gomega) {
			var found bool
			answer, found = notificationCarryingValue(env, valueRecorded1)
			g.Expect(found).To(BeTrue(), "the recorder saw no notification carrying %d", valueRecorded1)
		}, specWait).Should(Succeed())
		sequence := answer.SequenceNumber
		response := publishResponseCarrying(env, sequence)
		Expect(response).NotTo(BeNil(),
			"the recorder saw no Publish response carrying sequence number %d", sequence)
		Expect(response.AvailableSequenceNumbers).To(ContainElement(sequence),
			"the answer at sequence %d does not list it as available although it is unacknowledged", sequence)

		republish, err := sendRepublish(env, sub.ID(), sequence)
		Expect(err).NotTo(HaveOccurred(),
			"a Republish for the unacknowledged sequence %d failed although the retention queue holds it", sequence)
		_, isRepublish := republish.(*ua.RepublishResponse)
		Expect(isRepublish).To(BeTrue(), "the Republish answered with a %T, want a RepublishResponse", republish)

		acknowledging := env.Server.WaitHeldPublish()
		acknowledging.Answer(sub, valueRecorded2)
		_, err = sendRepublish(env, sub.ID(), sequence)
		Expect(err).To(MatchError(ua.StatusBadMessageNotAvailable),
			"a Republish for the acknowledged sequence %d did not fail with Bad_MessageNotAvailable", sequence)
	})

	It("answers no value past the Publish that acknowledged it without the retention option", func() {
		env := Start(GinkgoT())
		sub := env.Subscription()
		held := env.Server.WaitHeldPublish()
		held.Answer(sub, valueRecorded1)

		var answer Notification
		Eventually(func(g Gomega) {
			var found bool
			answer, found = notificationCarryingValue(env, valueRecorded1)
			g.Expect(found).To(BeTrue(), "the recorder saw no notification carrying %d", valueRecorded1)
		}, specWait).Should(Succeed())
		_, err := sendRepublish(env, sub.ID(), answer.SequenceNumber)
		Expect(err).To(MatchError(ua.StatusBadMessageNotAvailable),
			"a Republish for a sequence the server answered without the retention option did not fail with Bad_MessageNotAvailable")
	})

	It("records every produced value with its subscription id and sequence number", func() {
		env := Start(GinkgoT())
		sub := env.Subscription()
		staged := env.LastSequenceNumber() + 1

		sub.Retain(staged, valueStaged)
		entry, found := producedEntry(env.Server.Produced(), valueStaged)
		Expect(found).To(BeTrue(),
			"Produced holds no entry for %d although Retain(%d, %d) returned", valueStaged, staged, valueStaged)
		Expect(entry.SubscriptionID).To(Equal(sub.ID()),
			"the entry for the retained value names subscription %d, want %d", entry.SubscriptionID, sub.ID())
		Expect(entry.SequenceNumber).To(Equal(staged),
			"the entry for the retained value names sequence %d, want %d", entry.SequenceNumber, staged)

		env.Server.WaitHeldPublish().Answer(sub, valueRecorded1)
		var answer Notification
		Eventually(func(g Gomega) {
			var found bool
			answer, found = notificationCarryingValue(env, valueRecorded1)
			g.Expect(found).To(BeTrue(), "the recorder saw no notification carrying %d", valueRecorded1)
		}, specWait).Should(Succeed())
		entry, found = producedEntry(env.Server.Produced(), valueRecorded1)
		Expect(found).To(BeTrue(), "Produced holds no entry for the answered value %d", valueRecorded1)
		Expect(entry.SubscriptionID).To(Equal(sub.ID()),
			"the entry for the answered value names subscription %d, want %d", entry.SubscriptionID, sub.ID())
		Expect(entry.SequenceNumber).To(Equal(answer.SequenceNumber),
			"the entry for the answered value names sequence %d, want %d", entry.SequenceNumber, answer.SequenceNumber)

		env.Server.WaitHeldPublish().AnswerDuplicate(sub, valueDuplicateAnswer)
		var duplicate Notification
		Eventually(func(g Gomega) {
			var found bool
			duplicate, found = notificationCarryingValue(env, valueDuplicateAnswer)
			g.Expect(found).To(BeTrue(), "the recorder saw no notification carrying %d", valueDuplicateAnswer)
		}, specWait).Should(Succeed())
		entry, found = producedEntry(env.Server.Produced(), valueDuplicateAnswer)
		Expect(found).To(BeTrue(), "Produced holds no entry for the duplicated answer %d", valueDuplicateAnswer)
		Expect(entry.SubscriptionID).To(Equal(sub.ID()),
			"the entry for the duplicated answer names subscription %d, want %d", entry.SubscriptionID, sub.ID())
		Expect(entry.SequenceNumber).To(Equal(duplicate.SequenceNumber),
			"the entry for the duplicated answer names sequence %d, want %d", entry.SequenceNumber, duplicate.SequenceNumber)

		env.Server.WaitHeldPublish().AnswerSkipping(sub, valueSkippedAnswer)
		var skipped Notification
		Eventually(func(g Gomega) {
			var found bool
			skipped, found = notificationCarryingValue(env, valueSkippedAnswer)
			g.Expect(found).To(BeTrue(), "the recorder saw no notification carrying %d", valueSkippedAnswer)
		}, specWait).Should(Succeed())
		entry, found = producedEntry(env.Server.Produced(), valueSkippedAnswer)
		Expect(found).To(BeTrue(), "Produced holds no entry for the skipping answer %d", valueSkippedAnswer)
		Expect(entry.SubscriptionID).To(Equal(sub.ID()),
			"the entry for the skipping answer names subscription %d, want %d", entry.SubscriptionID, sub.ID())
		Expect(entry.SequenceNumber).To(Equal(skipped.SequenceNumber),
			"the entry for the skipping answer names sequence %d, want %d", entry.SequenceNumber, skipped.SequenceNumber)

		last := env.LastSequenceNumber()
		second := env.StartServer()
		moved := second.QueueTransferSuccess(last, last+1)
		moved.Retain(last+1, valueStaged)
		env.Relay.RedirectTo(second.Address())
		env.Relay.Cut()
		env.WaitUntilReconnected()
		Eventually(func() int { return second.LiveSubscriptions() }).WithTimeout(specWait).Should(Equal(1),
			"the client never transferred its subscription to the second server")
		entry, found = producedEntry(second.Produced(), valueStaged)
		Expect(found).To(BeTrue(), "Produced holds no entry for the value staged on the transfer handle")
		Expect(entry.SubscriptionID).To(Equal(sub.ID()),
			"the entry for the staged value names subscription %d, want %d, the id the transfer created it under", entry.SubscriptionID, sub.ID())
		Expect(entry.SequenceNumber).To(Equal(last+1),
			"the entry for the staged value names sequence %d, want %d", entry.SequenceNumber, last+1)
	})

	It("forgets every subscription, and a Republish for a forgotten id fails", func() {
		env := Start(GinkgoT())
		sub := env.Subscription()
		Expect(env.Server.LiveSubscriptions()).To(Equal(1),
			"a server with one client subscription holds %d live subscriptions, want 1", env.Server.LiveSubscriptions())

		env.Server.ForgetSubscriptions()

		Expect(env.Server.LiveSubscriptions()).To(Equal(0),
			"after ForgetSubscriptions the server still holds %d live subscriptions", env.Server.LiveSubscriptions())
		_, err := sendRepublish(env, sub.ID(), 1)
		Expect(err).To(MatchError(ua.StatusBadSubscriptionIDInvalid),
			"a Republish for a forgotten subscription did not fail with Bad_SubscriptionIdInvalid")
	})

	It("answers the next Publish with the last sequence number again, or with one skipped", func() {
		env := Start(GinkgoT())
		sub := env.Subscription()
		env.Server.WaitHeldPublish().Answer(sub, valueRecorded1)
		var last Notification
		Eventually(func(g Gomega) {
			var found bool
			last, found = notificationCarryingValue(env, valueRecorded1)
			g.Expect(found).To(BeTrue(), "the recorder saw no notification carrying %d", valueRecorded1)
		}, specWait).Should(Succeed())

		env.Server.WaitHeldPublish().AnswerDuplicate(sub, valueDuplicateAnswer)
		var duplicate Notification
		Eventually(func(g Gomega) {
			var found bool
			duplicate, found = notificationCarryingValue(env, valueDuplicateAnswer)
			g.Expect(found).To(BeTrue(), "the recorder saw no notification carrying %d", valueDuplicateAnswer)
		}, specWait).Should(Succeed())
		Expect(duplicate.SequenceNumber).To(Equal(last.SequenceNumber),
			"the duplicate answer carries sequence number %d, want %d, the last the server sent", duplicate.SequenceNumber, last.SequenceNumber)

		env.Server.WaitHeldPublish().AnswerSkipping(sub, valueSkippedAnswer)
		var skipped Notification
		Eventually(func(g Gomega) {
			var found bool
			skipped, found = notificationCarryingValue(env, valueSkippedAnswer)
			g.Expect(found).To(BeTrue(), "the recorder saw no notification carrying %d", valueSkippedAnswer)
		}, specWait).Should(Succeed())
		Expect(skipped.SequenceNumber).To(Equal(last.SequenceNumber+2),
			"the skipping answer carries sequence number %d, want %d, the last the server sent plus two", skipped.SequenceNumber, last.SequenceNumber+2)
	})
})
