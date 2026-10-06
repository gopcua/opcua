package spectest

import (
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/tests/spec/faults"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Relay DelayAt", func() {
	It("holds the reconnect's Read below the client's request timeout", func() {
		env := Start(GinkgoT(), WithClientOptions(opcua.RequestTimeout(2*time.Second)))
		env.Relay.DelayAt(faults.Read, time.Second)
		m := env.Mark()

		env.Relay.Cut()
		env.WaitUntilReconnected()

		var held ServiceRecord[ua.Request]
		Eventually(func(g Gomega) {
			var reads []ServiceRecord[ua.Request]
			for _, r := range env.Recorder.RequestsSince(m) {
				decoded, ok := r.Message()
				if !ok {
					continue
				}
				if _, isRead := decoded.(*ua.ReadRequest); isRead && r.Fate == Forwarded {
					reads = append(reads, r)
				}
			}
			g.Expect(reads).NotTo(BeEmpty(),
				"no forwarded Read request was recorded after the cut")
			for _, r := range reads {
				if read, written, ok := env.Recorder.TimesOf(r.Order); ok {
					if written.Sub(read) >= 950*time.Millisecond {
						held = r
					}
				}
			}
			g.Expect(held).NotTo(BeZero(),
				"no Read request was held at least 950 ms between read and write")
		}, specWait).Should(Succeed())

		Expect(env.ConnectionsSince(m)).To(Equal(1),
			"the relay accepted %d connections after the cut, want exactly one: the delay stayed below the client's request timeout, so the client kept the connection", env.ConnectionsSince(m))
		readNode(env.Client, env.Server.node)
	})

	It("holds the reconnect's Read past the client's request timeout", func() {
		env := Start(GinkgoT(), WithClientOptions(opcua.RequestTimeout(2*time.Second)))
		env.Relay.DelayAt(faults.Read, 4*time.Second)
		m := env.Mark()

		env.Relay.Cut()
		env.WaitUntilReconnected()

		var held ServiceRecord[ua.Request]
		Eventually(func(g Gomega) {
			var reads []ServiceRecord[ua.Request]
			for _, r := range env.Recorder.RequestsSince(m) {
				decoded, ok := r.Message()
				if !ok {
					continue
				}
				if _, isRead := decoded.(*ua.ReadRequest); isRead && r.Fate == Forwarded {
					reads = append(reads, r)
				}
			}
			for _, r := range reads {
				if read, written, ok := env.Recorder.TimesOf(r.Order); ok {
					if written.Sub(read) >= 3950*time.Millisecond {
						held = r
					}
				}
			}
			g.Expect(held).NotTo(BeZero(),
				"no Read request was held at least 3.95 s between read and write")
		}, 30*time.Second).Should(Succeed())

		Eventually(func(g Gomega) {
			g.Expect(env.ConnectionsSince(m)).To(BeNumerically(">=", 2),
				"the relay accepted %d connections after the cut, want at least two: the delay exceeded the client's request timeout, so the client gave up and reconnected", env.ConnectionsSince(m))
		}, 30*time.Second).Should(Succeed())
		env.WaitUntilReconnected()
	})

	It("writes later messages of the held connection after the held one, in their order", func() {
		relay, recorder, _ := newInjectedRelay()
		relay.DelayAt(faults.Read, 500*time.Millisecond)

		first, err := readRequestWire(31)
		Expect(err).NotTo(HaveOccurred(), "encoding the Read failed")
		second, err := readRequestWire(32)
		Expect(err).NotTo(HaveOccurred(), "encoding the second Read failed")
		third, err := readRequestWire(33)
		Expect(err).NotTo(HaveOccurred(), "encoding the third Read failed")

		var writeTimes []time.Time
		start := time.Now()
		Expect(recorder.forward(0, clientToServer, append(append(first, second...), third...), func(message []byte) error {
			writeTimes = append(writeTimes, time.Now())
			return nil
		}, func(armedCut) error { return nil })).To(Succeed(),
			"observing the three Reads failed")

		Expect(len(writeTimes)).To(Equal(3),
			"the relay wrote %d messages, want the three Reads", len(writeTimes))
		Expect(writeTimes[0].Sub(start)).To(BeNumerically(">=", 450*time.Millisecond),
			"the held Read was written %s after the forward call, want at least 450 ms", writeTimes[0].Sub(start))
		for i := 1; i < len(writeTimes); i++ {
			Expect(writeTimes[i].After(writeTimes[i-1])).To(BeTrue(),
				"message %d was written before message %d although all were released in order", i, i-1)
		}
		requests := recorder.Requests()
		Expect(requests).To(HaveLen(3),
			"the three Reads left %d request records, want 3", len(requests))
		for i, record := range requests {
			Expect(record.RequestID).To(Equal(uint32(31+i)),
				"request record %d holds request id %d, want %d in their wire order", i, record.RequestID, 31+i)
		}
	})

	It("writes the first chunk of a two-chunk request at once and the final chunk after the delay", func() {
		relay, recorder, _ := newInjectedRelay()
		relay.DelayAt(faults.Read, 500*time.Millisecond)

		typeID := ua.ServiceTypeID(&ua.ReadRequest{})
		Expect(typeID).NotTo(BeZero(), "ua.ServiceTypeID returned 0 for *ua.ReadRequest")
		body := ua.NewBuffer(nil)
		body.WriteStruct(ua.NewFourByteExpandedNodeID(0, typeID))
		body.WriteStruct(newReadRequest(41, ua.NewStringNodeID(1, "spectest.delay")))
		Expect(body.Error()).NotTo(HaveOccurred(), "encoding the Read body failed")
		split := len(body.Bytes()) / 2
		intermediate, err := messageChunk(uacp.ChunkTypeIntermediate, 41, body.Bytes()[:split])
		Expect(err).NotTo(HaveOccurred(), "encoding the intermediate chunk failed")
		final, err := messageChunk(uacp.ChunkTypeFinal, 41, body.Bytes()[split:])
		Expect(err).NotTo(HaveOccurred(), "encoding the final chunk failed")

		var writeTimes []time.Time
		start := time.Now()
		Expect(recorder.forward(0, clientToServer, append(intermediate, final...), func(message []byte) error {
			writeTimes = append(writeTimes, time.Now())
			return nil
		}, func(armedCut) error { return nil })).To(Succeed(),
			"observing the two-chunk Read failed")

		Expect(len(writeTimes)).To(Equal(2),
			"the relay wrote %d messages, want the two chunks", len(writeTimes))
		Expect(writeTimes[0].Sub(start)).To(BeNumerically("<", 250*time.Millisecond),
			"the intermediate chunk was written %s after the forward call, want it at once", writeTimes[0].Sub(start))
		Expect(writeTimes[1].Sub(start)).To(BeNumerically(">=", 450*time.Millisecond),
			"the final chunk was written %s after the forward call, want at least 450 ms", writeTimes[1].Sub(start))
		requests := recorder.Requests()
		Expect(requests).To(HaveLen(1),
			"the two-chunk Read left %d request records, want 1", len(requests))
		Expect(requests[0].RequestID).To(Equal(uint32(41)),
			"the reassembled record holds request id %d, want 41", requests[0].RequestID)
	})

	It("rejects DelayAt on CloseSecureChannel and on an unknown Message", func() {
		relay, _, _ := newInjectedRelay()
		ft := &fakeT{}
		relay.t = ft

		Expect(fatalPanics(func() { relay.DelayAt(faults.CloseSecureChannel, time.Second) })).To(BeTrue(),
			"DelayAt did not fail for CloseSecureChannel")
		Expect(fatalPanics(func() { relay.DelayAt(faults.Message(0), time.Second) })).To(BeTrue(),
			"DelayAt did not fail for an unknown Message")
		Expect(ft.fatals).To(HaveLen(2),
			"DelayAt failed %d times, want twice", len(ft.fatals))
		Expect(ft.fatals[0]).To(HavePrefix("spectest:"),
			"DelayAt on CloseSecureChannel did not fail as a harness fault")
		Expect(ft.fatals[0]).To(ContainSubstring("CloseSecureChannel"),
			"the harness fault does not name CloseSecureChannel")
		Expect(ft.fatals[1]).To(HavePrefix("spectest:"),
			"DelayAt on an unknown Message did not fail as a harness fault")
	})
})
