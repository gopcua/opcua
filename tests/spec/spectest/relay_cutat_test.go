package spectest

import (
	"bytes"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/tests/spec/message"
	"github.com/gopcua/opcua/ua"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func readNodeExpectingFailure(client *opcua.Client, nodeID *ua.NodeID) error {
	_, err := readNodeOnce(client, nodeID)
	return err
}

func readRequestsSince(env *Environment, m Mark) []ServiceRecord[ua.Request] {
	var reads []ServiceRecord[ua.Request]
	for _, r := range env.Recorder.RequestsSince(m) {
		message, ok := r.Message()
		if !ok {
			continue
		}
		if _, isRead := message.(*ua.ReadRequest); isRead {
			reads = append(reads, r)
		}
	}
	return reads
}

func responseOf(responses []ServiceRecord[ua.Response], connection int, requestID uint32) *ServiceRecord[ua.Response] {
	for i := range responses {
		if responses[i].Connection != connection || responses[i].RequestID != requestID {
			continue
		}
		return &responses[i]
	}
	return nil
}

var _ = Describe("Relay CutAt", func() {
	It("drops the next Read before it reaches the server and cuts the connection", func() {
		env := Start(GinkgoT())
		env.Relay.CutAt(BeforeRequestReachesServer, message.Read)
		m := env.Mark()

		readErr := readNodeExpectingFailure(env.Client, env.Server.node)
		Expect(readErr).To(HaveOccurred(),
			"the client's Read succeeded although the armed cut dropped its request")

		var dropped []ServiceRecord[ua.Request]
		for _, r := range readRequestsSince(env, m) {
			if r.Fate == Dropped {
				dropped = append(dropped, r)
			}
		}
		Expect(dropped).To(HaveLen(1),
			"RequestsSince(m) holds %d ReadRequests with Fate Dropped, want exactly the one the cut dropped", len(dropped))
		droppedRead := dropped[0]
		Expect(droppedRead.Connection).To(Equal(0),
			"the dropped Read was not recorded on the connection the client read through")
		message, ok := droppedRead.Message()
		Expect(ok).To(BeTrue(),
			"the dropped Read record does not yield its decoded message")
		readRequest, isRead := message.(*ua.ReadRequest)
		Expect(isRead).To(BeTrue(), "the dropped Read record holds a %T, want a *ua.ReadRequest", message)
		Expect(readRequest.NodesToRead).To(HaveLen(1))
		Expect(readRequest.NodesToRead[0].NodeID).To(Equal(env.Server.node),
			"the dropped Read does not name the harness node")
		Expect(responseOf(env.Recorder.ResponsesSince(m), 0, droppedRead.RequestID)).To(BeNil(),
			"connection 0 carries a response with the dropped Read's request id %d although its request never reached the server", droppedRead.RequestID)
		Expect(env.Relay.ArmedCuts()).To(BeEmpty(),
			"ArmedCuts still lists the cut after it fired")

		env.WaitUntilReconnected()
		Expect(env.ConnectionsSince(m)).To(Equal(1),
			"the client did not reconnect through exactly one new relay connection after the cut dropped its Read")
	})

	It("writes the Read response to the client before the after-response cut closes the connection", func() {
		env := Start(GinkgoT())
		env.Relay.CutAt(AfterResponseReachesClient, message.Read)
		m := env.Mark()

		readResp, readErr := readNodeOnce(env.Client, env.Server.node)
		Expect(readErr).NotTo(HaveOccurred(),
			"the client's Read failed although the after-response cut must deliver its response first")
		Expect(readResp).NotTo(BeNil(),
			"the client's Read returned no response although the after-response cut must deliver it")
		Expect(readResp.Results).To(HaveLen(1),
			"the Read response the after-response cut delivered carries %d results, want the harness node's one", len(readResp.Results))
		Expect(readResp.Results[0].Value.Value()).To(Equal(int32(0)),
			"the Read response the after-response cut delivered does not carry the harness node's value")

		var requestID uint32
		for _, r := range readRequestsSince(env, m) {
			if r.Connection == 0 {
				requestID = r.RequestID
			}
		}
		Expect(requestID).NotTo(BeZero(),
			"the recorder saw no Read request on connection 0 after the mark whose response the cut could fire on")
		response := responseOf(env.Recorder.ResponsesSince(m), 0, requestID)
		Expect(response).NotTo(BeNil(),
			"the recorder saw no response with the Read's request id on connection 0")
		Expect(response.Fate).To(Equal(Forwarded),
			"the cut did not record the response it wrote to the client as Forwarded")
		responseMessage, forwarded := response.Message()
		Expect(forwarded).To(BeTrue(),
			"the forwarded ReadResponse record does not yield its decoded message")
		_, isReadResponse := responseMessage.(*ua.ReadResponse)
		Expect(isReadResponse).To(BeTrue(),
			"the record with the Read's request id holds a %T, want a *ua.ReadResponse", responseMessage)
		env.WaitUntilReconnected()
		Expect(env.Relay.ArmedCuts()).To(BeEmpty(),
			"ArmedCuts still lists the cut after the client saw the connection close")
		Expect(env.ConnectionsSince(m)).To(Equal(1),
			"the client did not reconnect through exactly one new relay connection after the cut closed the connection")
	})

	It("lists armed cuts until they fire and fires two armed cuts in order on two consecutive Reads", func() {
		env := Start(GinkgoT())
		env.Relay.CutAt(BeforeRequestReachesServer, message.Read)
		env.Relay.CutAt(BeforeRequestReachesServer, message.Read)
		Expect(env.Relay.ArmedCuts()).To(Equal([]string{
			"before a Read request reaches the server",
			"before a Read request reaches the server",
		}), "ArmedCuts does not list both armed cuts in arming order before either fired")
		m := env.Mark()

		Expect(readNodeExpectingFailure(env.Client, env.Server.node)).To(HaveOccurred(),
			"the client's Read succeeded although the first armed cut dropped its request")

		Eventually(func(g Gomega) {
			g.Expect(env.Relay.ArmedCuts()).To(BeEmpty(),
				"the second armed cut did not fire on the reconnect's Read")
			var droppedReads []ServiceRecord[ua.Request]
			for _, r := range readRequestsSince(env, m) {
				if r.Fate == Dropped {
					droppedReads = append(droppedReads, r)
				}
			}
			g.Expect(droppedReads).To(HaveLen(2),
				"the two armed cuts did not drop exactly one Read each")
			g.Expect(droppedReads[0].Order).To(BeNumerically("<", droppedReads[1].Order),
				"the second armed cut fired before the first")
			g.Expect([]int{droppedReads[0].Connection, droppedReads[1].Connection}).To(Equal([]int{0, 1}),
				"the two armed cuts did not fire on the Reads of two consecutive connections")
			secondMessage, ok := droppedReads[1].Message()
			g.Expect(ok).To(BeTrue(),
				"the second dropped Read record does not yield its decoded message")
			secondRead, isRead := secondMessage.(*ua.ReadRequest)
			g.Expect(isRead).To(BeTrue(),
				"the second dropped Read record holds a %T, want a *ua.ReadRequest", secondMessage)
			g.Expect(secondRead.NodesToRead).To(HaveLen(1),
				"the reconnect's dropped Read does not read exactly one node")
			g.Expect(secondRead.NodesToRead[0].NodeID).To(Equal(ua.NewNumericNodeID(0, id.Server_NamespaceArray)),
				"the second armed cut fired on a Read that does not target the server's namespace array, so the client no longer reads the namespace array on reconnect")
		}).WithTimeout(specWait).Should(Succeed())

		Eventually(func() int { return env.ConnectionsSince(m) }).WithTimeout(specWait).Should(Equal(2),
			"the two cuts did not leave the client reconnecting through exactly two new relay connections")
		env.WaitUntilReconnected()
		readNode(env.Client, env.Server.node)
		Expect(env.Relay.ArmedCuts()).To(BeEmpty(),
			"ArmedCuts lists a cut again after a Read with no cut left to fire")
		Expect(env.ConnectionsSince(m)).To(Equal(2),
			"the client did not end on exactly two new relay connections after the two cuts")
	})

	It("does not fire a cut armed for Republish on a Read", func() {
		env := Start(GinkgoT())
		env.Relay.CutAt(BeforeRequestReachesServer, message.Republish)
		m := env.Mark()

		readResp := readNode(env.Client, env.Server.node)
		Expect(readResp.Results).To(HaveLen(1),
			"the Read a Republish cut must ignore carried %d results", len(readResp.Results))
		Expect(env.ConnectionsSince(m)).To(Equal(0),
			"a cut armed for Republish cut the connection although the client only sent a Read")
		Expect(env.Relay.ArmedCuts()).To(Equal([]string{
			"before a Republish request reaches the server",
		}), "ArmedCuts no longer lists the cut armed for Republish although no Republish was sent")
	})

	It("does not fire a cut armed for a Republish response on a Read round trip", func() {
		env := Start(GinkgoT())
		env.Relay.CutAt(AfterResponseReachesClient, message.Republish)
		m := env.Mark()

		readResp := readNode(env.Client, env.Server.node)
		Expect(readResp.Results).To(HaveLen(1),
			"the Read a Republish cut must ignore carried %d results", len(readResp.Results))
		Expect(env.ConnectionsSince(m)).To(Equal(0),
			"the cut armed for a Republish response cut the connection although the client only made a Read round trip")
		Expect(env.Relay.ArmedCuts()).To(Equal([]string{
			"after a Republish response reaches the client",
		}), "ArmedCuts no longer lists the cut armed for a Republish response after a Read round trip")
	})

	It("closes only the connection the cut fired on", func() {
		t := GinkgoT()
		upstreamListener, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred(), "the fake upstream could not listen")
		t.Cleanup(func() { _ = upstreamListener.Close() })
		upstreamConns := make(chan net.Conn, 4)
		go func() {
			for {
				conn, err := upstreamListener.Accept()
				if err != nil {
					return
				}
				upstreamConns <- conn
			}
		}()

		cutFired := make(chan struct{}, 1)
		relay, _ := newRelay(t, upstreamListener.Addr().String(), func() { cutFired <- struct{}{} })
		relay.CutAt(BeforeRequestReachesServer, message.Read)

		firstClient, err := net.Dial("tcp", relay.address())
		Expect(err).NotTo(HaveOccurred(), "dialing the relay for the first connection failed")
		secondClient, err := net.Dial("tcp", relay.address())
		Expect(err).NotTo(HaveOccurred(), "dialing the relay for the second connection failed")
		t.Cleanup(func() { _ = firstClient.Close() })
		t.Cleanup(func() { _ = secondClient.Close() })
		var firstUpstream, secondUpstream net.Conn
		Eventually(upstreamConns).WithTimeout(specWait).Should(Receive(&firstUpstream),
			"the relay never dialed the fake upstream for the first connection")
		Eventually(upstreamConns).WithTimeout(specWait).Should(Receive(&secondUpstream),
			"the relay never dialed the fake upstream for the second connection")

		request, err := readRequestWire(41)
		Expect(err).NotTo(HaveOccurred(), "encoding the Read request failed")
		_, err = firstClient.Write(request)
		Expect(err).NotTo(HaveOccurred(), "sending the Read on the first connection failed")

		firstClosed := make(chan error, 1)
		go func() {
			_, err := firstClient.Read(make([]byte, 1))
			firstClosed <- err
		}()
		Eventually(cutFired).WithTimeout(specWait).Should(Receive(),
			"the fired cut did not call onCut")
		var firstErr error
		Eventually(firstClosed).WithTimeout(specWait).Should(Receive(&firstErr),
			"the fired cut did not close the connection it fired on")
		Expect(firstErr).To(HaveOccurred(),
			"the first connection is still readable after the cut fired on it")

		_, err = secondClient.Write(request)
		Expect(err).NotTo(HaveOccurred(), "a write on the second connection failed although the cut fired on the first")
		secondData := make([]byte, len(request))
		_, err = io.ReadFull(secondUpstream, secondData)
		Expect(err).NotTo(HaveOccurred(),
			"the fake upstream saw nothing of the second connection's request, so the fired cut closed the second connection too: %v", err)
		Expect(secondData).To(Equal(request),
			"the fake upstream received something other than the second connection's request")
	})

	It("delivers the response before EOF and accepts the client's writes while draining after an after-response cut", func() {
		t := GinkgoT()
		upstreamListener, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred(), "the fake upstream could not listen")
		t.Cleanup(func() { _ = upstreamListener.Close() })
		upstreamConns := make(chan net.Conn, 4)
		go func() {
			for {
				conn, err := upstreamListener.Accept()
				if err != nil {
					return
				}
				upstreamConns <- conn
			}
		}()

		relay, _ := newRelay(t, upstreamListener.Addr().String(), nil)
		relay.CutAt(AfterResponseReachesClient, message.Read)

		clientConn, err := net.Dial("tcp", relay.address())
		Expect(err).NotTo(HaveOccurred(), "dialing the relay failed")
		t.Cleanup(func() { _ = clientConn.Close() })
		var upstream net.Conn
		Eventually(upstreamConns).WithTimeout(specWait).Should(Receive(&upstream),
			"the relay never dialed the fake upstream")

		request, err := readRequestWire(41)
		Expect(err).NotTo(HaveOccurred(), "encoding the Read request failed")
		response, err := readResponseWire(41)
		Expect(err).NotTo(HaveOccurred(), "encoding the Read response failed")
		_, err = clientConn.Write(request)
		Expect(err).NotTo(HaveOccurred(), "sending the Read request failed")

		go func() {
			_, _ = upstream.Read(make([]byte, len(request)))
			_, _ = upstream.Write(response)
		}()

		blobChunk, err := readRequestWire(41)
		Expect(err).NotTo(HaveOccurred(), "encoding the pipelined requests failed")
		blob := bytes.Repeat(blobChunk, 90*1024)
		writeDone := make(chan error, 1)
		go func() {
			_, err := clientConn.Write(blob)
			writeDone <- err
		}()

		received := make([]byte, len(response))
		_, err = io.ReadFull(clientConn, received)
		Expect(err).NotTo(HaveOccurred(),
			"the client did not read the full response although the cut must drain the client side first: %v", err)
		Expect(received).To(Equal(response), "the client received something other than the response")

		_, err = clientConn.Write([]byte("the client keeps sending"))
		Expect(err).NotTo(HaveOccurred(),
			"the client's write during the drain window failed, so the cut closed the client side before draining")

		end, err := clientConn.Read(make([]byte, 1))
		Expect(err).To(MatchError(io.EOF),
			"the client read %d bytes with error %v after the response, want EOF", end, err)
		Eventually(writeDone).WithTimeout(specWait).Should(Receive(Not(HaveOccurred())),
			"the client's in-flight write never completed, so the drain discarded nothing the relay should have consumed")
	})
})

