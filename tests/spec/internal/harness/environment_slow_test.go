package harness

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Environment slow consumer", func() {
	It("reports the blocked drain of a small notification buffer and delivers every value", func() {
		env := New(GinkgoT(), WithNotificationBuffer(4), WithSlowConsumer(200*time.Millisecond))
		sub := env.Subscription()
		m := env.Mark()
		expected := []int32{valueBeforeCut}

		for i := range 10 {
			value := int32(8300 + i)
			expected = append(expected, value)
			env.Server.WaitHeldPublish().Answer(sub, value)
		}

		Expect(env.ConsumerBlocked()).To(BeTrue(),
			"the drain never saw the notification channel full although its buffer holds 4 and 10 answers arrived")
		Eventually(func(g Gomega) {
			g.Expect(env.ReceivedSince(m)).To(Equal(expected[1:]),
				"the client did not deliver every answered value exactly once, in order")
		}, specWait).Should(Succeed())
		Expect(env.Received()).To(Equal(expected),
			"the values delivered before the answers are not the ones the client received")
	})

	It("never reports a blocked drain with the default buffer and pace", func() {
		env := New(GinkgoT())
		sub := env.Subscription()
		m := env.Mark()

		for i := range 10 {
			env.Server.WaitHeldPublish().Answer(sub, int32(8300+i))
		}

		Eventually(func(g Gomega) {
			g.Expect(env.ReceivedSince(m)).To(HaveLen(10),
				"the client did not deliver every answered value")
		}, specWait).Should(Succeed())
		Expect(env.ConsumerBlocked()).To(BeFalse(),
			"the drain saw the notification channel full although its buffer holds %d and the drain keeps pace", notificationBuffer)
	})
})
