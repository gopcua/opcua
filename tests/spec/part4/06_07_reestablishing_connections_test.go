package part4

import (
	"slices"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/tests/spec/spectest"
	"github.com/gopcua/opcua/ua"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	valueRetained       int32 = 7001
	valueAfterReconnect int32 = 7002
	valueSentinel       int32 = 7003
	valueDuplicate      int32 = 7004
)

var _ = Describe("Part 4 §6.7 Re-establishing connections https://reference.opcfoundation.org/Core/Part4/v105/docs/6.7", func() {
	var env *spectest.Environment
	BeforeEach(func() { env = spectest.Start(GinkgoT()) })

	Context("when the session survives a transport loss", func() {
		var sub spectest.Subscription
		var last uint32
		var m spectest.Mark
		BeforeEach(func() {
			sub = env.Subscription()
			last = env.LastSequenceNumber()
			sub.Retain(last+1, valueRetained)
			m = env.Mark()
			env.Relay.Cut()
			env.WaitUntilReconnected()
		})

		It("reactivates the existing session instead of creating one", Label("P4-6.7"), func() {
			sessionToken := preCutSessionToken(env)
			Expect(sessionToken).NotTo(BeNil(), "the recorder saw no ActivateSession request before the cut")
			var reactivationAnswer spectest.ServiceRecord[ua.Response]
			Eventually(func(g Gomega) {
				requests := env.Recorder.RequestsSince(m)
				responses := env.Recorder.ResponsesSince(m)
				reactivated := false
				for _, request := range requestsOfType[*ua.ActivateSessionRequest](requests) {
					message, decoded := request.Message()
					if !decoded || !message.Header().AuthenticationToken.Equal(sessionToken) {
						continue
					}
					answer, answered := answerTo(request, responses)
					if !answered {
						continue
					}
					if status, decoded := statusOf(answer); decoded && status == ua.StatusOK {
						reactivationAnswer = answer
						reactivated = true
						break
					}
				}
				g.Expect(reactivated).To(BeTrue(), "client sent no ActivateSession request carrying the pre-cut authentication token and answered Good after the reconnect")
			}, 15*time.Second).Should(Succeed())
			Expect(slices.ContainsFunc(requestsOfType[*ua.CreateSessionRequest](env.Recorder.RequestsSince(m)), func(request spectest.ServiceRecord[ua.Request]) bool {
				return request.Order < reactivationAnswer.Order
			})).To(BeFalse(), "client sent a CreateSession request before the server answered ActivateSession")
			Consistently(func(g Gomega) {
				g.Expect(requestsOfType[*ua.CreateSessionRequest](env.Recorder.RequestsSince(m))).To(BeEmpty(), "client sent a CreateSession request after the server answered ActivateSession")
			}, 2*time.Second).Should(Succeed())
			Expect(env.ConnectionsSince(m)).To(Equal(1), "the relay accepted %d connections after the cut, want exactly one", env.ConnectionsSince(m))
			disconnectedReports := 0
			for _, state := range env.StatesSince(m) {
				if state == opcua.Disconnected {
					disconnectedReports++
				}
			}
			Expect(disconnectedReports).To(Equal(1), "client reported the Disconnected state %d times after the cut, want exactly once", disconnectedReports)
		})

		It("calls Republish from the next expected sequence number, incrementing, until the server answers Bad_MessageNotAvailable", Label("P4-6.7", "issue-879", "known-defect"), func() {
			Eventually(func(g Gomega) {
				requests := env.Recorder.RequestsSince(m)
				responses := env.Recorder.ResponsesSince(m)
				complete := false
				first, firstSent := republishForSequence(requests, last+1)
				if firstSent {
					second, secondSent := republishForSequence(requests, last+2)
					if secondSent && second.Order > first.Order {
						if answer, answered := answerTo(second, responses); answered {
							if status, decoded := statusOf(answer); decoded && status == ua.StatusBadMessageNotAvailable {
								complete = true
							}
						}
					}
				}
				g.Expect(complete).To(BeTrue(), "no Republish request for sequence number %d followed by one for %d answered Bad_MessageNotAvailable was recorded after the reconnect", last+1, last+2)
			}, 15*time.Second).Should(Succeed())
			Consistently(func(g Gomega) {
				republishes := requestsOfType[*ua.RepublishRequest](env.Recorder.RequestsSince(m))
				g.Expect(republishes).To(HaveLen(2), "client sent more than the two expected Republish requests after the cut: %d recorded", len(republishes))
			}, 2*time.Second).Should(Succeed())
			Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
		})

		It("sends no Publish until Republish has answered Bad_MessageNotAvailable", Label("P4-6.7", "issue-879", "known-defect"), func() {
			notAvailableAnswer := waitAnsweredBadMessageNotAvailable(env, m)
			Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
			Expect(slices.ContainsFunc(requestsOfType[*ua.PublishRequest](env.Recorder.RequestsSince(m)), func(request spectest.ServiceRecord[ua.Request]) bool {
				return request.Connection == notAvailableAnswer.Connection && request.Order < notAvailableAnswer.Order
			})).To(BeFalse(), "client sent a Publish request on the new connection before the Republish was answered Bad_MessageNotAvailable")
		})

		It("sends no TransferSubscriptions for a subscription its own session owns", Label("P4-6.7", "P4-5.14.7.4", "known-defect"), func() {
			notAvailableAnswer := waitAnsweredBadMessageNotAvailable(env, m)
			Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
			Expect(slices.ContainsFunc(requestsOfType[*ua.TransferSubscriptionsRequest](env.Recorder.RequestsSince(m)), func(request spectest.ServiceRecord[ua.Request]) bool {
				return request.Order < notAvailableAnswer.Order
			})).To(BeFalse(), "client sent a TransferSubscriptions request before the Republish was answered Bad_MessageNotAvailable")
			Consistently(func(g Gomega) {
				g.Expect(requestsOfType[*ua.TransferSubscriptionsRequest](env.Recorder.RequestsSince(m))).To(BeEmpty(), "client sent a TransferSubscriptions request for a subscription its own session owns")
			}, 2*time.Second).Should(Succeed())
		})

		It("keeps the subscription id it had before the cut", Label("P4-6.7", "issue-879", "known-defect"), func() {
			waitAnsweredBadMessageNotAvailable(env, m)
			Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
			wrongID := false
			for _, republish := range requestsOfType[*ua.RepublishRequest](env.Recorder.RequestsSince(m)) {
				message, decoded := republish.Message()
				if !decoded {
					continue
				}
				if request, is := message.(*ua.RepublishRequest); is && request.SubscriptionID != sub.ID() {
					wrongID = true
				}
			}
			Expect(wrongID).To(BeFalse(), "client sent a Republish request naming a subscription id other than %d", sub.ID())
			Consistently(func(g Gomega) {
				g.Expect(requestsOfType[*ua.CreateSubscriptionRequest](env.Recorder.RequestsSince(m))).To(BeEmpty(), "client created a new subscription instead of keeping subscription %d", sub.ID())
			}, 2*time.Second).Should(Succeed())
		})
	})
})
