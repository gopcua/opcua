package part4

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/tests/spec/internal/fault"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/tests/spec/internal/matrix"
	"github.com/gopcua/opcua/tests/spec/internal/message"
	"github.com/gopcua/opcua/tests/spec/internal/rules"
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
	var env *harness.Environment
	BeforeEach(func() { env = harness.New(GinkgoT()) })

	Context("when the session survives a transport loss", func() {
		var sub harness.Subscription
		var last uint32
		var m harness.Mark
		BeforeEach(func() {
			sub = env.Subscription()
			last = env.LastSequenceNumber()
			sub.Retain(last+1, valueRetained)
			m = env.Mark()
			env.Relay.Cut()
			env.WaitUntilReconnected()
		})

		It("reactivates the existing session instead of creating one", Label("P4-6.7"), func() {
			ctx := matrix.Context{Env: env, Mark: m}
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
			rules.RepublishesFromNextSequence.Check(matrix.Context{Env: env, Mark: m, LastSeq: last})
			Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
		})

		It("sends no Publish until Republish has answered Bad_MessageNotAvailable", Label("P4-6.7", "issue-879", "known-defect"), func() {
			rules.SendsNoPublishBeforeNotAvailable.Check(matrix.Context{Env: env, Mark: m})
			Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
		})

		It("sends no TransferSubscriptions for a subscription its own session owns", Label("P4-6.7", "P4-5.14.7.4", "known-defect"), func() {
			rules.SendsNoTransferForOwnSubscription.Check(matrix.Context{Env: env, Mark: m})
			Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
		})

		It("keeps the subscription id it had before the cut", Label("P4-6.7", "issue-879", "known-defect"), func() {
			rules.KeepsSubscriptionID.Check(matrix.Context{Env: env, Mark: m, Sub: sub})
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
		var sub harness.Subscription
		var last uint32
		var m harness.Mark
		BeforeEach(func() {
			sub = env.Subscription()
			last = env.LastSequenceNumber()
			sub.FailRepublish(last+1, ua.StatusBadSubscriptionIDInvalid)
			m = env.Mark()
			env.Relay.Cut()
			env.WaitUntilReconnected()
		})

		It("creates a new subscription", Label("P4-6.7", "should", "known-defect"), func() {
			rules.RecreatesAfterRefusal.Check(matrix.Context{Env: env, Mark: m, LastSeq: last})
			Expect(env.Server.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", env.Server.UnusedScripts())
		})

		It("resumes publishing with the new subscription", Label("P4-6.7", "issue-895", "known-defect"), MustPassRepeatedly(10), func() {
			rules.RecreatesAfterRefusal.Check(matrix.Context{Env: env, Mark: m, LastSeq: last})
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
		var sub harness.Subscription
		var last uint32
		var m harness.Mark
		BeforeEach(func() {
			sub = env.Subscription()
			last = env.LastSequenceNumber()
			sub.Retain(last+1, valueRetained)
		})

		DescribeTable("the second cut during Republish recovery",
			func(moment harness.Moment, check func(env *harness.Environment, m harness.Mark, recovery int)) {
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
			Entry("the Republish request is lost (`BeforeRequestReachesServer`)", harness.BeforeRequestReachesServer, func(env *harness.Environment, m harness.Mark, recovery int) {
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
			Entry("the connection drops right after the Republish response is delivered (`AfterResponseReachesClient`)", harness.AfterResponseReachesClient, func(env *harness.Environment, m harness.Mark, recovery int) {
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
		var second *harness.ScriptedServer
		var last uint32
		var m harness.Mark
		BeforeEach(func() {
			second = env.StartServer()
			env.Relay.RedirectTo(second.Address())
			last = env.LastSequenceNumber()
		})

		It("creates a new session only after ActivateSession fails", Label("P4-6.7"), func() {
			m = env.Mark()
			env.Relay.Cut()
			env.WaitUntilReconnected()
			rules.CreatesSessionOnlyAfterActivateFailed.Check(matrix.Context{Env: env, Mark: m})
		})

		DescribeTable("transfers, and creates new subscriptions when the transfer fails",
			func(arm func(second *harness.ScriptedServer), flow func(env *harness.Environment, second *harness.ScriptedServer, m harness.Mark, last uint32)) {
				arm(second)
				m = env.Mark()
				env.Relay.Cut()
				env.WaitUntilReconnected()
				flow(env, second, m, last)
			},
			Entry("refused per subscription with Bad_SubscriptionIdInvalid",
				func(second *harness.ScriptedServer) { second.QueueTransferRefusal(ua.StatusBadSubscriptionIDInvalid) },
				func(env *harness.Environment, second *harness.ScriptedServer, m harness.Mark, last uint32) {
					transferFailedFlow(env, second, m, transferRefusedPerResult(ua.StatusBadSubscriptionIDInvalid), requireRecreatedCarriesFirstSubscriptionParameters)
				},
				Label("P4-6.7", "P4-5.14.7", "should")),
			Entry("unsupported, with Bad_ServiceUnsupported",
				func(second *harness.ScriptedServer) {},
				func(env *harness.Environment, second *harness.ScriptedServer, m harness.Mark, last uint32) {
					transferFailedFlow(env, second, m, transferAnsweredWithStatus(ua.StatusBadServiceUnsupported), requireRecreatedCarriesFirstSubscriptionParameters)
				},
				Label("P4-6.7", "should")),
			Entry("refused per subscription with Bad_UserAccessDenied",
				func(second *harness.ScriptedServer) { second.QueueTransferRefusal(ua.StatusBadUserAccessDenied) },
				func(env *harness.Environment, second *harness.ScriptedServer, m harness.Mark, last uint32) {
					transferFailedFlow(env, second, m, transferRefusedPerResult(ua.StatusBadUserAccessDenied), requireNoRepublishBetween)
				},
				Label("P4-6.7", "P4-7.38.1", "known-defect")),
			Entry("transferred, with the last delivered notification still available",
				func(second *harness.ScriptedServer) {
					moved := second.QueueTransferSuccess(last, last+1)
					moved.Retain(last+1, valueRetained)
				},
				func(env *harness.Environment, second *harness.ScriptedServer, m harness.Mark, last uint32) {
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
			rules.RepublishesRecreatedFromOne.Check(matrix.Context{Env: env, Mark: m, Recreated: created})
			Expect(second.UnusedScripts()).To(BeEmpty(), "scripts this spec armed were never used: %v", second.UnusedScripts())
		})
	})
})

const dataRaceWindow = 3 * time.Second

// The spec asserts nothing itself: under -race, the race detector fails
// the test binary if Close and Dial race on the client's connection again.
var _ = Describe("when the client is closed while it re-dials", func() {
	It("does not race Close against the reconnect Dial", Label("P4-6.7", "issue-883"), func() {
		env := harness.New(GinkgoT())
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

func requireRecreatedCarriesFirstSubscriptionParameters(env *harness.Environment, m harness.Mark, _, createAnswer harness.ServiceRecord[ua.Response]) {
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
		env := harness.New(GinkgoT(), harness.WithClientOptions(opcua.ReconnectInterval(reconnectIntervalLong)))
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
		env := harness.New(GinkgoT(), harness.WithClientOptions(opcua.RequestTimeout(publishHoldRequestTimeout)))
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

// The §6.7 failure matrix: scenarios whose prepare step and
// transport loss decide what the client must re-establish after the
// relay cuts — the session, the session on a second server that
// refuses the transfer, or the subscriptions on a server that forgot
// them. reestablishingRules decides which rule Its a fault registers.
var _ = Describe("P4-6.7", func() {
	DescribeTableSubtree(sessionSurvives.Name, func(f fault.Fault) {
		obs := matrix.Run(sessionSurvives, f)
		obs.BeforeCloseInvariants()
		if obs.Applies(rules.ReactivatesSession) {
			It("ReactivatesSession", obs.Labels("ReactivatesSession"), func() {
				rules.ReactivatesSession.Check(obs.Context())
			})
		}
		if obs.Applies(rules.CreatesNoSession) {
			It("CreatesNoSession", obs.Labels("CreatesNoSession"), func() {
				rules.CreatesNoSession.Check(obs.Context())
			})
		}
		if obs.Applies(rules.RepublishesFromNextSequence) {
			It("RepublishesFromNextSequence", obs.Labels("RepublishesFromNextSequence"), func() {
				rules.RepublishesFromNextSequence.Check(obs.Context())
			})
		}
		if obs.Applies(rules.SendsNoPublishBeforeNotAvailable) {
			It("SendsNoPublishBeforeNotAvailable", obs.Labels("SendsNoPublishBeforeNotAvailable"), func() {
				rules.SendsNoPublishBeforeNotAvailable.Check(obs.Context())
			})
		}
		if obs.Applies(rules.KeepsSubscriptionID) {
			It("KeepsSubscriptionID", obs.Labels("KeepsSubscriptionID"), func() {
				rules.KeepsSubscriptionID.Check(obs.Context())
			})
		}
		if obs.Applies(rules.SendsNoTransferForOwnSubscription) {
			It("SendsNoTransferForOwnSubscription", obs.Labels("SendsNoTransferForOwnSubscription"), func() {
				rules.SendsNoTransferForOwnSubscription.Check(obs.Context())
			})
		}
		if obs.Applies(rules.CreatesSessionAfterActivateTimedOut) {
			It("CreatesSessionAfterActivateTimedOut", obs.Labels("CreatesSessionAfterActivateTimedOut"), func() {
				rules.CreatesSessionAfterActivateTimedOut.Check(obs.Context())
			})
		}
		obs.AfterCloseInvariants()
	}, matrix.Entries(sessionSurvives, fault.AllFaults...))

	DescribeTableSubtree(sessionLost.Name, func(f fault.Fault) {
		obs := matrix.Run(sessionLost, f)
		obs.BeforeCloseInvariants()
		if obs.Applies(rules.CreatesSessionOnlyAfterActivateFailed) {
			It("CreatesSessionOnlyAfterActivateFailed", obs.Labels("CreatesSessionOnlyAfterActivateFailed"), func() {
				rules.CreatesSessionOnlyAfterActivateFailed.Check(obs.Context())
			})
		}
		if obs.Applies(rules.RecreatesAfterRefusal) {
			It("RecreatesAfterRefusal", obs.Labels("RecreatesAfterRefusal"), func() {
				rules.RecreatesAfterRefusal.Check(obs.Context())
			})
		}
		if obs.Applies(rules.CreatesSessionAfterActivateTimedOut) {
			It("CreatesSessionAfterActivateTimedOut", obs.Labels("CreatesSessionAfterActivateTimedOut"), func() {
				rules.CreatesSessionAfterActivateTimedOut.Check(obs.Context())
			})
		}
		obs.AfterCloseInvariants()
	}, matrix.Entries(sessionLost, fault.AllFaults...))

	DescribeTableSubtree(subscriptionsLost.Name, func(f fault.Fault) {
		obs := matrix.Run(subscriptionsLost, f)
		obs.BeforeCloseInvariants()
		if obs.Applies(rules.RecreatesAfterRefusal) {
			It("RecreatesAfterRefusal", obs.Labels("RecreatesAfterRefusal"), func() {
				rules.RecreatesAfterRefusal.Check(obs.Context())
			})
		}
		if obs.Applies(rules.RepublishesRecreatedFromOne) {
			It("RepublishesRecreatedFromOne", obs.Labels("RepublishesRecreatedFromOne"), func() {
				rules.RepublishesRecreatedFromOne.Check(obs.Context())
			})
		}
		if obs.Applies(rules.CreatesSessionAfterActivateTimedOut) {
			It("CreatesSessionAfterActivateTimedOut", obs.Labels("CreatesSessionAfterActivateTimedOut"), func() {
				rules.CreatesSessionAfterActivateTimedOut.Check(obs.Context())
			})
		}
		obs.AfterCloseInvariants()
	}, matrix.Entries(subscriptionsLost, fault.AllFaults...))
})

// The unfiled issues the triage named, one label per group of cases
// with one root cause.
var (
	issueFailedSubscriptionStep      = matrix.Unfiled("failed-subscription-step", "a failed subscription step during reconnect makes recreateSession drop a healthy session without closing it")
	issueConnectionFailureOnActivate = matrix.Unfiled("connection-failure-on-activate", "a connection failure during ActivateSession makes the client forget its session without retrying or closing it")
	issueTimedOutActivationSession   = matrix.Unfiled("timed-out-activation-session", "after an ActivateSession timeout the old session stays open on the server")
	issueDrainedConnectionError      = matrix.Unfiled("drained-connection-error", "the reconnect loop's error drain discards a connection error, so the client reports Connected on a dead channel")
	issueHELHandshakeHang            = matrix.Unfiled("hel-handshake-hang", "the HEL/ACK handshake ignores its context, so a reconnect hangs on an unanswered HEL; PR #919 fixes it")
)

// recreatePathFaults lists the faults under which the client does not
// reactivate its session but creates a new one. On that path it
// recreates its subscriptions and resumes publishing, so the delivery
// and publishing entries below, which belong to the reactivation path,
// do not cover them.
var recreatePathFaults = []string{
	"CutAfterResponse/OpenSecureChannel", "DelayAboveTimeout/ActivateSession",
	"RequestLost/ActivateSession", "ResponseLost/ActivateSession"}

// cutPublishFaults lists the faults that cut the connection on the
// Publish service: the workload answers one held Publish right after
// the arm on them, so their armed cut fires on that exchange and their
// labelled HaveFired and ResumePublishing checks pass on main.
var cutPublishFaults = []string{
	"CutAfterResponse/Publish", "RequestLost/Publish", "ResponseLost/Publish"}

// failedSubscriptionStepFaults lists the SessionLost faults that fail
// a subscription step of the reconnect.
var failedSubscriptionStepFaults = []string{
	"CutAfterResponse/CreateSubscription", "CutAfterResponse/Read", "CutAfterResponse/TransferSubscriptions",
	"DelayAboveTimeout/CreateMonitoredItems", "DelayAboveTimeout/CreateSubscription",
	"Overload/CreateMonitoredItems/Bad_ResourceUnavailable", "Overload/CreateMonitoredItems/Bad_TooManyOperations",
	"Overload/CreateSubscription/Bad_ResourceUnavailable", "Overload/CreateSubscription/Bad_TooManyOperations",
	"RequestLost/CreateMonitoredItems", "RequestLost/CreateSubscription", "RequestLost/TransferSubscriptions",
	"ResponseLost/CreateMonitoredItems", "ResponseLost/CreateSubscription", "ResponseLost/TransferSubscriptions"}

// failedActivationFaults lists the faults that fail the ActivateSession
// of a reconnect on its connection.
var failedActivationFaults = []string{
	"CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession", "ResponseLost/ActivateSession"}

// sessionSurvives leaves the session and its subscription on the one
// server the client is connected to: the client must re-activate the
// session and republish what it missed.
var sessionSurvives = matrix.Scenario{
	Clause:  "P4-6.7",
	Name:    "SessionSurvives",
	Ordinal: 1,
	Sends: []message.Message{
		message.HEL, message.OpenSecureChannel, message.ActivateSession, message.Read,
		message.Republish, message.Publish, message.CloseSession, message.CloseSecureChannel,
	},
	Options:  reestablishingOptions,
	Workload: reestablishing{prepare: prepareSessionSurvives}.workload,
	Rules: reestablishingRules(
		rules.ReactivatesSession,
		rules.CreatesNoSession,
		rules.RepublishesFromNextSequence,
		rules.SendsNoPublishBeforeNotAvailable,
		rules.KeepsSubscriptionID,
		rules.SendsNoTransferForOwnSubscription,
	),
	Invariants: subscriptionInvariants,
	KnownDefects: []matrix.KnownDefect{
		{Issue: "issue-879", Check: "RepublishesFromNextSequence", Applies: matrix.EveryFault},
		{Issue: "issue-879", Check: "SendsNoPublishBeforeNotAvailable", Applies: matrix.EveryFault},
		{Issue: "issue-879", Check: "KeepsSubscriptionID", Applies: matrix.EveryFault},
		{Issue: "issue-879", Check: "SendsNoTransferForOwnSubscription", Applies: matrix.EveryFault},
		{Issue: "issue-879", Check: "DeliverEachValueOnce", Applies: matrix.EveryFaultExcept(recreatePathFaults...)},
		{Issue: "issue-879", Check: "ResumePublishing", Applies: matrix.EveryFaultExcept(recreatePathFaults...)},
		{Issue: "issue-879", Check: "HaveFired", Applies: matrix.FaultsTargetingExcept(cutPublishFaults, "Publish", "Republish")},
		{Issue: issueConnectionFailureOnActivate, Check: "CloseEveryKnownSession", Applies: matrix.FaultsNamed(failedActivationFaults...)},
		{Issue: issueConnectionFailureOnActivate, Check: "KeepOneSessionOpen", Applies: matrix.FaultsNamed(failedActivationFaults...)},
		{Issue: issueConnectionFailureOnActivate, Check: "CreatesNoSession", Applies: matrix.FaultsNamed(failedActivationFaults...)},
		{Issue: issueConnectionFailureOnActivate, Check: "ReactivatesSession", Applies: matrix.FaultsNamed("CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession")},
		{Issue: issueHELHandshakeHang, Check: "CloseEveryKnownSession", Applies: matrix.FaultsNamed("Link/HELUnanswered")},
		{Issue: issueHELHandshakeHang, Check: "CreatesNoSession", Applies: matrix.FaultsNamed("Link/HELUnanswered")},
		{Issue: issueHELHandshakeHang, Check: "ReactivatesSession", Applies: matrix.FaultsNamed("Link/HELUnanswered")},
		{Issue: "issue-828", Check: "CreatesNoSession", Applies: matrix.FaultsNamed("Link/Stall")},
		{Issue: "issue-828", Check: "ReactivatesSession", Applies: matrix.FaultsNamed("Link/Stall")},
		{Issue: issueTimedOutActivationSession, Check: "CloseEveryKnownSession", Applies: matrix.FaultsNamed("DelayAboveTimeout/ActivateSession")},
		{Issue: issueTimedOutActivationSession, Check: "KeepOneSessionOpen", Applies: matrix.FaultsNamed("DelayAboveTimeout/ActivateSession")},
		{Issue: issueDrainedConnectionError, Check: "CloseEveryKnownSession", Applies: matrix.FaultsNamed("CutAfterResponse/Read")},
	},
}

// sessionLost redirects the client to a second server that refuses
// the transfer: the client must create a new session and recreate its
// subscription there.
var sessionLost = matrix.Scenario{
	Clause:  "P4-6.7",
	Name:    "SessionLost",
	Ordinal: 2,
	Sends: []message.Message{
		message.HEL, message.OpenSecureChannel, message.ActivateSession, message.CreateSession, message.Read,
		message.TransferSubscriptions, message.CreateSubscription, message.CreateMonitoredItems,
		message.Publish, message.CloseSession, message.CloseSecureChannel,
	},
	Options:  reestablishingOptions,
	Workload: reestablishing{prepare: prepareSessionLost, prepareBeforeArm: true}.workload,
	Rules: reestablishingRules(
		rules.CreatesSessionOnlyAfterActivateFailed,
		rules.RecreatesAfterRefusal,
	),
	Invariants: subscriptionInvariants,
	KnownDefects: []matrix.KnownDefect{
		{Issue: "issue-879", Check: "ResumePublishing", Applies: matrix.FaultsNamed(
			"DelayAboveTimeout/Read",
			"Overload/Publish/Bad_ResourceUnavailable", "Overload/Publish/Bad_TooManyOperations",
			"Overload/Publish/Bad_TooManyPublishRequests", "RequestLost/Read", "ResponseLost/Read")},
		{Issue: "issue-879", Check: "KeepOneSubscriptionPerClientSubscription", Applies: matrix.FaultsNamed(
			"DelayAboveTimeout/Read", "RequestLost/Read", "ResponseLost/Read")},
		{Issue: "issue-879", Check: "RecreatesAfterRefusal", Applies: matrix.FaultsNamed(
			"RequestLost/Read", "ResponseLost/Read")},
		{Issue: issueFailedSubscriptionStep, Check: "CloseEveryKnownSession", Applies: matrix.FaultsNamed(failedSubscriptionStepFaults...)},
		{Issue: issueFailedSubscriptionStep, Check: "KeepOneSessionOpen", Applies: matrix.FaultsNamed(failedSubscriptionStepFaults...)},
		{Issue: issueFailedSubscriptionStep, Check: "RecreatesAfterRefusal", Applies: matrix.FaultsNamed("CutAfterResponse/CreateSubscription")},
		{Issue: issueConnectionFailureOnActivate, Check: "CloseEveryKnownSession", Applies: matrix.FaultsNamed("CutAfterResponse/CreateSession")},
		{Issue: issueConnectionFailureOnActivate, Check: "KeepOneSessionOpen", Applies: matrix.FaultsNamed("CutAfterResponse/CreateSession")},
		{Issue: issueConnectionFailureOnActivate, Check: "CreatesSessionOnlyAfterActivateFailed", Applies: matrix.FaultsNamed("CutAfterResponse/OpenSecureChannel", "RequestLost/ActivateSession")},
		{Issue: issueHELHandshakeHang, Check: "CreatesSessionOnlyAfterActivateFailed", Applies: matrix.FaultsNamed("Link/HELUnanswered")},
		{Issue: issueHELHandshakeHang, Check: "KeepOneSessionOpen", Applies: matrix.FaultsNamed("Link/HELUnanswered")},
		{Issue: issueHELHandshakeHang, Check: "KeepOneSubscriptionPerClientSubscription", Applies: matrix.FaultsNamed("Link/HELUnanswered")},
		{Issue: issueHELHandshakeHang, Check: "RecreatesAfterRefusal", Applies: matrix.FaultsNamed("Link/HELUnanswered")},
		{Issue: issueHELHandshakeHang, Check: "ResumePublishing", Applies: matrix.FaultsNamed("Link/HELUnanswered")},
		{Issue: "issue-828", Check: "CreatesSessionOnlyAfterActivateFailed", Applies: matrix.FaultsNamed("Link/Stall")},
		{Issue: "issue-828", Check: "DeliverEachValueOnce", Applies: matrix.FaultsNamed("Link/Stall")},
		{Issue: "issue-828", Check: "RecreatesAfterRefusal", Applies: matrix.FaultsNamed("Link/Stall")},
		{Issue: "issue-828", Check: "ResumePublishing", Applies: matrix.FaultsNamed("Link/Stall")},
		{Issue: issueDrainedConnectionError, Check: "CloseEveryKnownSession", Applies: matrix.FaultsNamed("CutAfterResponse/CreateMonitoredItems")},
		{Issue: issueDrainedConnectionError, Check: "ResumePublishing", Applies: matrix.FaultsNamed("CutAfterResponse/CreateMonitoredItems")},
	},
}

// subscriptionsLost makes the server forget every subscription while
// the session survives: the client must recreate its subscription.
var subscriptionsLost = matrix.Scenario{
	Clause:  "P4-6.7",
	Name:    "SubscriptionsLost",
	Ordinal: 3,
	Sends: []message.Message{
		message.HEL, message.OpenSecureChannel, message.ActivateSession, message.Read,
		message.Republish, message.CreateSubscription, message.CreateMonitoredItems,
		message.Publish, message.CloseSession, message.CloseSecureChannel,
	},
	Options:  reestablishingOptions,
	Workload: reestablishing{prepare: prepareSubscriptionsLost}.workload,
	Rules: reestablishingRules(
		rules.RecreatesAfterRefusal,
		rules.RepublishesRecreatedFromOne,
	),
	Invariants: subscriptionInvariants,
	KnownDefects: []matrix.KnownDefect{
		{Issue: "issue-879", Check: "RecreatesAfterRefusal", Applies: matrix.EveryFault},
		{Issue: "issue-879", Check: "RepublishesRecreatedFromOne", Applies: matrix.EveryFault},
		{Issue: "issue-879", Check: "HaveFired", Applies: matrix.FaultsTargetingExcept(cutPublishFaults, "CreateSubscription", "CreateMonitoredItems", "Publish", "Republish")},
		{Issue: "issue-895", Check: "ResumePublishing", Applies: matrix.EveryFaultExcept(recreatePathFaults...)},
		{Issue: "issue-895", Check: "KeepOneSubscriptionPerClientSubscription", Applies: matrix.EveryFaultExcept(recreatePathFaults...)},
		{Issue: issueConnectionFailureOnActivate, Check: "CloseEveryKnownSession", Applies: matrix.FaultsNamed(failedActivationFaults...)},
		{Issue: issueConnectionFailureOnActivate, Check: "KeepOneSessionOpen", Applies: matrix.FaultsNamed(failedActivationFaults...)},
		{Issue: issueHELHandshakeHang, Check: "CloseEveryKnownSession", Applies: matrix.FaultsNamed("Link/HELUnanswered")},
		{Issue: issueTimedOutActivationSession, Check: "CloseEveryKnownSession", Applies: matrix.FaultsNamed("DelayAboveTimeout/ActivateSession")},
		{Issue: issueTimedOutActivationSession, Check: "KeepOneSessionOpen", Applies: matrix.FaultsNamed("DelayAboveTimeout/ActivateSession")},
		{Issue: issueDrainedConnectionError, Check: "CloseEveryKnownSession", Applies: matrix.FaultsNamed("CutAfterResponse/Read")},
	},
}

// reestablishingRules returns the rules §6.7 prescribes per fault: an
// ActivateSession that timed out must be followed by a new session, a
// DelayAboveTimeout or Overload fault on any other service leaves the
// invariants only, and every other fault the scenario's own rules.
func reestablishingRules(own ...matrix.Rule) func(fault.Fault) []matrix.Rule {
	return func(f fault.Fault) []matrix.Rule {
		switch {
		case f.Name() == "DelayAboveTimeout/ActivateSession":
			return []matrix.Rule{rules.CreatesSessionAfterActivateTimedOut}
		case strings.HasPrefix(f.Name(), "DelayAboveTimeout/"):
			return nil
		case strings.HasPrefix(f.Name(), "Overload/"):
			return nil
		}
		return own
	}
}

// target06_07 is what a scenario's prepare step leaves the workload
// with: the server the client must end on, whether it must recreate
// its subscription there, and how the scripted transfer answer
// refuses when the prepare queued one.
type target06_07 struct {
	server          *harness.ScriptedServer
	recreate        bool
	transferRefusal func(harness.ServiceRecord[ua.Response]) bool
}

// reestablishing is the workload the §6.7 scenarios share: the
// prepare step that stages the server side of the transport loss and
// names the target the client must recover on. prepareBeforeArm says
// the prepare step must run before the arm point — the redirect the
// server-side faults arm behind — instead of after the consumer's
// burst.
type reestablishing struct {
	prepare          func(env *harness.Environment) target06_07
	prepareBeforeArm bool
}

func reestablishingOptions(f fault.Fault, block int32) []harness.Option {
	return []harness.Option{
		harness.WithRetentionQueue(),
		harness.WithPublishingInterval(10 * time.Millisecond),
		harness.WithClientOptions(opcua.RequestTimeout(2 * time.Second)),
		harness.WithFirstValue(reestablishingValuesOf(block).first),
	}
}

// workload drives one §6.7 case. The workload answers v3 and the sentinel
// on the subscription the client publishes with: the one the recreate
// scenarios make it create, or — when the scenario keeps the session —
// whatever live subscription the client holds, because a client that
// recreates its subscription there violates KeepsSubscriptionID and
// the case must still reach its checks. The prepare step runs before
// the arm point when the scenario's faults must arm behind its
// redirect, and after the consumer's burst otherwise — SubscriptionsLost
// forgets the old subscription only after the burst answered on it.
func (s reestablishing) workload(env *harness.Environment, f fault.Fault, block int32) matrix.Outcome {
	values := reestablishingValuesOf(block)
	sub := env.Subscription()
	if held, ok := env.Server.TryWaitHeldPublish(15 * time.Second); ok {
		held.Answer(sub, values.v1)
		waitReceived(env, values.v1)
	}
	last := env.LastSequenceNumber()
	sub.Retain(last+1, values.v2)
	var t target06_07
	if s.prepareBeforeArm {
		t = s.prepare(env)
	}
	m := env.Mark()
	injected := f.Inject(env)
	// The arm exchange: a Publish-targeting fault gets a held Publish
	// answered right after the arm, so its armed cut has a response to
	// fire on that exists only because the workload armed first — not
	// the answer of an exchange that raced the arm.
	if targetsPublish(f) {
		if held, ok := env.Server.TryWaitHeldPublish(15 * time.Second); ok {
			held.Answer(sub, values.vArm)
		}
	}
	for i := range consumerBurst(f) {
		if held, ok := env.Server.TryWaitHeldPublish(15 * time.Second); ok {
			held.Answer(sub, values.burst[i])
		}
	}
	if !s.prepareBeforeArm {
		t = s.prepare(env)
	}
	if breaksTransport(f) {
		env.Relay.Cut()
	}
	env.TryWaitUntilReconnected(30 * time.Second)
	faultEnd := time.Now()

	answering := sub
	recreated := harness.Subscription{}
	canAnswer := true
	if t.recreate {
		created, ok := t.server.TryWaitCreatedSubscription(m, 15*time.Second)
		if ok {
			answering = created
			recreated = created
		} else {
			canAnswer = false
		}
	} else if created, ok := t.server.SubscriptionCreatedSince(m); ok {
		answering = created
	}
	sentinel := int32(0)
	var answeredAt time.Time
	if canAnswer {
		if held, ok := t.server.TryWaitHeldPublish(15 * time.Second); ok {
			held.Answer(answering, values.v3)
			if next, ok := t.server.TryWaitHeldPublish(15 * time.Second); ok {
				next.Answer(answering, values.sentinel)
				sentinel = values.sentinel
				answeredAt = time.Now()
			}
		}
	}
	return matrix.Outcome{
		Injected:   injected,
		FaultEnd:   faultEnd,
		Sentinel:   sentinel,
		AnsweredAt: answeredAt,
		Rules: matrix.Context{
			Env:             env,
			Mark:            m,
			LastSeq:         last,
			Sub:             sub,
			Recreated:       recreated,
			Server:          t.server,
			TransferRefusal: t.transferRefusal,
		},
	}
}

// prepareSessionSurvives leaves the session and its subscription on
// the one server the client is connected to; the cut after the arm
// point is the transport loss.
func prepareSessionSurvives(env *harness.Environment) target06_07 {
	return target06_07{server: env.Server}
}

// prepareSessionLost starts a second server that inherits the
// retention queue, refuses the next transfer with
// Bad_SubscriptionIdInvalid and sends the client's reconnects to it,
// all before the arm point so server-side faults arm there; the cut
// after the arm point is the transport loss, and the client must
// recreate its session and its subscription on the second server.
func prepareSessionLost(env *harness.Environment) target06_07 {
	second := env.StartServer()
	second.QueueTransferRefusal(ua.StatusBadSubscriptionIDInvalid)
	env.Relay.RedirectTo(second.Address())
	return target06_07{server: second, recreate: true, transferRefusal: refusedBadSubscriptionIDInvalid}
}

// prepareSubscriptionsLost marks every subscription on the server
// deleted: the session survives, but the server answers no
// subscription the client holds; the cut after the arm point is the
// transport loss.
func prepareSubscriptionsLost(env *harness.Environment) target06_07 {
	env.Server.ForgetSubscriptions()
	return target06_07{server: env.Server, recreate: true}
}

// breaksTransport says whether the workload cuts the relay after the
// fault is armed: a stalled link is its own transport loss — the link
// goes silent instead of closing — and so is a Publish fault that cuts
// the connection itself, because the arm exchange answers a held
// Publish whose request or response fires the armed cut.
func breaksTransport(f fault.Fault) bool {
	if f.Name() == "Link/Stall" {
		return false
	}
	if targetsPublish(f) {
		switch strings.Split(f.Name(), "/")[0] {
		case "RequestLost", "ResponseLost", "CutAfterResponse":
			return false
		}
	}
	return true
}

// waitReceived waits until the client has delivered v: a fault armed
// while the relay still holds the answer the client has not seen can
// fire on that answer's own response, and whether it does is a race
// between the relay's pump and the arm.
func waitReceived(env *harness.Environment, v int32) {
	deadline := time.Now().Add(15 * time.Second)
	for !slices.Contains(env.Received(), v) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
}

// consumerBurst says how many values the workload answers in a row
// right after the arm point: the slow consumer's channel, four deep,
// fills only when a burst outruns its 200 ms drain.
func consumerBurst(f fault.Fault) int {
	if f.Name() == "Consumer/Slow" {
		return 8
	}
	return 0
}

// refusedBadSubscriptionIDInvalid recognizes the transfer answer the
// SessionLost base action scripts: one result, Bad_SubscriptionIdInvalid.
func refusedBadSubscriptionIDInvalid(answer harness.ServiceRecord[ua.Response]) bool {
	message, decoded := answer.Message()
	if !decoded {
		return false
	}
	response, isTransfer := message.(*ua.TransferSubscriptionsResponse)
	return isTransfer && len(response.Results) == 1 && response.Results[0].StatusCode == ua.StatusBadSubscriptionIDInvalid
}

// reestablishingValues are the values one case answers, unique per case: every
// value, the first one New answers included, derives from the case's
// own block — 1000 times the case's ordinal among every scenario ×
// fault pair plus a per-value offset — so a received value matches the
// notification that carried it by value and never a value another case
// answered. The burst is the eight values the slow consumer's case
// answers right after arming. vArm is answered on the held Publish
// right after the arm point, so a Publish-targeting fault has an
// exchange to fire on that exists only because the workload armed.
type reestablishingValues struct {
	first    int32
	v1       int32
	v2       int32
	v3       int32
	vArm     int32
	sentinel int32
	burst    [8]int32
}

// reestablishingValuesOf returns the value block of one case, derived
// from the case's block alone so the New options and the workload read
// the same values.
func reestablishingValuesOf(base int32) reestablishingValues {
	values := reestablishingValues{first: base + 1, v1: base + 2, v2: base + 3, v3: base + 4, vArm: base + 5, sentinel: base + 999}
	for i := range values.burst {
		values.burst[i] = base + int32(11+i)
	}
	return values
}
