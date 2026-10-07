package part4

import (
	"context"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/tests/spec/internal/message"
	"github.com/gopcua/opcua/tests/spec/internal/rules"
	"github.com/gopcua/opcua/tests/spec/internal/spectest"
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
			ctx := rules.Context{Env: env, Mark: m}
			rules.ReactivatesSession.Check(ctx)
			rules.CreatesNoSession.Check(ctx)
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
			rules.RepublishesFromNextSequence.Check(rules.Context{Env: env, Mark: m, LastSeq: last})
			Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
		})

		It("sends no Publish until Republish has answered Bad_MessageNotAvailable", Label("P4-6.7", "issue-879", "known-defect"), func() {
			rules.SendsNoPublishBeforeNotAvailable.Check(rules.Context{Env: env, Mark: m})
			Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
		})

		It("sends no TransferSubscriptions for a subscription its own session owns", Label("P4-6.7", "P4-5.14.7.4", "known-defect"), func() {
			rules.SendsNoTransferForOwnSubscription.Check(rules.Context{Env: env, Mark: m})
			Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
		})

		It("keeps the subscription id it had before the cut", Label("P4-6.7", "issue-879", "known-defect"), func() {
			rules.KeepsSubscriptionID.Check(rules.Context{Env: env, Mark: m, Sub: sub})
			Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
		})

		It("delivers the retained notification, then the following ones, each once and in order", Label("P4-6.7", "issue-879", "known-defect"), MustPassRepeatedly(10), func() {
			rules.WaitAnsweredBadMessageNotAvailable(env, m)
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

		It("does not deliver a sequence number twice", Label("P4-6.7", "interop", "known-defect"), func() {
			rules.WaitAnsweredBadMessageNotAvailable(env, m)
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
			rules.RecreatesAfterRefusal.Check(rules.Context{Env: env, Mark: m, LastSeq: last})
			Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
		})

		It("resumes publishing with the new subscription", Label("P4-6.7", "issue-895", "known-defect"), MustPassRepeatedly(10), func() {
			rules.RecreatesAfterRefusal.Check(rules.Context{Env: env, Mark: m, LastSeq: last})
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
				env.Relay.CutAt(moment, message.Republish)
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
				sessionToken := rules.PreCutSessionToken(env)
				Expect(sessionToken).NotTo(BeNil(), "the recorder saw no ActivateSession request, so the pre-cut session token is unknown")
				Eventually(func(g Gomega) {
					activations := rules.RequestsOfType[*ua.ActivateSessionRequest](env.Recorder.RequestsSince(m))
					onRecovery := false
					for _, record := range activations {
						if record.Connection == recovery {
							onRecovery = true
						}
					}
					g.Expect(onRecovery).To(BeTrue(), "client sent no ActivateSession request on the recovery connection")
				}, 15*time.Second).Should(Succeed())
				Consistently(func(g Gomega) {
					g.Expect(rules.RequestsOfType[*ua.CreateSessionRequest](env.Recorder.RequestsSince(m))).To(BeEmpty(), "client sent a CreateSession request after the first cut")
					for _, record := range rules.RequestsOfType[*ua.ActivateSessionRequest](env.Recorder.RequestsSince(m)) {
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
					answer, answered := rules.BadMessageNotAvailableAnswer(env, m)
					republish, sent := rules.RepublishForSequence(rules.RecordsOnConnection(requests, recovery), last+1)
					g.Expect(sent).To(BeTrue(), "client sent no Republish request for sequence number %d on the recovery connection", last+1)
					g.Expect(answered).To(BeTrue(), "client sent no Republish request answered Bad_MessageNotAvailable")
					g.Expect(answer.Connection).To(Equal(recovery), "the Republish request answered Bad_MessageNotAvailable ran on connection %d, want the recovery connection %d", answer.Connection, recovery)
					g.Expect(answer.Order).To(BeNumerically(">", republish.Order), "the Republish request answered Bad_MessageNotAvailable did not follow the Republish request for sequence number %d on the recovery connection", last+1)
				}, 15*time.Second).Should(Succeed())
			}, Label("P4-6.7", "known-defect")),
			Entry("the connection drops right after the Republish response is delivered (`AfterResponseReachesClient`)", spectest.AfterResponseReachesClient, func(env *spectest.Environment, m spectest.Mark, recovery int) {
				Eventually(func(g Gomega) {
					requests := env.Recorder.RequestsSince(m)
					seen := len(rules.RecordsOnConnection(rules.RequestsOfType[*ua.RepublishRequest](requests), recovery)) > 0 || len(rules.RecordsOnConnection(rules.RequestsOfType[*ua.PublishRequest](requests), recovery)) > 0
					g.Expect(seen).To(BeTrue(), "client sent no Republish or Publish request on the recovery connection")
				}, 15*time.Second).Should(Succeed())
				Consistently(func(g Gomega) {
					requests := env.Recorder.RequestsSince(m)
					notifications := env.Recorder.Notifications()
					for _, record := range rules.RecordsOnConnection(rules.RequestsOfType[*ua.RepublishRequest](requests), recovery) {
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
			rules.CreatesSessionOnlyAfterActivateFailed.Check(rules.Context{Env: env, Mark: m})
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
					transferFailedFlow(env, second, m, transferRefusedPerResult(ua.StatusBadSubscriptionIDInvalid), requireRecreatedCarriesFirstSubscriptionParameters)
				},
				Label("P4-6.7", "P4-5.14.7", "should")),
			Entry("unsupported, with Bad_ServiceUnsupported",
				func(second *spectest.ScriptedServer) {},
				func(env *spectest.Environment, second *spectest.ScriptedServer, m spectest.Mark, last uint32) {
					transferFailedFlow(env, second, m, transferAnsweredWithStatus(ua.StatusBadServiceUnsupported), requireRecreatedCarriesFirstSubscriptionParameters)
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

		It("opens no new connection once closed", Label("P4-6.7"), func() {
			m := env.Mark()
			_ = env.Client.Close(context.Background())
			Eventually(func(g Gomega) {
				g.Expect(env.StatesSince(m)).To(ContainElement(opcua.Closed),
					"the client reported no Closed state after being closed; states since the mark taken before the close: %v", env.StatesSince(m))
			}, noReconnectWindow).Should(Succeed())
			Consistently(func(g Gomega) {
				g.Expect(env.ConnectionsSince(m)).To(Equal(0),
					"the client opened %d new relay connections since the mark taken before the close", env.ConnectionsSince(m))
				states := env.StatesSince(m)
				g.Expect(states[len(states)-1]).To(Equal(opcua.Closed),
					"the client reported the state %v after Closed; states since the mark taken before the close: %v", states[len(states)-1], states)
			}, noReconnectWindow).Should(Succeed())
		})

		It("republishes a recreated subscription from sequence number 1", Label("P4-6.7", "issue-879", "known-defect"), func() {
			second.QueueTransferRefusal(ua.StatusBadSubscriptionIDInvalid)
			m = env.Mark()
			env.Relay.Cut()
			env.WaitUntilReconnected()
			created := second.WaitCreatedSubscription(m)
			env.Relay.Cut()
			rules.RepublishesRecreatedFromOne.Check(rules.Context{Env: env, Mark: m, Recreated: created})
			Expect(second.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", second.UnusedScripts())
		})
	})
})

const dataRaceWindow = 3 * time.Second

var _ = Describe("when the client is closed while it re-dials", func() {
	It("does not race Close against the reconnect Dial", Label("P4-6.7", "issue-883", "known-defect"), func() {
		AddReportEntry("data-race", []string{"(*Client).Close", "(*Client).Dial"})
		env := spectest.Start(GinkgoT())
		ctx := context.Background()
		deadline := time.Now().Add(dataRaceWindow)
		var closing, dialing sync.WaitGroup
		closing.Add(1)
		go func() {
			defer closing.Done()
			for time.Now().Before(deadline) {
				_ = env.Client.Close(ctx)
			}
		}()
		dialing.Add(1)
		go func() {
			defer dialing.Done()
			for time.Now().Before(deadline) {
				_ = env.Client.Dial(ctx)
			}
		}()
		closing.Wait()
		dialing.Wait()
	})
})

func requireRecreatedCarriesFirstSubscriptionParameters(env *spectest.Environment, m spectest.Mark, _, createAnswer spectest.ServiceRecord[ua.Response]) {
	var first *ua.CreateSubscriptionRequest
	for _, record := range rules.RequestsOfType[*ua.CreateSubscriptionRequest](env.Recorder.Requests()) {
		message, _ := record.Message()
		first, _ = message.(*ua.CreateSubscriptionRequest)
		break
	}
	Expect(first).NotTo(BeNil(), "the recorder saw no CreateSubscription request before the cut")
	var recreated *ua.CreateSubscriptionRequest
	for _, record := range rules.RequestsOfType[*ua.CreateSubscriptionRequest](env.Recorder.RequestsSince(m)) {
		if record.Connection != createAnswer.Connection || record.RequestID != createAnswer.RequestID {
			continue
		}
		message, _ := record.Message()
		recreated, _ = message.(*ua.CreateSubscriptionRequest)
		break
	}
	Expect(recreated).NotTo(BeNil(), "the recorder saw no CreateSubscription request for the recreated subscription")
	Expect(recreated.RequestedPublishingInterval).To(Equal(first.RequestedPublishingInterval),
		"the recreating CreateSubscription requests publishing interval %v, want %v, the first subscription's", recreated.RequestedPublishingInterval, first.RequestedPublishingInterval)
	Expect(recreated.RequestedLifetimeCount).To(Equal(first.RequestedLifetimeCount),
		"the recreating CreateSubscription requests lifetime count %d, want %d, the first subscription's", recreated.RequestedLifetimeCount, first.RequestedLifetimeCount)
	Expect(recreated.RequestedMaxKeepAliveCount).To(Equal(first.RequestedMaxKeepAliveCount),
		"the recreating CreateSubscription requests max keep-alive count %d, want %d, the first subscription's", recreated.RequestedMaxKeepAliveCount, first.RequestedMaxKeepAliveCount)
}

const (
	reconnectIntervalLong  = 10 * time.Second
	reconnectWithoutWait   = 5 * time.Second
	noReconnectWindow      = 500 * time.Millisecond
	recreatedRepublishWait = 15 * time.Second
	deliveredAfterHoldWait = 15 * time.Second
)

var _ = Describe("when the reconnect interval is long", func() {
	It("reconnects without waiting the reconnect interval when the first redial succeeds", Label("P4-6.7"), func() {
		env := spectest.Start(GinkgoT(), spectest.WithClientOptions(opcua.ReconnectInterval(reconnectIntervalLong)))
		m := env.Mark()
		env.Relay.Cut()
		Eventually(func(g Gomega) {
			g.Expect(env.StatesSince(m)).To(ContainElement(opcua.Connected),
				"the client did not report Connected within %s of the cut although its first redial succeeds; states after the cut: %v", reconnectWithoutWait, env.StatesSince(m))
		}, reconnectWithoutWait).Should(Succeed())
	})
})

const (
	publishHoldRequestTimeout           = 2 * time.Second
	publishHold                         = 3 * time.Second
	secondSubscribeInterval             = 100 * time.Millisecond
	secondSubscribeLifetimeCount        = 1_000_000
	secondSubscribeKeepAliveCount       = 1000
	publishNotificationBuffer           = 64
	secondSubscribeWait                 = 15 * time.Second
	secondMonitorClientHandle           = 1
	secondSubscriptionValue       int32 = 7100
)

var _ = Describe("when the request timeout is short", func() {
	It("keeps a Publish open past the request timeout after recreating the subscription", Label("P4-6.7"), func() {
		env := spectest.Start(GinkgoT(), spectest.WithClientOptions(opcua.RequestTimeout(publishHoldRequestTimeout)))
		second := env.StartServer()
		env.Relay.RedirectTo(second.Address())
		second.QueueTransferRefusal(ua.StatusBadSubscriptionIDInvalid)
		m := env.Mark()
		env.Relay.Cut()
		env.WaitUntilReconnected()
		created := second.WaitCreatedSubscription(m)
		m2 := env.Mark()
		notifications := make(chan *opcua.PublishNotificationData, publishNotificationBuffer)
		subscribeCtx, subscribeCancel := context.WithTimeout(context.Background(), secondSubscribeWait)
		secondSubscription, subscribeErr := env.Client.Subscribe(subscribeCtx, &opcua.SubscriptionParameters{
			Interval:          secondSubscribeInterval,
			LifetimeCount:     secondSubscribeLifetimeCount,
			MaxKeepAliveCount: secondSubscribeKeepAliveCount,
		}, notifications)
		subscribeCancel()
		Expect(subscribeErr).NotTo(HaveOccurred(), "the client created no second subscription: %v", subscribeErr)
		secondCreated := second.WaitCreatedSubscription(m2)
		node := rules.MonitoredNode(env)
		Expect(node).NotTo(BeNil(), "the recorder saw no CreateMonitoredItems request, so the node the client monitors is unknown")
		monitorCtx, monitorCancel := context.WithTimeout(context.Background(), secondSubscribeWait)
		_, monitorErr := secondSubscription.Monitor(monitorCtx, ua.TimestampsToReturnBoth,
			opcua.NewMonitoredItemCreateRequestWithDefaults(node, ua.AttributeIDValue, secondMonitorClientHandle))
		monitorCancel()
		Expect(monitorErr).NotTo(HaveOccurred(), "the client monitored no node on the second subscription: %v", monitorErr)
		firstHeld := second.WaitHeldPublish()
		firstHeld.Answer(secondCreated, secondSubscriptionValue)
		secondCreateOrder := 0
		for _, record := range env.Recorder.ResponsesSince(m) {
			message, decoded := record.Message()
			if !decoded {
				continue
			}
			if response, isCreate := message.(*ua.CreateSubscriptionResponse); isCreate && response.SubscriptionID == secondSubscription.SubscriptionID {
				secondCreateOrder = record.Order
			}
		}
		Expect(secondCreateOrder).NotTo(BeZero(), "the recorder saw no CreateSubscription response for the second subscription")
		held := second.WaitHeldPublish()
		heldOrder, heldRecorded := held.Order()
		Expect(heldRecorded).To(BeTrue(), "the held Publish has no recorded request")
		Expect(heldOrder).To(BeNumerically(">", secondCreateOrder),
			"the held Publish request was recorded before the second subscription's CreateSubscription response, so its request predates the request timeout the client applies to Publish requests sent after that response")
		time.Sleep(publishHold)
		held.Answer(created, valueAfterReconnect)
		Eventually(func(g Gomega) {
			g.Expect(env.ReceivedSince(m)).To(Equal([]int32{valueAfterReconnect}),
				"the client did not deliver the value answered after holding the Publish for %s; delivered since the mark: %v; errors since the mark: %v", publishHold, env.ReceivedSince(m), env.ReceivedErrorsSince(m))
		}, deliveredAfterHoldWait).Should(Succeed())
		Expect(env.ReceivedErrorsSince(m)).To(BeEmpty(),
			"the client delivered errors since the mark: %v", env.ReceivedErrorsSince(m))
		Expect(second.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", second.UnusedScripts())
	})
})
