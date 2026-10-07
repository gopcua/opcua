package harness

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const outliveWindow = 25 * time.Second

var _ = Describe("Environment Start", func() {
	It("completes the session handshake, records the subscription and delivers the first answered Publish", func() {
		env := Start(GinkgoT())

		wantSequence := []string{
			"*ua.OpenSecureChannelRequest",
			"*ua.CreateSessionRequest",
			"*ua.ActivateSessionRequest",
			"*ua.CreateSubscriptionRequest",
		}
		var names []string
		for _, r := range env.Recorder.Requests() {
			m, ok := r.Message()
			if !ok {
				continue
			}
			names = append(names, fmt.Sprintf("%T", m))
		}
		matched := 0
		for _, name := range names {
			if matched < len(wantSequence) && name == wantSequence[matched] {
				matched++
			}
		}
		Expect(matched).To(Equal(len(wantSequence)),
			"the client did not send OpenSecureChannel, CreateSession, ActivateSession and CreateSubscription in that order among its requests: %v", names)

		var createResp *ua.CreateSubscriptionResponse
		for _, r := range env.Recorder.Responses() {
			if m, ok := r.Message(); ok {
				if resp, isCreate := m.(*ua.CreateSubscriptionResponse); isCreate {
					createResp = resp
				}
			}
		}
		Expect(createResp).NotTo(BeNil(), "the recorder saw no CreateSubscriptionResponse")
		Expect(createResp.SubscriptionID).To(Equal(env.Subscription().ID()),
			"the recorded subscription id does not match the harness subscription")

		Expect(env.LastSequenceNumber()).To(Equal(uint32(1)),
			"the first answered Publish does not carry sequence number 1")

		received := env.Received()
		Expect(received).To(HaveLen(1), "the harness did not deliver exactly the pre-cut value")
		answered := env.Recorder.answeredPublishes()
		Expect(answered).NotTo(BeEmpty(), "the recorder saw no answered Publish response")
		Expect(answered[len(answered)-1].value).To(Equal(received[0]),
			"the value of the last sequenced PublishResponse is not the delivered value")
	})

	It("reconnects through exactly one fresh relay connection and records the state path", func() {
		env := Start(GinkgoT())
		m := env.Mark()
		before := env.Relay.ConnectionCount()
		oldAddr := serverAddrOf(env.Recorder, 0)
		Expect(oldAddr).NotTo(BeNil(), "the recorder did not map a server-side address to connection 0")
		index, state := env.Recorder.ConnectionOf(oldAddr)
		Expect(index).To(Equal(0), "the live connection's address does not map to connection 0")
		Expect(state).To(Equal(Open), "the relay does not report the live connection as open")

		env.Relay.Cut()
		env.WaitUntilReconnected()

		Expect(env.ConnectionsSince(m)).To(Equal(1),
			"the client did not reconnect through exactly one new relay connection")
		Expect(env.Relay.ConnectionCount()).To(Equal(before+1),
			"the relay connection count did not grow by exactly one")
		index, state = env.Recorder.ConnectionOf(oldAddr)
		Expect(index).To(Equal(0), "the cut connection's address no longer maps to connection 0")
		Expect(state).To(Equal(Closed), "the relay does not report the cut connection as closed")
		newAddr := serverAddrOf(env.Recorder, 1)
		Expect(newAddr).NotTo(BeNil(), "the recorder did not map a server-side address to the reconnected connection")
		index, state = env.Recorder.ConnectionOf(newAddr)
		Expect(index).To(Equal(1), "the reconnected server-side address does not map to connection 1")
		Expect(state).To(Equal(Open), "the relay does not report the reconnected connection as open")

		states := env.StatesSince(m)
		sawDown := false
		sawConnected := false
		for _, state := range states {
			if state == opcua.Disconnected || state == opcua.Reconnecting {
				sawDown = true
			}
			if sawDown && state == opcua.Connected {
				sawConnected = true
			}
		}
		Expect(sawDown).To(BeTrue(),
			"the client reported no Disconnected or Reconnecting state after the cut: %v", states)
		Expect(sawConnected).To(BeTrue(),
			"the client reported no Connected state after a down state: %v", states)

		Eventually(func() []opcua.ConnState { return env.StatesSince(m) }).WithTimeout(specWait).
			Should(ContainElement(opcua.Connected), "the client never reported a Connected state after the cut")
		env.WaitUntilReconnected()

		_, state = env.Recorder.ConnectionOf(&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1})
		Expect(state).To(Equal(Unknown),
			"ConnectionOf maps an address the relay never accepted to a state other than Unknown")
	})

	It("delivers no received value and no error notification from a cut", func() {
		env := Start(GinkgoT())
		m := env.Mark()

		env.Relay.Cut()
		env.WaitUntilReconnected()

		Expect(env.ReceivedSince(m)).To(BeEmpty(),
			"a cut cannot deliver a value on this tree: the connection it cut is the only carrier")
		Expect(env.ReceivedErrorsSince(m)).To(BeEmpty(),
			"the client delivers no error notification for a cut on this tree")
	})

	It("fails spectest: when a held Publish is answered twice", func() {
		env := Start(GinkgoT())
		held := env.Server.WaitHeldPublish()
		Expect(held.Connection()).To(Equal(0),
			"the held Publish the client sent while connected did not arrive on connection 0")
		held.Answer(env.Subscription(), 5001)

		fake := &fakeT{}
		env.Server.t = fake
		Expect(fatalPanics(func() { held.Answer(env.Subscription(), 5002) })).To(BeTrue(),
			"answering a held Publish twice did not fail")
		env.Server.t = GinkgoT()

		Expect(fake.fatals).To(HaveLen(1),
			"answering a held Publish twice failed %d times, want exactly the already-answered fault", len(fake.fatals))
		Expect(fake.fatals[0]).To(ContainSubstring("was already answered"),
			"answering a held Publish twice did not fail as an already-answered fault")
		Expect(fake.fatals[0]).To(HavePrefix("spectest:"),
			"answering a held Publish twice did not fail as a harness fault")
	})

	It("fails spectest: when a held Publish is answered after its connection was cut", func() {
		env := Start(GinkgoT())
		held := env.Server.WaitHeldPublish()
		env.Relay.Cut()

		fake := &fakeT{}
		env.Server.t = fake
		Expect(fatalPanics(func() { held.Answer(env.Subscription(), 5003) })).To(BeTrue(),
			"answering a held Publish whose connection closed did not fail")
		env.Server.t = GinkgoT()

		Expect(fake.fatals).To(HaveLen(1),
			"answering a held Publish whose connection closed failed %d times, want exactly the closed-connection fault", len(fake.fatals))
		Expect(fake.fatals[0]).To(ContainSubstring("is not open"),
			"answering a held Publish whose connection closed did not fail as a closed-connection fault")
		Expect(fake.fatals[0]).To(HavePrefix("spectest:"),
			"answering a held Publish whose connection closed did not fail as a harness fault")
	})

	It("fails with the plain message when no Publish request arrives", func() {
		fake := &fakeT{}
		srv := newScriptedServer(fake)
		srv.heldWait = 20 * time.Millisecond
		defer func() {
			for _, cleanup := range fake.cleanups {
				cleanup()
			}
		}()

		Expect(fatalPanics(func() { srv.WaitHeldPublish() })).To(BeTrue(),
			"WaitHeldPublish without a client did not fail")
		Expect(fake.fatals).To(HaveLen(1),
			"WaitHeldPublish without a client did not fail exactly once")
		Expect(fake.fatals[0]).To(Equal("client sent no Publish request"),
			"WaitHeldPublish without a client did not fail with the plain message")
	})

	It("fails spectest: when a client bypasses the relay and publishes to the scripted server", func() {
		env := Start(GinkgoT())
		direct, err := opcua.NewClient(env.Server.Address(), opcua.SecurityMode(ua.MessageSecurityModeNone))
		Expect(err).NotTo(HaveOccurred(), "creating the direct client failed")
		connectCtx, connectCancel := context.WithTimeout(context.Background(), specWait)
		defer connectCancel()
		Expect(direct.Connect(connectCtx)).To(Succeed(), "the direct client never connected to the scripted server")
		GinkgoT().Cleanup(func() {
			closeCtx, closeCancel := context.WithTimeout(context.Background(), specWait)
			defer closeCancel()
			_ = direct.Close(closeCtx)
		})
		notifications := make(chan *opcua.PublishNotificationData, notificationBuffer)
		subscribeCtx, subscribeCancel := context.WithTimeout(context.Background(), specWait)
		defer subscribeCancel()
		subscription, err := direct.Subscribe(subscribeCtx, &opcua.SubscriptionParameters{
			Interval:          100 * time.Millisecond,
			LifetimeCount:     lifetimeCount,
			MaxKeepAliveCount: maxKeepAliveCount,
		}, notifications)
		Expect(err).NotTo(HaveOccurred(), "the direct client created no subscription")
		monitorCtx, monitorCancel := context.WithTimeout(context.Background(), specWait)
		defer monitorCancel()
		_, err = subscription.Monitor(monitorCtx, ua.TimestampsToReturnBoth,
			opcua.NewMonitoredItemCreateRequestWithDefaults(env.Server.node, ua.AttributeIDValue, monitorClientHandle))
		Expect(err).NotTo(HaveOccurred(), "the direct client monitored no node")

		Eventually(func() bool {
			env.Server.mu.Lock()
			defer env.Server.mu.Unlock()
			return env.Server.faultErr != nil
		}).WithTimeout(specWait).Should(BeTrue(),
			"the direct client's Publish never reached the scripted server as an off-relay request")

		fake := &fakeT{}
		env.Server.t = fake
		Expect(fatalPanics(func() { env.Server.WaitHeldPublish() })).To(BeTrue(),
			"WaitHeldPublish did not fail on the off-relay Publish")
		env.Server.t = GinkgoT()
		env.Server.mu.Lock()
		env.Server.faultErr = nil
		env.Server.mu.Unlock()
		Expect(fake.fatals).To(HaveLen(1),
			"the off-relay Publish failed %d times, want exactly one harness fault", len(fake.fatals))
		Expect(fake.fatals[0]).To(HavePrefix("spectest:"),
			"the off-relay Publish did not fail as a harness fault")
		Expect(fake.fatals[0]).To(ContainSubstring("never accepted"),
			"the off-relay Publish did not fail naming the relay")
	})

	It("never lets the server's subscription service delete the subscription across 25 s of publishing", func() {
		env := Start(GinkgoT(), WithPublishingInterval(10*time.Millisecond))
		sub := env.Subscription()
		service := env.Server.srv.SubscriptionService
		service.Mu.Lock()
		initial := len(service.Subs)
		service.Mu.Unlock()
		Expect(initial).To(BeNumerically(">", 0),
			"the subscription service holds no subscription, so this check would pass without testing anything")

		expected := env.Received()
		value := valueBeforeCut + 1
		deadline := time.Now().Add(outliveWindow)
		for time.Now().Before(deadline) {
			env.Server.WaitHeldPublish().Answer(sub, value)
			expected = append(expected, value)
			value++
			service.Mu.Lock()
			current := len(service.Subs)
			service.Mu.Unlock()
			Expect(current).To(BeNumerically(">=", initial),
				"the server's subscription service shrank its Subs map from %d to %d, so it deleted the subscription before the test ended", initial, current)
		}

		Eventually(func() []int32 { return env.Received() }).WithTimeout(15*time.Second).Should(Equal(expected),
			"the client did not deliver every answered value exactly once, in order")

		for _, r := range env.Recorder.Responses() {
			m, ok := r.Message()
			if !ok {
				continue
			}
			if resp, isPublish := m.(*ua.PublishResponse); isPublish {
				Expect(resp.SubscriptionID).To(Equal(sub.ID()),
					"an answered Publish response carries a subscription id other than the pre-cut one")
			}
		}
	})
})

var _ = Describe("Environment TryWaitUntilReconnected", func() {
	It("reports the reconnect after a cut", func() {
		env := Start(GinkgoT())
		env.Relay.Cut()
		Expect(env.TryWaitUntilReconnected(specWait)).To(BeTrue(),
			"the client did not pass through Reconnecting back to Connected within %s of the cut", specWait)
	})

	It("reports no reconnect when the connection never dropped", func() {
		env := Start(GinkgoT())
		Expect(env.TryWaitUntilReconnected(200*time.Millisecond)).To(BeFalse(),
			"TryWaitUntilReconnected reported a reconnect although the connection never dropped")
	})
})

type addr string

func (a addr) Network() string { return "tcp" }
func (a addr) String() string  { return string(a) }

func serverAddrOf(recorder *Recorder, index int) net.Addr {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	for a, entry := range recorder.connections {
		if entry.index == index {
			return addr(a)
		}
	}
	return nil
}
