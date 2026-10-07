package invariants

import (
	"context"
	"testing"
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/harness"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const specWait = 15 * time.Second

func TestInvariants(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "invariants")
}

func closeClient(env *harness.Environment) {
	ctx, cancel := context.WithTimeout(context.Background(), specWait)
	defer cancel()
	Expect(env.Client.Close(ctx)).To(Succeed(), "closing the client failed")
}

var _ = Describe("Observe", func() {
	It("matches every invariant on an undisturbed environment", func() {
		env := harness.Start(GinkgoT(), harness.WithRetentionQueue())
		sub := env.Subscription()
		caseStart := time.Now()
		env.Server.WaitHeldPublish().Answer(sub, 101)
		env.Server.WaitHeldPublish().Answer(sub, 102)
		env.Server.WaitHeldPublish().Answer(sub, 103)
		sentinelAnswered := time.Now()
		env.Server.WaitHeldPublish().Answer(sub, 199)
		Eventually(func() []int32 { return env.Received() }, specWait).Should(HaveLen(5),
			"the client never received the sentinel along with the answered values")

		observed := Observe(env, nil)
		observed.WithFaultEnd(caseStart)
		observed.WithSentinel(199, sentinelAnswered)

		Expect(observed).To(DeliverEachValueOnce())
		Expect(observed).To(DeliverInOrder())
		Expect(observed).To(ResumePublishing(15 * time.Second))
		Expect(observed).To(KeepOneSessionOpen())
		Expect(observed).To(KeepOneSubscriptionPerClientSubscription())

		closeClient(env)
		Expect(Observe(env, nil)).To(CloseEveryKnownSession())
	})

	It("names a planted loss", func() {
		env := harness.Start(GinkgoT(), harness.WithRetentionQueue())
		sub := env.Subscription()
		caseStart := time.Now()
		env.Server.WaitHeldPublish().Answer(sub, 101)
		env.Server.WaitHeldPublish().Answer(sub, 102)
		env.Server.WaitHeldPublish().Answer(sub, 103)
		sentinelAnswered := time.Now()
		env.Server.WaitHeldPublish().Answer(sub, 199)
		Eventually(func() []int32 { return env.Received() }, specWait).Should(HaveLen(5),
			"the client never received the sentinel along with the answered values")
		sub.Retain(9099, 150)

		observed := Observe(env, nil)
		observed.WithFaultEnd(caseStart)
		observed.WithSentinel(199, sentinelAnswered)

		Expect(observed).To(DeliverInOrder())
		Expect(observed).To(ResumePublishing(15 * time.Second))
		Expect(observed).To(KeepOneSessionOpen())
		Expect(observed).To(KeepOneSubscriptionPerClientSubscription())
		matcher := DeliverEachValueOnce()
		success, err := matcher.Match(observed)
		Expect(err).NotTo(HaveOccurred(), "matching the snapshot failed")
		Expect(success).To(BeFalse(), "the planted loss 150 passed DeliverEachValueOnce")
		Expect(matcher.FailureMessage(observed)).To(ContainSubstring("150"),
			"the failure does not name the planted loss")
	})
})
