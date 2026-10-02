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
		var newConnection int
		BeforeEach(func() {
			sub = env.Subscription()
			last = env.LastSequenceNumber()
			sub.Retain(last+1, valueRetained)
			m = env.Mark()
			newConnection = env.Relay.ConnectionCount()
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

		It("delivers the retained notification, then the following ones, each once and in order", Label("P4-6.7", "issue-879", "known-defect"), MustPassRepeatedly(10), func() {
			waitAnsweredBadMessageNotAvailable(env, m)
			requireSubscriptionAlive(env, m, sub, "after the cut")
			env.Server.WaitHeldPublish().Answer(sub, valueAfterReconnect)
			requireSubscriptionAlive(env, m, sub, "after the cut")
			env.Server.WaitHeldPublish().Answer(sub, valueSentinel)
			Eventually(func(g Gomega) {
				g.Expect(env.ReceivedSince(m)).To(Equal([]int32{valueRetained, valueAfterReconnect, valueSentinel}), "client did not deliver the retained notification then the following ones, each once and in order; delivered: %v", env.ReceivedSince(m))
			}, 15*time.Second).Should(Succeed())
			reconnectNotification, reconnectAnswered := notificationCarryingValue(env, valueAfterReconnect)
			Expect(reconnectAnswered).To(BeTrue(), "the recorder saw no answered Publish response carrying %d", valueAfterReconnect)
			Expect(reconnectNotification.SequenceNumber).To(Equal(last+2), "the Publish answered with valueAfterReconnect carried sequence number %d, want %d", reconnectNotification.SequenceNumber, last+2)
			sentinelNotification, sentinelAnswered := notificationCarryingValue(env, valueSentinel)
			Expect(sentinelAnswered).To(BeTrue(), "the recorder saw no answered Publish response carrying %d", valueSentinel)
			Expect(sentinelNotification.SequenceNumber).To(Equal(last+3), "the Publish answered with valueSentinel carried sequence number %d, want %d", sentinelNotification.SequenceNumber, last+3)
			Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
		})

		It("keeps publishing when a pause and a resume arrive together", Label("P4-5.14.1.2", "issue-895", "known-defect", "racy"), MustPassRepeatedly(20), func() {
			held := env.Server.WaitHeldPublish()
			Expect(held.Connection()).To(Equal(newConnection), "client sent no Publish request on the new connection")
			requireSubscriptionAlive(env, m, sub, "after the cut")
			held.Answer(sub, valueAfterReconnect)
			second := env.Server.WaitHeldPublish()
			Expect(second.Connection()).To(Equal(newConnection), "client sent no further Publish request after the first one was answered")
		})

		It("does not deliver a sequence number twice", Label("P4-6.7", "interop", "known-defect"), func() {
			waitAnsweredBadMessageNotAvailable(env, m)
			requireSubscriptionAlive(env, m, sub, "after the cut")
			env.Server.WaitHeldPublish().AnswerWithSequenceNumber(sub, last+1, valueDuplicate)
			requireSubscriptionAlive(env, m, sub, "after the cut")
			env.Server.WaitHeldPublish().Answer(sub, valueSentinel)
			Eventually(func(g Gomega) {
				g.Expect(env.ReceivedSince(m)).To(ContainElement(valueSentinel), "client did not deliver the notification sent after the duplicated sequence number; delivered: %v", env.ReceivedSince(m))
			}, 15*time.Second).Should(Succeed())
			Expect(env.ReceivedSince(m)).To(Equal([]int32{valueRetained, valueSentinel}), "client delivered %v after the cut, want the retained notification then the sentinel with the duplicated sequence number dropped", env.ReceivedSince(m))
			duplicateNotification, duplicateAnswered := notificationCarryingValue(env, valueDuplicate)
			Expect(duplicateAnswered).To(BeTrue(), "the recorder saw no answered Publish response carrying %d", valueDuplicate)
			Expect(duplicateNotification.SequenceNumber).To(Equal(last+1), "the Publish answered with valueDuplicate carried sequence number %d, want %d", duplicateNotification.SequenceNumber, last+1)
			sentinelNotification, sentinelAnswered := notificationCarryingValue(env, valueSentinel)
			Expect(sentinelAnswered).To(BeTrue(), "the recorder saw no answered Publish response carrying %d", valueSentinel)
			Expect(sentinelNotification.SequenceNumber).To(Equal(last+2), "the Publish answered with valueSentinel carried sequence number %d, want %d", sentinelNotification.SequenceNumber, last+2)
			Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
		})
	})

	Context("when Republish answers Bad_SubscriptionIdInvalid", func() {
		var sub spectest.Subscription
		var last uint32
		var m spectest.Mark
		BeforeEach(func() {
			sub = env.Subscription()
			last = env.LastSequenceNumber()
			sub.FailRepublish(last+1, ua.StatusBadSubscriptionIDInvalid)
			m = env.Mark()
			env.Relay.Cut()
			env.WaitUntilReconnected()
		})

		It("creates a new subscription", Label("P4-6.7", "should", "known-defect"), func() {
			waitSubscriptionRecreated(env, m, last)
			Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
		})

		It("resumes publishing with the new subscription", Label("P4-6.7", "issue-895", "known-defect"), MustPassRepeatedly(10), func() {
			waitSubscriptionRecreated(env, m, last)
			created := env.Server.WaitCreatedSubscription(m)
			held := env.Server.WaitHeldPublish()
			held.Answer(created, valueAfterReconnect)
			Eventually(func(g Gomega) {
				g.Expect(env.ReceivedSince(m)).To(Equal([]int32{valueAfterReconnect}), "client did not resume publishing with the recreated subscription; delivered: %v", env.ReceivedSince(m))
			}, 15*time.Second).Should(Succeed())
			notification, answered := notificationCarryingValue(env, valueAfterReconnect)
			Expect(answered).To(BeTrue(), "the recorder saw no answered Publish response carrying %d", valueAfterReconnect)
			Expect(notification.SubscriptionID).To(Equal(created.ID()), "the Publish answered with valueAfterReconnect was published on subscription %d, want the recreated subscription %d", notification.SubscriptionID, created.ID())
			Expect(notification.SequenceNumber).To(Equal(uint32(1)), "the Publish answered with valueAfterReconnect carried sequence number %d, want 1", notification.SequenceNumber)
			second := env.Server.WaitHeldPublish()
			Expect(second.Connection()).To(Equal(held.Connection()), "client sent no further Publish request on the recreated subscription after the first one was answered")
			Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
		})
	})

	Context("when the connection drops again during Republish recovery", func() {
		var sub spectest.Subscription
		var last uint32
		var m spectest.Mark
		BeforeEach(func() {
			sub = env.Subscription()
			last = env.LastSequenceNumber()
			sub.Retain(last+1, valueRetained)
		})

		DescribeTable("the second cut during Republish recovery",
			func(moment spectest.Moment, check func(env *spectest.Environment, m spectest.Mark, recovery int)) {
				env.Relay.CutAt(moment, spectest.Republish)
				m = env.Mark()
				env.Relay.Cut()
				env.WaitUntilReconnected()
				Eventually(func(g Gomega) {
					g.Expect(env.Relay.ArmedCuts()).To(BeEmpty(), "the armed cut never fired: %v", env.Relay.ArmedCuts())
				}, 15*time.Second).Should(Succeed())
				env.WaitUntilReconnected()
				recovery := 0
				Eventually(func(g Gomega) {
					g.Expect(env.ConnectionsSince(m)).To(Equal(2), "the relay accepted %d connections after the first cut, want exactly two", env.ConnectionsSince(m))
					recovery = env.Relay.ConnectionCount() - 1
				}, 15*time.Second).Should(Succeed())
				check(env, m, recovery)
				requireSubscriptionAlive(env, m, sub, "after the second cut")
				env.Server.WaitHeldPublish().Answer(sub, valueSentinel)
				Eventually(func(g Gomega) {
					g.Expect(env.ReceivedSince(m)).To(Equal([]int32{valueRetained, valueSentinel}), "client did not deliver the retained notification then the sentinel after the first cut; delivered: %v; errors: %v", env.ReceivedSince(m), env.ReceivedErrorsSince(m))
				}, 15*time.Second).Should(Succeed())
				sessionToken := preCutSessionToken(env)
				Expect(sessionToken).NotTo(BeNil(), "the recorder saw no ActivateSession request, so the pre-cut session token is unknown")
				Eventually(func(g Gomega) {
					activations := requestsOfType[*ua.ActivateSessionRequest](env.Recorder.RequestsSince(m))
					onRecovery := false
					for _, record := range activations {
						if record.Connection == recovery {
							onRecovery = true
						}
					}
					g.Expect(onRecovery).To(BeTrue(), "client sent no ActivateSession request on the recovery connection")
				}, 15*time.Second).Should(Succeed())
				Consistently(func(g Gomega) {
					g.Expect(requestsOfType[*ua.CreateSessionRequest](env.Recorder.RequestsSince(m))).To(BeEmpty(), "client sent a CreateSession request after the first cut")
					for _, record := range requestsOfType[*ua.ActivateSessionRequest](env.Recorder.RequestsSince(m)) {
						message, decoded := record.Message()
						if !decoded {
							continue
						}
						g.Expect(message.Header().AuthenticationToken.Equal(sessionToken)).To(BeTrue(), "client sent an ActivateSession request carrying an authentication token other than the pre-cut session's")
					}
				}, 2*time.Second).Should(Succeed())
			},
			Entry("the Republish request is lost (`BeforeRequestReachesServer`)", spectest.BeforeRequestReachesServer, func(env *spectest.Environment, m spectest.Mark, recovery int) {
				Eventually(func(g Gomega) {
					requests := env.Recorder.RequestsSince(m)
					answer, answered := badMessageNotAvailableAnswer(env, m)
					republish, sent := republishForSequence(recordsOnConnection(requests, recovery), last+1)
					g.Expect(sent).To(BeTrue(), "client sent no Republish request for sequence number %d on the recovery connection", last+1)
					g.Expect(answered).To(BeTrue(), "client sent no Republish request answered Bad_MessageNotAvailable")
					g.Expect(answer.Connection).To(Equal(recovery), "the Republish request answered Bad_MessageNotAvailable ran on connection %d, want the recovery connection %d", answer.Connection, recovery)
					g.Expect(answer.Order).To(BeNumerically(">", republish.Order), "the Republish request answered Bad_MessageNotAvailable did not follow the Republish request for sequence number %d on the recovery connection", last+1)
				}, 15*time.Second).Should(Succeed())
			}, Label("P4-6.7", "known-defect")),
			Entry("the connection drops right after the Republish response is delivered (`AfterResponseReachesClient`)", spectest.AfterResponseReachesClient, func(env *spectest.Environment, m spectest.Mark, recovery int) {
				Eventually(func(g Gomega) {
					requests := env.Recorder.RequestsSince(m)
					seen := len(recordsOnConnection(requestsOfType[*ua.RepublishRequest](requests), recovery)) > 0 || len(recordsOnConnection(requestsOfType[*ua.PublishRequest](requests), recovery)) > 0
					g.Expect(seen).To(BeTrue(), "client sent no Republish or Publish request on the recovery connection")
				}, 15*time.Second).Should(Succeed())
				Consistently(func(g Gomega) {
					requests := env.Recorder.RequestsSince(m)
					notifications := env.Recorder.Notifications()
					for _, record := range recordsOnConnection(requestsOfType[*ua.RepublishRequest](requests), recovery) {
						highestDelivered := uint32(0)
						for _, notification := range notifications {
							if notification.Order < record.Order && notification.SequenceNumber > highestDelivered {
								highestDelivered = notification.SequenceNumber
							}
						}
						message, decoded := record.Message()
						if !decoded {
							continue
						}
						if request, is := message.(*ua.RepublishRequest); is {
							g.Expect(request.RetransmitSequenceNumber).To(BeNumerically(">", highestDelivered), "client sent a Republish request on the recovery connection for sequence number %d at or below %d, the highest delivered before it", request.RetransmitSequenceNumber, highestDelivered)
						}
					}
				}, 2*time.Second).Should(Succeed())
			}, Label("P4-6.7", "known-defect")),
		)
	})

	Context("when the session is gone", func() {
		var second *spectest.ScriptedServer
		var last uint32
		var m spectest.Mark
		BeforeEach(func() {
			second = env.StartServer()
			env.Relay.RedirectTo(second.Address())
			last = env.LastSequenceNumber()
		})

		It("creates a new session only after ActivateSession fails", Label("P4-6.7"), func() {
			m = env.Mark()
			env.Relay.Cut()
			env.WaitUntilReconnected()
			Eventually(func(g Gomega) {
				requests := env.Recorder.RequestsSince(m)
				responses := env.Recorder.ResponsesSince(m)
				createSessions := requestsOfType[*ua.CreateSessionRequest](requests)
				preceded := false
				if len(createSessions) > 0 {
					for _, record := range requestsOfType[*ua.ActivateSessionRequest](requests) {
						answer, answered := answerTo(record, responses)
						if !answered {
							continue
						}
						if status, decoded := statusOf(answer); decoded && status == ua.StatusBadSessionIDInvalid && answer.Order < createSessions[0].Order {
							preceded = true
						}
					}
				}
				g.Expect(preceded).To(BeTrue(), "client created a new session without first trying to activate the old one and being refused Bad_SessionIdInvalid; requests after the first cut: %v", requestTypeNames(requests))
			}, 15*time.Second).Should(Succeed())
		})

		DescribeTable("transfers, and creates new subscriptions when the transfer fails",
			func(arm func(second *spectest.ScriptedServer), flow func(env *spectest.Environment, second *spectest.ScriptedServer, m spectest.Mark, last uint32)) {
				arm(second)
				m = env.Mark()
				env.Relay.Cut()
				env.WaitUntilReconnected()
				flow(env, second, m, last)
			},
			Entry("refused per subscription with Bad_SubscriptionIdInvalid",
				func(second *spectest.ScriptedServer) { second.QueueTransferRefusal(ua.StatusBadSubscriptionIDInvalid) },
				func(env *spectest.Environment, second *spectest.ScriptedServer, m spectest.Mark, last uint32) {
					transferFailedFlow(env, second, m, transferRefusedPerResult(ua.StatusBadSubscriptionIDInvalid), noExtraTransferCheck)
				},
				Label("P4-6.7", "P4-5.14.7", "should")),
			Entry("unsupported, with Bad_ServiceUnsupported",
				func(second *spectest.ScriptedServer) {},
				func(env *spectest.Environment, second *spectest.ScriptedServer, m spectest.Mark, last uint32) {
					transferFailedFlow(env, second, m, transferAnsweredWithStatus(ua.StatusBadServiceUnsupported), noExtraTransferCheck)
				},
				Label("P4-6.7", "should")),
			Entry("refused per subscription with Bad_UserAccessDenied",
				func(second *spectest.ScriptedServer) { second.QueueTransferRefusal(ua.StatusBadUserAccessDenied) },
				func(env *spectest.Environment, second *spectest.ScriptedServer, m spectest.Mark, last uint32) {
					transferFailedFlow(env, second, m, transferRefusedPerResult(ua.StatusBadUserAccessDenied), requireNoRepublishBetween)
				},
				Label("P4-6.7", "P4-7.38.1", "known-defect")),
			Entry("transferred, with the last delivered notification still available",
				func(second *spectest.ScriptedServer) {
					moved := second.QueueTransferSuccess(last, last+1)
					moved.Retain(last+1, valueRetained)
				},
				func(env *spectest.Environment, second *spectest.ScriptedServer, m spectest.Mark, last uint32) {
					transferredFlow(env, second, m, last)
				},
				Label("P4-6.7", "P4-5.14.7")),
		)
	})
})
