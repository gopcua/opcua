package spectest

import (
	"context"
	"fmt"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/tests/spec/message"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const valueOverloaded int32 = 8201

func readPairSince(env *Environment, m Mark) (ServiceRecord[ua.Request], ServiceRecord[ua.Response], bool) {
	for _, request := range env.Recorder.RequestsSince(m) {
		decoded, ok := request.Message()
		if !ok {
			continue
		}
		if _, isRead := decoded.(*ua.ReadRequest); !isRead {
			continue
		}
		answer := responseOf(env.Recorder.ResponsesSince(m), request.Connection, request.RequestID)
		if answer == nil {
			continue
		}
		return request, *answer, true
	}
	return ServiceRecord[ua.Request]{}, ServiceRecord[ua.Response]{}, false
}

var _ = Describe("ScriptedServer sessions and services", func() {
	It("counts one known session after Start and none after the client closes", func() {
		env := Start(GinkgoT())
		Expect(env.Server.KnownSessions()).To(Equal(1),
			"a client with one open session leaves the server counting %d known sessions, want 1", env.Server.KnownSessions())

		ctx, cancel := context.WithTimeout(context.Background(), specWait)
		defer cancel()
		Expect(env.Client.Close(ctx)).To(Succeed(), "closing the client failed")

		Eventually(func(g Gomega) {
			g.Expect(env.Server.KnownSessions()).To(Equal(0),
				"the session stayed known after the client closed it")
		}, specWait).Should(Succeed())
	})

	It("does not count a session whose CreateSession response the relay dropped", func() {
		env := Start(GinkgoT())
		second := env.StartServer()
		env.Relay.RedirectTo(second.Address())
		env.Relay.CutAt(ResponseNeverReachesClient, message.CreateSession)

		env.Relay.Cut()
		env.WaitUntilReconnected()

		Eventually(func(g Gomega) {
			open, known := second.sessionCounts()
			g.Expect(open).To(BeNumerically(">=", 1),
				"the redirected client recreated no session on the second server")
			g.Expect(open).To(Equal(known+1),
				"the server counts %d open and %d known sessions, want exactly the dropped-response session between them", open, known)
		}, specWait).Should(Succeed())
	})

	It("ignores a CloseSession for a token it did not create", func() {
		env := Start(GinkgoT())
		Expect(env.Server.KnownSessions()).To(Equal(1),
			"a client with one open session leaves the server counting %d known sessions, want 1", env.Server.KnownSessions())

		// The client injects its session's token into every request it
		// sends, so an untracked token can only arrive as recorded
		// bytes: a Good CloseSession round trip naming a token the
		// server never issued.
		request, err := closeSessionWire(ua.ServiceTypeID(&ua.CloseSessionRequest{}), 91,
			&ua.CloseSessionRequest{
				RequestHeader:       &ua.RequestHeader{AuthenticationToken: ua.NewStringNodeID(1, "spectest.untracked")},
				DeleteSubscriptions: true,
			})
		Expect(err).NotTo(HaveOccurred(), "encoding the CloseSession with an untracked token failed")
		response, err := closeSessionWire(ua.ServiceTypeID(&ua.CloseSessionResponse{}), 91,
			&ua.CloseSessionResponse{ResponseHeader: &ua.ResponseHeader{ServiceDiagnostics: &ua.DiagnosticInfo{}}})
		Expect(err).NotTo(HaveOccurred(), "encoding the CloseSession response failed")
		env.Recorder.observe(0, clientToServer, request)
		env.Recorder.observe(0, serverToClient, response)

		Expect(env.Server.KnownSessions()).To(Equal(1),
			"a Good CloseSession for an untracked token changed the known-session count")
	})

	It("follows the client's subscriptions in LiveSubscriptions", func() {
		env := Start(GinkgoT())
		Expect(env.Server.LiveSubscriptions()).To(Equal(1),
			"a client with one subscription leaves the server counting %d live subscriptions, want 1", env.Server.LiveSubscriptions())

		notifications := make(chan *opcua.PublishNotificationData, notificationBuffer)
		ctx, cancel := context.WithTimeout(context.Background(), specWait)
		defer cancel()
		_, err := env.Client.Subscribe(ctx, &opcua.SubscriptionParameters{
			Interval:          defaultPublishingInterval,
			LifetimeCount:     lifetimeCount,
			MaxKeepAliveCount: maxKeepAliveCount,
		}, notifications)
		Expect(err).NotTo(HaveOccurred(), "the client's second Subscribe failed")
		Expect(env.Server.LiveSubscriptions()).To(Equal(2),
			"after a second Subscribe the server counts %d live subscriptions, want 2", env.Server.LiveSubscriptions())

		cancelCtx, cancelCancel := context.WithTimeout(context.Background(), specWait)
		defer cancelCancel()
		Expect(env.ClientSubscription().Cancel(cancelCtx)).To(Succeed(), "cancelling the first subscription failed")
		Expect(env.Server.LiveSubscriptions()).To(Equal(1),
			"after cancelling the first subscription the server counts %d live subscriptions, want 1", env.Server.LiveSubscriptions())
	})

	It("answers the next CreateSubscription with the scripted ServiceFault", func() {
		env := Start(GinkgoT())
		env.Server.AnswerNextWith(message.CreateSubscription, ua.StatusBadTooManyOperations)
		m := env.Mark()

		notifications := make(chan *opcua.PublishNotificationData, notificationBuffer)
		ctx, cancel := context.WithTimeout(context.Background(), specWait)
		defer cancel()
		_, subscribeErr := env.Client.Subscribe(ctx, &opcua.SubscriptionParameters{
			Interval:          defaultPublishingInterval,
			LifetimeCount:     lifetimeCount,
			MaxKeepAliveCount: maxKeepAliveCount,
		}, notifications)
		Expect(subscribeErr).To(MatchError(ua.StatusBadTooManyOperations),
			"a Subscribe after AnswerNextWith(CreateSubscription, Bad_TooManyOperations) failed with %v", subscribeErr)

		Eventually(func(g Gomega) {
			var creates []ServiceRecord[ua.Request]
			for _, record := range env.Recorder.RequestsSince(m) {
				decoded, ok := record.Message()
				if !ok {
					continue
				}
				if _, isCreate := decoded.(*ua.CreateSubscriptionRequest); isCreate {
					creates = append(creates, record)
				}
			}
			g.Expect(creates).To(HaveLen(1),
				"the client sent %d CreateSubscription requests, want one", len(creates))
			answer := responseOf(env.Recorder.ResponsesSince(m), creates[0].Connection, creates[0].RequestID)
			g.Expect(answer).NotTo(BeNil(), "no response is recorded for the CreateSubscription")
			message, ok := answer.Message()
			g.Expect(ok).To(BeTrue(), "the recorded response yields no decoded message")
			fault, isFault := message.(*ua.ServiceFault)
			g.Expect(isFault).To(BeTrue(), "the recorded response is a %T, want a ServiceFault", message)
			g.Expect(fault.ResponseHeader.ServiceResult).To(Equal(ua.StatusBadTooManyOperations),
				"the ServiceFault carries status %v, want Bad_TooManyOperations", fault.ResponseHeader.ServiceResult)
		}, specWait).Should(Succeed())
	})

	It("answers the next Publish at once with the scripted ServiceFault", func() {
		env := Start(GinkgoT())
		held := env.Server.WaitHeldPublish()
		env.Server.AnswerNextWith(message.Publish, ua.StatusBadTooManyPublishRequests)
		m := env.Mark()
		held.Answer(env.Subscription(), valueOverloaded)

		Eventually(func(g Gomega) {
			var publishes []ServiceRecord[ua.Request]
			for _, record := range env.Recorder.RequestsSince(m) {
				decoded, ok := record.Message()
				if !ok {
					continue
				}
				if _, isPublish := decoded.(*ua.PublishRequest); isPublish && record.Connection == held.Connection() {
					publishes = append(publishes, record)
				}
			}
			g.Expect(publishes).ToNot(BeEmpty(),
				"the client sent no Publish request after the previous one was answered")
			answer := responseOf(env.Recorder.ResponsesSince(m), publishes[0].Connection, publishes[0].RequestID)
			g.Expect(answer).NotTo(BeNil(), "no response is recorded for the next Publish")
			message, ok := answer.Message()
			g.Expect(ok).To(BeTrue(), "the recorded response yields no decoded message")
			fault, isFault := message.(*ua.ServiceFault)
			g.Expect(isFault).To(BeTrue(), "the recorded response is a %T, want a ServiceFault", message)
			g.Expect(fault.ResponseHeader.ServiceResult).To(Equal(ua.StatusBadTooManyPublishRequests),
				"the ServiceFault carries status %v, want Bad_TooManyPublishRequests", fault.ResponseHeader.ServiceResult)
		}, specWait).Should(Succeed())
	})

	It("rejects AnswerNextWith for a message the harness does not serve", func() {
		env := Start(GinkgoT())
		fake := &fakeT{}
		env.Server.t = fake
		Expect(fatalPanics(func() { env.Server.AnswerNextWith(message.Read, ua.StatusBadTooManyOperations) })).To(BeTrue(),
			"AnswerNextWith did not fail for Read")
		Expect(fatalPanics(func() { env.Server.AnswerNextWith(message.CreateSession, ua.StatusBadTooManyOperations) })).To(BeTrue(),
			"AnswerNextWith did not fail for CreateSession")
		Expect(fatalPanics(func() { env.Server.AnswerNextWith(message.CloseSecureChannel, ua.StatusBadTooManyOperations) })).To(BeTrue(),
			"AnswerNextWith did not fail for CloseSecureChannel")
		env.Server.t = GinkgoT()
		Expect(fake.fatals).To(HaveLen(3),
			"AnswerNextWith failed %d times, want once per rejected message", len(fake.fatals))
		for i, message := range []string{"Read", "CreateSession", "CloseSecureChannel"} {
			Expect(fake.fatals[i]).To(HavePrefix("spectest:"),
				"AnswerNextWith(%s) did not fail as a harness fault", message)
			Expect(fake.fatals[i]).To(ContainSubstring(message),
				"the harness fault does not name %s", message)
		}
	})

	It("holds the next response the relay writes to the client", func() {
		env := Start(GinkgoT())
		env.Relay.HoldNextResponse(time.Second)
		m := env.Mark()

		readNode(env.Client, env.Server.node)

		var answer ServiceRecord[ua.Response]
		Eventually(func(g Gomega) {
			var found bool
			_, answer, found = readPairSince(env, m)
			g.Expect(found).To(BeTrue(), "no answered Read round trip is recorded")
		}, specWait).Should(Succeed())
		read, written, ok := env.Recorder.TimesOf(answer.Order)
		Expect(ok).To(BeTrue(), "the Read response was never written to the client")
		Expect(written.Sub(read)).To(BeNumerically(">=", 950*time.Millisecond),
			"the response was written %s after the request was read, want at least 950 ms", written.Sub(read))
	})
})

// closeSessionWire encodes a CloseSession request or response body as
// one message chunk on the request id given.
func closeSessionWire(typeID uint16, requestID uint32, service any) ([]byte, error) {
	if typeID == 0 {
		return nil, fmt.Errorf("ua.ServiceTypeID returned 0 for %T", service)
	}
	body := ua.NewBuffer(nil)
	body.WriteStruct(ua.NewFourByteExpandedNodeID(0, typeID))
	body.WriteStruct(service)
	if body.Error() != nil {
		return nil, body.Error()
	}
	return messageChunk(uacp.ChunkTypeFinal, requestID, body.Bytes())
}
