package harness

import (
	"time"

	"github.com/gopcua/opcua"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Connected hold", func() {
	It("releases the Connected hold only after the relay recorded the cut connection closed", func() {
		env := New(GinkgoT())

		// A Connected report with no fired after-response cut must not
		// wait: a client that never armed one is never delayed.
		before := time.Now()
		env.holdConnectedAfterCutClose(opcua.Connected)
		Expect(time.Since(before)).To(BeNumerically("<", time.Second),
			"the Connected hold waited although no after-response cut fired")

		// A fired after-response cut whose connection the relay has not
		// recorded closed must hold the report: whatever the client's
		// monitor does after the report must not race the teardown of
		// the cut connection. Connection 0 is the live client's, so its
		// close is recorded only once the relay cuts it.
		env.Relay.mu.Lock()
		env.Relay.draining[0] = true
		env.Relay.mu.Unlock()
		start := time.Now()
		released := make(chan struct{})
		go func() {
			env.holdConnectedAfterCutClose(opcua.Connected)
			close(released)
		}()
		Consistently(released, 1*time.Second, 10*time.Millisecond).ShouldNot(BeClosed(),
			"the Connected hold returned while the relay had not recorded the cut connection closed")

		// The hold must release — a client whose cut connection never
		// closes would otherwise be blocked — but never before its
		// bound.
		Eventually(released, connectedHoldBound+3*time.Second, 10*time.Millisecond).Should(BeClosed(),
			"the Connected hold was still held after its bound, so a client would be blocked")
		Expect(time.Since(start)).To(BeNumerically(">=", connectedHoldBound),
			"the Connected hold released before its bound while the cut connection was open")

		// Once the relay records the close, a Connected report passes
		// through at once.
		env.Relay.Cut()
		Expect(env.Recorder.ConnectionStateOf(0)).To(Equal(Closed),
			"the relay cut did not record connection 0 closed")
		before = time.Now()
		env.holdConnectedAfterCutClose(opcua.Connected)
		Expect(time.Since(before)).To(BeNumerically("<", time.Second),
			"the Connected hold waited although the relay had recorded the cut connection closed")
	})
})