// newInjectedRelay builds a relay whose recorder the injected-batch
// specs feed raw wire bytes, with a fake T that collects harness
// message.
func newInjectedRelay() (*Relay, *Recorder, *fakeT) {
	ft := &fakeT{}
	relay, recorder := newRelay(ft, "opc.tcp://127.0.0.1:1", nil)
	DeferCleanup(func() {
		for _, cleanup := range ft.cleanups {
			cleanup()
		}
	})
	return relay, recorder, ft
}

var _ = Describe("Relay CutAt injected batches", func() {

	It("fires a cut armed for Republish on an injected Republish request", func() {
		relay, recorder, _ := newInjectedRelay()
		relay.CutAt(BeforeRequestReachesServer, message.Republish)

		wire, err := republishRequestWire(21)
		Expect(err).NotTo(HaveOccurred(), "encoding the Republish failed")
		var written [][]byte
		finished := false
		Expect(recorder.forward(0, clientToServer, wire, func(message []byte) error {
			written = append(written, message)
			return nil
		}, func(armedCut) error {
			finished = true
			return nil
		})).To(Succeed(),
			"observing the Republish the cut fired on failed")

		requests := recorder.Requests()
		Expect(requests).To(HaveLen(1),
			"the Republish left %d request records, want 1", len(requests))
		Expect(requests[0].Fate).To(Equal(Dropped),
			"the Republish cut did not record the request it dropped as Dropped")
		message, ok := requests[0].Message()
		Expect(ok).To(BeTrue(),
			"the dropped Republish record does not yield its decoded message")
		_, isRepublish := message.(*ua.RepublishRequest)
		Expect(isRepublish).To(BeTrue(),
			"the dropped record holds a %T, want a *ua.RepublishRequest", message)
		Expect(written).To(BeEmpty(),
			"the relay wrote the Republish although the cut dropped it")
		Expect(finished).To(BeFalse(),
			"the relay finished an after-response cut although the cut dropped the request")
		Expect(relay.ArmedCuts()).To(BeEmpty(),
			"ArmedCuts still lists the cut after it fired")
	})

	It("records and writes nothing after the message a before-request cut fires on", func() {
		relay, recorder, _ := newInjectedRelay()
		relay.CutAt(BeforeRequestReachesServer, message.Read)

		first, err := readRequestWire(31)
		Expect(err).NotTo(HaveOccurred(), "encoding the first Read failed")
		second, err := readRequestWire(32)
		Expect(err).NotTo(HaveOccurred(), "encoding the second Read failed")
		var written [][]byte
		Expect(recorder.forward(0, clientToServer, append(first, second...), func(message []byte) error {
			written = append(written, message)
			return nil
		}, func(armedCut) error { return nil })).To(Succeed(),
			"observing the two Reads failed")

		requests := recorder.Requests()
		Expect(requests).To(HaveLen(1),
			"the cut fired on the first Read, so the second must leave no record: %v", requests)
		Expect(requests[0].RequestID).To(Equal(uint32(31)),
			"the recorded request is not the first Read the cut fired on")
		Expect(requests[0].Fate).To(Equal(Dropped),
			"the Read the cut fired on is not recorded as Dropped")
		Expect(written).To(BeEmpty(),
			"the relay wrote a message although the cut dropped the first Read")
		Expect(relay.ArmedCuts()).To(BeEmpty(),
			"ArmedCuts still lists the cut after it fired")
	})

	It("does not fire an after-response cut on a response whose request was never recorded", func() {
		relay, recorder, _ := newInjectedRelay()
		relay.CutAt(AfterResponseReachesClient, message.Read)
		finished := false

		response, err := readResponseWire(77)
		Expect(err).NotTo(HaveOccurred(), "encoding the response failed")
		Expect(recorder.forward(0, serverToClient, response, func([]byte) error { return nil }, func(armedCut) error {
			finished = true
			return nil
		})).To(Succeed(),
			"observing the response failed")

		Expect(finished).To(BeFalse(),
			"the cut fired on a response whose request the recorder never saw")
		responses := recorder.Responses()
		Expect(responses).To(HaveLen(1),
			"the response left %d response records, want 1", len(responses))
		Expect(responses[0].Fate).To(Equal(Forwarded),
			"the response the cut must ignore is not Forwarded")
		Expect(relay.ArmedCuts()).To(Equal([]string{
			"after a Read response reaches the client",
		}), "ArmedCuts no longer lists the cut although no recorded request matches the response")
	})

	It("does not fire an after-response cut on a response to a request of another connection", func() {
		relay, recorder, _ := newInjectedRelay()
		relay.CutAt(AfterResponseReachesClient, message.Read)
		finished := false

		staged, err := readRequestWire(9)
		Expect(err).NotTo(HaveOccurred(), "encoding the request failed")
		recorder.observe(1, clientToServer, staged)
		response, err := readResponseWire(9)
		Expect(err).NotTo(HaveOccurred(), "encoding the response failed")
		Expect(recorder.forward(0, serverToClient, response, func([]byte) error { return nil }, func(armedCut) error {
			finished = true
			return nil
		})).To(Succeed(),
			"observing the response failed")

		Expect(finished).To(BeFalse(),
			"the cut fired on a response to a request of another connection")
		responses := recorder.Responses()
		Expect(responses).To(HaveLen(1),
			"the response left %d response records, want 1", len(responses))
		Expect(responses[0].Fate).To(Equal(Forwarded),
			"the response the cut must ignore is not Forwarded")
		Expect(relay.ArmedCuts()).To(Equal([]string{
			"after a Read response reaches the client",
		}), "ArmedCuts no longer lists the cut although the matching request belongs to another connection")
	})

	It("does not fire an after-response cut on a response to a request of another service", func() {
		relay, recorder, _ := newInjectedRelay()
		relay.CutAt(AfterResponseReachesClient, message.Read)
		finished := false

		staged, err := republishRequestWire(9)
		Expect(err).NotTo(HaveOccurred(), "encoding the Republish failed")
		recorder.observe(0, clientToServer, staged)
		response, err := readResponseWire(9)
		Expect(err).NotTo(HaveOccurred(), "encoding the response failed")
		Expect(recorder.forward(0, serverToClient, response, func([]byte) error { return nil }, func(armedCut) error {
			finished = true
			return nil
		})).To(Succeed(),
			"observing the response failed")

		Expect(finished).To(BeFalse(),
			"the cut fired on a response to a request of another service")
		responses := recorder.Responses()
		Expect(responses).To(HaveLen(1),
			"the response left %d response records, want 1", len(responses))
		Expect(responses[0].Fate).To(Equal(Forwarded),
			"the response the cut must ignore is not Forwarded")
		Expect(relay.ArmedCuts()).To(Equal([]string{
			"after a Read response reaches the client",
		}), "ArmedCuts no longer lists the cut although the matching request belongs to another service")
	})

	It("fires an after-response cut on a ServiceFault answering the recorded request", func() {
		relay, recorder, _ := newInjectedRelay()
		relay.CutAt(AfterResponseReachesClient, message.Read)
		finished := false

		staged, err := readRequestWire(9)
		Expect(err).NotTo(HaveOccurred(), "encoding the request failed")
		recorder.observe(0, clientToServer, staged)
		fault, err := serviceFaultWire(9)
		Expect(err).NotTo(HaveOccurred(), "encoding the ServiceFault failed")
		var written [][]byte
		Expect(recorder.forward(0, serverToClient, fault, func(message []byte) error {
			written = append(written, message)
			return nil
		}, func(armedCut) error {
			finished = true
			return nil
		})).To(Succeed(),
			"observing the ServiceFault failed")

		Expect(written).To(Equal([][]byte{fault}),
			"the relay did not write the ServiceFault the cut fired on")
		Expect(finished).To(BeTrue(),
			"the relay did not finish the after-response cut after the ServiceFault")
		Expect(relay.ArmedCuts()).To(BeEmpty(),
			"ArmedCuts still lists the cut after it fired")
		responses := recorder.Responses()
		Expect(responses).To(HaveLen(1),
			"the ServiceFault left %d response records, want 1", len(responses))
		Expect(responses[0].Fate).To(Equal(Forwarded),
			"the cut did not record the ServiceFault it wrote to the client as Forwarded")
		message, ok := responses[0].Message()
		Expect(ok).To(BeTrue(),
			"the forwarded ServiceFault record does not yield its decoded message")
		_, isFault := message.(*ua.ServiceFault)
		Expect(isFault).To(BeTrue(),
			"the record holds a %T, want a *ua.ServiceFault", message)
	})

	It("writes the cut's response and records nothing after it in the same read", func() {
		relay, recorder, _ := newInjectedRelay()
		relay.CutAt(AfterResponseReachesClient, message.Read)
		var events []string

		staged, err := readRequestWire(9)
		Expect(err).NotTo(HaveOccurred(), "encoding the request failed")
		recorder.observe(0, clientToServer, staged)
		cutResponse, err := readResponseWire(9)
		Expect(err).NotTo(HaveOccurred(), "encoding the response failed")
		trailing, err := readResponseWire(10)
		Expect(err).NotTo(HaveOccurred(), "encoding the trailing response failed")
		var written [][]byte
		Expect(recorder.forward(0, serverToClient, append(cutResponse, trailing...), func(message []byte) error {
			written = append(written, message)
			events = append(events, "write")
			return nil
		}, func(armedCut) error {
			events = append(events, "halfClose", "cut")
			return nil
		})).To(Succeed(),
			"observing the two responses failed")

		Expect(written).To(Equal([][]byte{cutResponse}),
			"the relay wrote something other than the cut's response to the client")
		Expect(events).To(Equal([]string{"write", "halfClose", "cut"}),
			"the relay did not write the response, half-close the client side and cut in that order")
		Expect(relay.ArmedCuts()).To(BeEmpty(),
			"ArmedCuts still lists the cut after it fired")
		responses := recorder.Responses()
		Expect(responses).To(HaveLen(1),
			"the trailing response after the cut left a record although the relay never wrote it: %v", responses)
		Expect(responses[0].RequestID).To(Equal(uint32(9)),
			"the recorded response is not the one the cut fired on")
		Expect(responses[0].Fate).To(Equal(Forwarded),
			"the cut's response is not recorded as Forwarded")
	})

	It("reports a failed half-close as a harness fault and still finishes the cut", func() {
		relay, recorder, ft := newInjectedRelay()
		relay.CutAt(AfterResponseReachesClient, message.Read)
		finished := false

		staged, err := readRequestWire(9)
		Expect(err).NotTo(HaveOccurred(), "encoding the request failed")
		recorder.observe(0, clientToServer, staged)
		response, err := readResponseWire(9)
		Expect(err).NotTo(HaveOccurred(), "encoding the response failed")
		closeErr := errors.New("the spectest half-close failure")
		Expect(recorder.forward(0, serverToClient, response, func([]byte) error { return nil }, func(armedCut) error {
			finished = true
			return closeErr
		})).To(Succeed(),
			"the failed half-close made observing the response fail")

		Expect(finished).To(BeTrue(),
			"the relay did not finish the cut although the half-close failed")
		Expect(relay.ArmedCuts()).To(BeEmpty(),
			"ArmedCuts still lists the cut although the relay finished the cut")
		Expect(fatalPanics(func() { recorder.Responses() })).To(BeTrue(),
			"the failed half-close was not reported as a harness fault")
		Expect(ft.fatals).To(HaveLen(1),
			"the failed half-close was reported %d times, want once", len(ft.fatals))
		Expect(ft.fatals[0]).To(ContainSubstring("could not half-close"),
			"the harness fault does not report the failed half-close")
	})

	It("records observed messages without claiming armed cuts", func() {
		relay, recorder, _ := newInjectedRelay()
		relay.CutAt(BeforeRequestReachesServer, message.Read)

		request, err := readRequestWire(11)
		Expect(err).NotTo(HaveOccurred(), "encoding the request failed")
		recorder.observe(0, clientToServer, request)
		requests := recorder.Requests()
		Expect(requests).To(HaveLen(1),
			"the observed request left %d request records, want 1", len(requests))
		Expect(requests[0].Fate).To(Equal(Forwarded),
			"observing the request claimed the armed cut instead of only recording it")

		relay.CutAt(AfterResponseReachesClient, message.Read)
		staged, err := readRequestWire(12)
		Expect(err).NotTo(HaveOccurred(), "encoding the staged request failed")
		recorder.observe(0, clientToServer, staged)
		response, err := readResponseWire(12)
		Expect(err).NotTo(HaveOccurred(), "encoding the response failed")
		recorder.observe(0, serverToClient, response)
		responses := recorder.Responses()
		Expect(responses).To(HaveLen(1),
			"the observed response left %d response records, want 1", len(responses))
		Expect(responses[0].Fate).To(Equal(Forwarded),
			"observing the response claimed the armed cut instead of only recording it")

		Expect(relay.ArmedCuts()).To(Equal([]string{
			"before a Read request reaches the server",
			"after a Read response reaches the client",
		}), "ArmedCuts does not list both cuts although observing messages fires none of them")
	})

	It("releases a claimed drop cut when a write before the cut fails, so the next matching message fires it", func() {
		relay, recorder, _ := newInjectedRelay()
		relay.CutAt(BeforeRequestReachesServer, message.Read)

		ignored, err := republishRequestWire(31)
		Expect(err).NotTo(HaveOccurred(), "encoding the ignored Republish failed")
		matching, err := readRequestWire(32)
		Expect(err).NotTo(HaveOccurred(), "encoding the Read the cut fires on failed")
		writeErr := errors.New("the spectest write failure")
		Expect(recorder.forward(0, clientToServer, append(ignored, matching...), func([]byte) error {
			return writeErr
		}, func(armedCut) error { return nil })).To(MatchError(writeErr),
			"forward returned an error other than the write error")

		Expect(relay.ArmedCuts()).To(Equal([]string{
			"before a Read request reaches the server",
		}), "ArmedCuts no longer lists the cut although the failed write left its message undelivered")

		second, err := readRequestWire(33)
		Expect(err).NotTo(HaveOccurred(), "encoding the second Read failed")
		Expect(recorder.forward(0, clientToServer, second, func([]byte) error { return nil }, func(armedCut) error { return nil })).To(Succeed(),
			"observing the second Read failed")
		Expect(relay.ArmedCuts()).To(BeEmpty(),
			"the released cut did not fire on the next matching message")

		var dropped []uint32
		for _, r := range recorder.Requests() {
			if r.Fate == Dropped {
				dropped = append(dropped, r.RequestID)
			}
		}
		Expect(dropped).To(Equal([]uint32{32, 33}),
			"the released cut did not drop the Read of each batch it fired on")
	})

	It("releases a claimed after-response cut when the write of its response fails, so the next matching response fires it", func() {
		relay, recorder, _ := newInjectedRelay()
		relay.CutAt(AfterResponseReachesClient, message.Read)

		staged, err := readRequestWire(9)
		Expect(err).NotTo(HaveOccurred(), "encoding the request failed")
		recorder.observe(0, clientToServer, staged)
		response, err := readResponseWire(9)
		Expect(err).NotTo(HaveOccurred(), "encoding the response failed")
		writeErr := errors.New("the spectest write failure")
		finished := false
		Expect(recorder.forward(0, serverToClient, response, func([]byte) error {
			return writeErr
		}, func(armedCut) error {
			finished = true
			return nil
		})).To(MatchError(writeErr),
			"forward returned an error other than the write error")
		Expect(finished).To(BeFalse(),
			"the relay finished the after-response cut although writing the response failed")
		Expect(relay.ArmedCuts()).To(Equal([]string{
			"after a Read response reaches the client",
		}), "ArmedCuts no longer lists the cut although its response never reached the client")

		Expect(recorder.forward(0, serverToClient, response, func([]byte) error { return nil }, func(armedCut) error {
			finished = true
			return nil
		})).To(Succeed(),
			"observing the second response failed")
		Expect(finished).To(BeTrue(),
			"the released cut did not fire on the next matching response")
		Expect(relay.ArmedCuts()).To(BeEmpty(),
			"ArmedCuts still lists the cut after the released cut fired")
	})

	It("leaves the second of two identical armed cuts listed and armed while the first fires", func() {
		relay, recorder, _ := newInjectedRelay()
		relay.CutAt(BeforeRequestReachesServer, message.Read)
		relay.CutAt(BeforeRequestReachesServer, message.Read)

		first, err := readRequestWire(41)
		Expect(err).NotTo(HaveOccurred(), "encoding the first Read failed")
		Expect(recorder.forward(0, clientToServer, first, func([]byte) error { return nil }, func(armedCut) error { return nil })).To(Succeed(),
			"observing the first Read failed")
		Expect(relay.ArmedCuts()).To(Equal([]string{
			"before a Read request reaches the server",
		}), "firing the first cut removed the second identical cut as well")

		second, err := readRequestWire(42)
		Expect(err).NotTo(HaveOccurred(), "encoding the second Read failed")
		Expect(recorder.forward(0, clientToServer, second, func([]byte) error { return nil }, func(armedCut) error { return nil })).To(Succeed(),
			"observing the second Read failed")
		Expect(relay.ArmedCuts()).To(BeEmpty(),
			"the second identical cut never fired on its own matching message")

		var dropped []uint32
		for _, r := range recorder.Requests() {
			if r.Fate == Dropped {
				dropped = append(dropped, r.RequestID)
			}
		}
		Expect(dropped).To(Equal([]uint32{41, 42}),
			"the two identical cuts did not drop one Read each")
	})
})

