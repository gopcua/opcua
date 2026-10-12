package harness

import (
	"os"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Relay link faults", func() {
	It("closes the next three accepted connections before dialling upstream", func() {
		env := New(GinkgoT())
		env.Relay.CloseNextAccepts(3)
		m := env.Mark()

		env.Relay.Cut()
		env.WaitUntilReconnected()

		Eventually(func(g Gomega) {
			g.Expect(env.ConnectionsSince(m)).To(BeNumerically(">=", 4),
				"the relay accepted %d connections after the cut, want at least four: three closed on accept and the fourth connected", env.ConnectionsSince(m))
		}, specWait).Should(Succeed())
		var withoutUpstream []int
		var dialled []int
		for index := m.connections; index < env.Relay.ConnectionCount(); index++ {
			if env.Recorder.connectionUpstreamOf(index) == "" {
				withoutUpstream = append(withoutUpstream, index)
			}
			if env.Relay.dialledUpstream(index) {
				dialled = append(dialled, index)
			}
		}
		Expect(withoutUpstream).To(Equal([]int{m.connections, m.connections + 1, m.connections + 2}),
			"the connections with no upstream are %v, want exactly the first three accepted after the cut", withoutUpstream)
		Expect(dialled).To(Equal([]int{m.connections + 3}),
			"the relay dialled the upstream server for connections %v, want only the first connection it did not close on accept", dialled)
		readNode(env.Client, env.Server.node)
	})

	It("closes the listener for a window and reopens it on the same address", func() {
		env := New(GinkgoT())
		_, reconnect, err := resolveIntervals(options{}, os.Getenv)
		Expect(err).NotTo(HaveOccurred(), "resolving the reconnect interval failed")
		window := 3 * reconnect
		address := env.Relay.address()

		closedAt := time.Now()
		env.Relay.CloseListenerFor(window)
		m := env.Mark()
		env.Relay.Cut()

		deadline := closedAt.Add(window)
		for time.Now().Before(deadline) {
			Expect(env.Relay.ConnectionCount()).To(Equal(m.connections),
				"the relay accepted a connection while its listener was closed")
			time.Sleep(10 * time.Millisecond)
		}

		env.WaitUntilReconnected()
		Expect(env.Relay.address()).To(Equal(address),
			"the relay's address changed after reopening its listener")
		readNode(env.Client, env.Server.node)
	})

	It("stalls the current connection in both directions while later connections forward normally", func() {
		env := New(GinkgoT())
		stalledIndex := env.Relay.ConnectionCount() - 1
		m := env.Mark()

		env.Relay.Stall()
		// The stalled connection swallows the request, so the read
		// fails on the client's request timeout; it only has to make
		// the client send one.
		go func() { _, _ = readNodeOnce(env.Client, env.Server.node) }()
		var stalled []ServiceRecord[ua.Request]
		Eventually(func(g Gomega) {
			stalled = nil
			for _, record := range env.Recorder.RequestsSince(m) {
				if record.Fate == Stalled {
					stalled = append(stalled, record)
				}
			}
			g.Expect(stalled).NotTo(BeEmpty(),
				"no request was recorded Stalled although the connection is stalled")
		}, specWait).Should(Succeed())
		for _, record := range stalled {
			Expect(record.Connection).To(Equal(stalledIndex),
				"a request on connection %d was recorded Stalled, want only the stalled connection %d", record.Connection, stalledIndex)
			Expect(record.WrittenAt).To(BeZero(),
				"a Stalled request was written to the server although the connection is stalled")
		}
		Consistently(func(g Gomega) {
			g.Expect(env.Recorder.ConnectionStateOf(stalledIndex)).To(Equal(Open),
				"the stalled connection did not stay open")
		}, 2*time.Second).Should(Succeed())

		env.Relay.Cut()
		env.WaitUntilReconnected()
		Expect(env.Relay.ConnectionCount()).To(BeNumerically(">", stalledIndex+1),
			"the client opened no new connection after the stalled one")
		var later []ServiceRecord[ua.Request]
		for _, record := range env.Recorder.RequestsSince(m) {
			if record.Connection > stalledIndex {
				later = append(later, record)
			}
		}
		Expect(later).NotTo(BeEmpty(),
			"no request rode a connection the client opened after the stalled one")
		for _, record := range later {
			Expect(record.Fate).ToNot(Equal(Stalled),
				"a request on connection %d, opened after the stall, is recorded Stalled", record.Connection)
		}
		readNode(env.Client, env.Server.node)
	})

	It("forwards HEL and discards the ACK of the next accepted connection", func() {
		env := New(GinkgoT())
		env.Relay.DiscardNextACK()
		m := env.Mark()

		env.Relay.Cut()

		var ack TransportRecord
		Eventually(func(g Gomega) {
			var acks []TransportRecord
			for _, record := range env.Recorder.Transport() {
				if record.Order > m.order && record.Type == ACK {
					acks = append(acks, record)
				}
			}
			g.Expect(acks).To(HaveLen(1),
				"the relay recorded %d ACKs after the cut, want the one of the next connection", len(acks))
			ack = acks[0]
		}, specWait).Should(Succeed())
		Expect(ack.Fate).To(Equal(Stalled),
			"the discarded ACK is recorded %s, want Stalled", ack.Fate)
		hel, found := helRecordOnConnection(env, ack.Connection)
		Expect(found).To(BeTrue(),
			"the connection whose ACK was discarded carries no HEL")
		Expect(hel.Fate).To(Equal(Forwarded),
			"the HEL whose ACK was discarded is recorded %s, want Forwarded", hel.Fate)
		Consistently(func(g Gomega) {
			g.Expect(env.StatesSince(m)).NotTo(ContainElement(opcua.Connected),
				"the client reported Connected although its HEL was never acknowledged")
			g.Expect(env.Recorder.ConnectionStateOf(ack.Connection)).To(Equal(Open),
				"the connection whose ACK was discarded did not stay open")
		}, 2*time.Second).Should(Succeed())

		// The discard is one shot, so cutting the connection whose ACK
		// it ate lets the client's next handshake complete; the spec
		// leaves the client connected instead of wedged on an ACK that
		// never comes.
		env.Relay.Cut()
		env.WaitUntilReconnected()
		readNode(env.Client, env.Server.node)
	})
})
