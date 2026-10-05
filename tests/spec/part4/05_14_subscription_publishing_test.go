package part4

import (
	"time"

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
		requireHeldPublishTimedOut(env, env.Server)
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
		requireHeldPublishTimedOut(env, second)
		Expect(second.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", second.UnusedScripts())
	})
})

func requireHeldPublishTimedOut(env *spectest.Environment, server *spectest.ScriptedServer) {
	held := server.WaitHeldPublish()
	heldOrder, recorded := held.Order()
	Expect(recorded).To(BeTrue(), "the held Publish request was never recorded, so its wire order is unknown")
	Eventually(func(g Gomega) {
		sent := false
		for _, record := range requestsOfType[*ua.PublishRequest](env.Recorder.Requests()) {
			if record.Connection == held.Connection() && record.Order > heldOrder {
				sent = true
				break
			}
		}
		g.Expect(sent).To(BeTrue(), "the client sent no second Publish within %s while the first stayed unanswered, so it never timed out the held Publish", publishTimeoutWait)
	}, publishTimeoutWait).Should(Succeed())
}