func TestCutAtRejectsUnknownMomentAndMessage(t *testing.T) {
	ft := &fakeT{}
	relay, _ := newRelay(ft, "opc.tcp://127.0.0.1:1", nil)
	defer func() {
		for _, cleanup := range ft.cleanups {
			cleanup()
		}
	}()

	if !fatalPanics(func() { relay.CutAt(Moment(9), message.Read) }) {
		t.Fatalf("CutAt did not fail for an unknown Moment")
	}
	if len(ft.fatals) != 1 || !strings.HasPrefix(ft.fatals[0], "spectest:") || !strings.Contains(ft.fatals[0], "unknown Moment") {
		t.Fatalf("CutAt failed with %q, want the harness fault naming the unknown Moment", ft.fatals)
	}

	if !fatalPanics(func() { relay.CutAt(BeforeRequestReachesServer, message.Message(0)) }) {
		t.Fatalf("CutAt did not fail for an unknown Message")
	}
	if len(ft.fatals) != 2 || !strings.HasPrefix(ft.fatals[1], "spectest:") || !strings.Contains(ft.fatals[1], "unknown Message") {
		t.Fatalf("CutAt failed with %q, want the harness fault naming the unknown Message", ft.fatals)
	}

	if !fatalPanics(func() { relay.CutAt(AfterResponseReachesClient, message.CloseSecureChannel) }) {
		t.Fatalf("CutAt did not fail for a response moment on CloseSecureChannel")
	}
	if len(ft.fatals) != 3 || !strings.HasPrefix(ft.fatals[2], "spectest:") || !strings.Contains(ft.fatals[2], "CloseSecureChannel") {
		t.Fatalf("CutAt failed with %q, want the harness fault naming the response-less CloseSecureChannel", ft.fatals)
	}

	if !fatalPanics(func() { relay.CutAt(ResponseNeverReachesClient, message.CloseSecureChannel) }) {
		t.Fatalf("CutAt did not fail for the response-never moment on CloseSecureChannel")
	}
	if len(ft.fatals) != 4 || !strings.HasPrefix(ft.fatals[3], "spectest:") || !strings.Contains(ft.fatals[3], "CloseSecureChannel") {
		t.Fatalf("CutAt failed with %q, want the harness fault naming the response-less CloseSecureChannel", ft.fatals)
	}

	if cuts := relay.ArmedCuts(); len(cuts) != 0 {
		t.Fatalf("ArmedCuts lists %v although every CutAt call failed", cuts)
	}
}
