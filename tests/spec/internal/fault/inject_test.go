package fault

import (
	"context"
	"strings"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/tests/spec/internal/harness"
	"github.com/gopcua/opcua/ua"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const valueInjected1 int32 = 8501
const valueInjected2 int32 = 8502
const valueInjected3 int32 = 8503
const valueInjectedBase int32 = 8600

func faultNamed(name string) Fault {
	for _, fault := range AllFaults {
		if fault.Name() == name {
			return fault
		}
	}
	Fail("no fault named " + name)
	return nil
}

func readNodeOnce(env *harness.Environment) (*ua.ReadResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return env.Client.Read(ctx, &ua.ReadRequest{
		NodesToRead: []*ua.ReadValueID{{NodeID: ua.NewNumericNodeID(0, 2259), AttributeID: ua.AttributeIDValue}},
	})
}

func answerTwoPublishes(env *harness.Environment, first, second int32) (uint32, harness.Notification, harness.Notification) {
	sub := env.Subscription()
	before := env.Recorder.Notifications()
	lastBefore := before[len(before)-1].SequenceNumber
	env.Server.WaitHeldPublish().Answer(sub, first)
	env.Server.WaitHeldPublish().Answer(sub, second)
	var firstNotification, secondNotification harness.Notification
	Eventually(func(g Gomega) {
		for _, notification := range env.Recorder.Notifications() {
			if notification.Value == first {
				firstNotification = notification
			}
			if notification.Value == second {
				secondNotification = notification
			}
		}
		g.Expect(firstNotification.Value).To(Equal(first))
		g.Expect(secondNotification.Value).To(Equal(second))
	}, 15*time.Second).Should(Succeed())
	return lastBefore, firstNotification, secondNotification
}

var heldPublish harness.HeldPublish

var _ = DescribeTable("Inject arms and Fired observes",
	func(name string, setup func(env *harness.Environment), driver func(env *harness.Environment), settle func(env *harness.Environment)) {
		fault := faultNamed(name)
		opts := []harness.Option{
			harness.WithClientOptions(opcua.RequestTimeout(2 * time.Second)),
			harness.WithPublishingInterval(10 * time.Millisecond),
		}
		opts = append(opts, fault.Options()...)
		env := harness.Start(GinkgoT(), opts...)
		if setup != nil {
			setup(env)
		}

		injected := fault.Inject(env)
		Expect(injected).NotTo(BeNil(), "%s returned no Injected", name)
		Expect(injected.Fired()).To(BeFalse(), "%s fired before its driver ran", name)

		driver(env)

		Eventually(injected.Fired).Within(15*time.Second).Should(BeTrue(), "%s never fired", name)
		if settle != nil {
			settle(env)
		}
	},
	Entry("RequestLost/Read", "RequestLost/Read", nil, func(env *harness.Environment) {
		_, err := readNodeOnce(env)
		Expect(err).To(HaveOccurred(), "the Read succeeded although its request was lost")
	},
		func(env *harness.Environment) { env.WaitUntilReconnected() }),
	Entry("ResponseLost/Read", "ResponseLost/Read", nil, func(env *harness.Environment) {
		_, err := readNodeOnce(env)
		Expect(err).To(HaveOccurred(), "the Read succeeded although its response was lost")
	},
		func(env *harness.Environment) { env.WaitUntilReconnected() }),
	Entry("CutAfterResponse/Read", "CutAfterResponse/Read", nil, func(env *harness.Environment) {
		_, err := readNodeOnce(env)
		Expect(err).NotTo(HaveOccurred(), "the Read failed although the cut delivers its response first")
	},
		func(env *harness.Environment) { env.WaitUntilReconnected() }),
	Entry("DelayBelowTimeout/Read", "DelayBelowTimeout/Read", nil, func(env *harness.Environment) {
		connections := env.Relay.ConnectionCount()
		_, err := readNodeOnce(env)
		Expect(err).NotTo(HaveOccurred(), "the Read failed although its delay stays below the request timeout")
		Expect(env.Relay.ConnectionCount()).To(Equal(connections),
			"the client left its connection although the delay stays below the request timeout")
	}, nil),
	Entry("DelayAboveTimeout/Read", "DelayAboveTimeout/Read", nil, func(env *harness.Environment) {
		go func() { _, _ = readNodeOnce(env) }()
	}, nil),
	Entry("Overload/Publish/Bad_TooManyPublishRequests", "Overload/Publish/Bad_TooManyPublishRequests",
		func(env *harness.Environment) {
			// Grab the client's outstanding Publish before the arming, so
			// the scripted fault takes the Publish the client sends after
			// the driver answers the grabbed one, and not the spontaneous
			// follow-up racing the arming.
			heldPublish = env.Server.WaitHeldPublish()
		},
		func(env *harness.Environment) {
			heldPublish.Answer(env.Subscription(), valueInjected1)
		}, nil),
	Entry("Overload/CreateSubscription/Bad_TooManyOperations", "Overload/CreateSubscription/Bad_TooManyOperations", nil, func(env *harness.Environment) {
		notifications := make(chan *opcua.PublishNotificationData, 64)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, err := env.Client.Subscribe(ctx, &opcua.SubscriptionParameters{
			Interval:          10 * time.Millisecond,
			LifetimeCount:     1_000_000,
			MaxKeepAliveCount: 1000,
		}, notifications)
		Expect(err).To(MatchError(ua.StatusBadTooManyOperations),
			"the Subscribe succeeded although the next answer is scripted Bad_TooManyOperations")
	}, nil),
	Entry("Link/ClosedOnAccept", "Link/ClosedOnAccept", nil, func(env *harness.Environment) {
		env.Relay.Cut()
	},
		func(env *harness.Environment) { env.WaitUntilReconnected() }),
	Entry("Link/ListenerClosed", "Link/ListenerClosed", nil, func(env *harness.Environment) {
		env.Relay.Cut()
	},
		func(env *harness.Environment) { env.WaitUntilReconnected() }),
	Entry("Link/Stall", "Link/Stall", nil, func(env *harness.Environment) {
		go func() { _, _ = readNodeOnce(env) }()
	}, nil),
	Entry("Link/HELUnanswered", "Link/HELUnanswered", nil, func(env *harness.Environment) {
		env.Relay.Cut()
	},
		func(env *harness.Environment) { env.Relay.Cut(); env.WaitUntilReconnected() }),
	Entry("Server/Pause", "Server/Pause", nil, func(env *harness.Environment) {
		_, err := readNodeOnce(env)
		Expect(err).NotTo(HaveOccurred(), "the Read failed although its response is only held one second")
	}, nil),
	Entry("Server/DuplicateSequence", "Server/DuplicateSequence", nil, func(env *harness.Environment) {
		lastBefore, duplicate, after := answerTwoPublishes(env, valueInjected2, valueInjected3)
		Expect(duplicate.SequenceNumber).To(Equal(lastBefore),
			"the first answer after Inject carried sequence number %d, want %d, the last the server sent", duplicate.SequenceNumber, lastBefore)
		Expect(after.SequenceNumber).To(Equal(duplicate.SequenceNumber+1),
			"the second answer carried sequence number %d, want %d, one past the duplicate", after.SequenceNumber, duplicate.SequenceNumber+1)
	}, nil),
	Entry("Server/SkippedSequence", "Server/SkippedSequence", nil, func(env *harness.Environment) {
		lastBefore, skipped, after := answerTwoPublishes(env, valueInjected2, valueInjected3)
		Expect(skipped.SequenceNumber).To(Equal(lastBefore+2),
			"the first answer after Inject carried sequence number %d, want %d, one past a skipped number", skipped.SequenceNumber, lastBefore+2)
		Expect(after.SequenceNumber).To(Equal(skipped.SequenceNumber+1),
			"the second answer carried sequence number %d, want %d, one past the skipped answer", after.SequenceNumber, skipped.SequenceNumber+1)
	}, nil),
	Entry("Consumer/Slow", "Consumer/Slow", nil, func(env *harness.Environment) {
		sub := env.Subscription()
		for i := range 10 {
			env.Server.WaitHeldPublish().Answer(sub, valueInjectedBase+int32(i))
		}
	}, nil),
)

var _ = Describe("Inject arms on the server the client reconnects to", func() {
	It("answers the reconnect's CreateSubscription on the second server with the scripted fault", func() {
		env := harness.Start(GinkgoT(),
			harness.WithClientOptions(opcua.RequestTimeout(2*time.Second)),
			harness.WithPublishingInterval(10*time.Millisecond))
		second := env.StartServer()
		second.QueueTransferRefusal(ua.StatusBadSubscriptionIDInvalid)
		env.Relay.RedirectTo(second.Address())
		Expect(env.UpstreamServer()).To(Equal(second),
			"UpstreamServer did not name the server the redirect points at")
		fault := faultNamed("Overload/CreateSubscription/Bad_TooManyOperations")
		injected := fault.Inject(env)
		Expect(injected).NotTo(BeNil(), "the overload fault returned no Injected")
		env.Relay.Cut()
		env.WaitUntilReconnected()

		ownAddress := strings.TrimPrefix(second.Address(), "opc.tcp://")
		Eventually(func(g Gomega) {
			responses := env.Recorder.Responses()
			saw := false
			for _, record := range env.Recorder.Requests() {
				message, decoded := record.Message()
				if !decoded {
					continue
				}
				if _, isCreate := message.(*ua.CreateSubscriptionRequest); !isCreate {
					continue
				}
				if env.Recorder.UpstreamOf(record.Connection) != ownAddress {
					continue
				}
				for i := range responses {
					answer := responses[i]
					if answer.Connection != record.Connection || answer.RequestID != record.RequestID {
						continue
					}
					answerMessage, answerDecoded := answer.Message()
					if !answerDecoded {
						continue
					}
					if faultMessage, isFault := answerMessage.(*ua.ServiceFault); isFault && faultMessage.ResponseHeader.ServiceResult == ua.StatusBadTooManyOperations {
						saw = true
					}
				}
			}
			g.Expect(saw).To(BeTrue(),
				"the CreateSubscription on the second server was answered with no Bad_TooManyOperations ServiceFault")
		}, 15*time.Second).Should(Succeed())
		Eventually(injected.Fired).Within(15*time.Second).Should(BeTrue(),
			"the fault never fired although the reconnect's CreateSubscription reached the second server")
	})
})

var _ = Describe("Inject over the whole catalogue", func() {
	It("arms every fault with a stable name and no options but the consumer's", func() {
		env := harness.Start(GinkgoT(),
			harness.WithClientOptions(opcua.RequestTimeout(2*time.Second)),
			harness.WithPublishingInterval(10*time.Millisecond))
		for _, fault := range AllFaults {
			injected := fault.Inject(env)
			Expect(injected).NotTo(BeNil(), "%s returned no Injected", fault.Name())
			name := fault.Name()
			Expect(fault.Name()).To(Equal(name), "%s does not keep its name", name)
			if name == "Consumer/Slow" {
				Expect(fault.Options()).To(HaveLen(2),
					"Consumer/Slow needs its notification-buffer and slow-consumer options")
				continue
			}
			Expect(fault.Options()).To(BeNil(), "%s needs no Start options", name)
		}
	})
})
