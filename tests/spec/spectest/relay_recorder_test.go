package spectest

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uasc"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const specNodeValue int32 = 4242
const specWait = 15 * time.Second

func startSpecServer() (string, *ua.NodeID, func()) {
	t := GinkgoT()
	var lastErr error
	for range 5 {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("spectest: reserving a free port failed: %v", err)
		}
		port := l.Addr().(*net.TCPAddr).Port
		if err := l.Close(); err != nil {
			t.Fatalf("spectest: releasing the reserved port failed: %v", err)
		}

		specServer := server.New(
			server.EndPoint("127.0.0.1", port),
			server.EnableSecurity("None", ua.MessageSecurityModeNone),
			server.EnableAuthMode(ua.UserTokenTypeAnonymous),
		)
		namespace := server.NewNodeNameSpace(specServer, "spectest")
		specServer.AddNamespace(namespace)
		nodeID := namespace.AddNewVariableStringNode("spectest_node", specNodeValue).ID()

		if err := specServer.Start(context.Background()); err != nil {
			lastErr = err
			_ = specServer.Close()
			continue
		}
		var stopOnce sync.Once
		stop := func() {
			stopOnce.Do(func() { _ = specServer.Close() })
		}
		t.Cleanup(stop)
		return fmt.Sprintf("opc.tcp://127.0.0.1:%d", port), nodeID, stop
	}
	t.Fatalf("spectest: the server could not start on a free port in 5 attempts, last error: %v", lastErr)
	return "", nil, nil
}

func connectThroughRelay(opts ...opcua.Option) (*opcua.Client, *Relay, *Recorder, *ua.NodeID) {
	t := GinkgoT()
	serverAddr, nodeID, _ := startSpecServer()
	relay, recorder := newRelay(t, serverAddr)

	clientOpts := append([]opcua.Option{
		opcua.SecurityMode(ua.MessageSecurityModeNone),
		opcua.AutoReconnect(true),
		opcua.ReconnectInterval(50 * time.Millisecond),
	}, opts...)
	client, err := opcua.NewClient("opc.tcp://"+relay.address(), clientOpts...)
	Expect(err).NotTo(HaveOccurred(), "creating the client failed")
	ctx, cancel := context.WithTimeout(context.Background(), specWait)
	defer cancel()
	Expect(client.Connect(ctx)).To(Succeed(), "the client never connected through the relay")
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), specWait)
		defer cancel()
		_ = client.Close(ctx)
	})
	return client, relay, recorder, nodeID
}

func requestTypeNames(records []ServiceRecord[ua.Request]) []string {
	names := make([]string, len(records))
	for i, r := range records {
		m, ok := r.Message()
		if !ok {
			names[i] = "not forwarded"
			continue
		}
		names[i] = fmt.Sprintf("%T", m)
	}
	return names
}

func readNode(client *opcua.Client, nodeID *ua.NodeID) *ua.ReadResponse {
	ctx, cancel := context.WithTimeout(context.Background(), specWait)
	defer cancel()
	resp, err := client.Read(ctx, &ua.ReadRequest{
		NodesToRead: []*ua.ReadValueID{{NodeID: nodeID, AttributeID: ua.AttributeIDValue}},
	})
	Expect(err).NotTo(HaveOccurred(), "the client's read of the node failed")
	return resp
}

func orderOf(entry any) int {
	switch e := entry.(type) {
	case ServiceRecord[ua.Request]:
		return e.Order
	case ServiceRecord[ua.Response]:
		return e.Order
	case TransportRecord:
		return e.Order
	}
	Fail(fmt.Sprintf("the log holds a %T, want a request, response or transport record", entry))
	return -1
}

type fakeT struct {
	fatals   []string
	cleanups []func()
}

func (f *fakeT) Helper() {}

func (f *fakeT) Fatalf(format string, args ...any) {
	f.fatals = append(f.fatals, fmt.Sprintf(format, args...))
}

func (f *fakeT) Cleanup(cleanup func()) {
	f.cleanups = append(f.cleanups, cleanup)
}

func garbageMessage() []byte {
	wire, err := messageChunk(uacp.ChunkTypeFinal, 42, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	Expect(err).NotTo(HaveOccurred(), "encoding a complete message whose body does not decode failed")
	return wire
}

func readRequestWire(requestID uint32) ([]byte, error) {
	typeID := ua.ServiceTypeID(&ua.ReadRequest{})
	if typeID == 0 {
		return nil, fmt.Errorf("ua.ServiceTypeID returned 0 for *ua.ReadRequest, want its registered type id")
	}
	body := ua.NewBuffer(nil)
	body.WriteStruct(ua.NewFourByteExpandedNodeID(0, typeID))
	body.WriteStruct(newReadRequest(requestID, ua.NewStringNodeID(1, "spectest.forward")))
	if body.Error() != nil {
		return nil, body.Error()
	}
	return messageChunk(uacp.ChunkTypeFinal, requestID, body.Bytes())
}

func readResponseWire(requestID uint32) ([]byte, error) {
	typeID := ua.ServiceTypeID(&ua.ReadResponse{})
	if typeID == 0 {
		return nil, fmt.Errorf("ua.ServiceTypeID returned 0 for *ua.ReadResponse, want its registered type id")
	}
	body := ua.NewBuffer(nil)
	body.WriteStruct(ua.NewFourByteExpandedNodeID(0, typeID))
	body.WriteStruct(&ua.ReadResponse{
		ResponseHeader: &ua.ResponseHeader{ServiceDiagnostics: &ua.DiagnosticInfo{}},
		Results:        []*ua.DataValue{},
	})
	if body.Error() != nil {
		return nil, body.Error()
	}
	return messageChunk(uacp.ChunkTypeFinal, requestID, body.Bytes())
}

var _ = Describe("Relay and Recorder", func() {
	It("records the session handshake and a read in order on one connection", func() {
		client, relay, recorder, nodeID := connectThroughRelay()
		_ = relay
		resp := readNode(client, nodeID)
		Expect(resp.Results).To(HaveLen(1))
		Expect(resp.Results[0].Value.Value()).To(Equal(any(specNodeValue)), "the client read the wrong value")

		Eventually(func(g Gomega) {
			var conn0 []ServiceRecord[ua.Request]
			for _, r := range recorder.Requests() {
				if r.Connection == 0 {
					conn0 = append(conn0, r)
				}
			}
			if len(conn0) < 4 {
				g.Expect(len(conn0)).To(BeNumerically(">=", 4),
					"connection 0 carries %d requests, want at least the session handshake and the node Read", len(conn0))
				return
			}
			g.Expect(requestTypeNames(conn0[:4])).To(Equal([]string{
				"*ua.OpenSecureChannelRequest",
				"*ua.CreateSessionRequest",
				"*ua.ActivateSessionRequest",
				"*ua.ReadRequest",
			}), "the first requests on connection 0 are not the session handshake followed by the node Read")
			for i := 1; i < len(conn0); i++ {
				g.Expect(conn0[i-1].Order).To(BeNumerically("<", conn0[i].Order),
					"Orders of requests %d and %d on connection 0 are not strictly increasing", i-1, i)
			}

			var readReq ServiceRecord[ua.Request]
			readFound := false
			for _, r := range conn0 {
				if m, ok := r.Message(); ok {
					if _, isRead := m.(*ua.ReadRequest); isRead {
						readReq = r
						readFound = true
					}
				}
			}
			g.Expect(readFound).To(BeTrue(), "no forwarded Read on connection 0")
			var nodeResp *ua.ReadResponse
			for _, r := range recorder.Responses() {
				if r.Connection != 0 || r.RequestID != readReq.RequestID {
					continue
				}
				m, ok := r.Message()
				g.Expect(ok).To(BeTrue(), "the response to the node Read is not a forwarded message")
				rr, isRead := m.(*ua.ReadResponse)
				g.Expect(isRead).To(BeTrue(), "the response to the node Read is %T", m)
				nodeResp = rr
			}
			g.Expect(nodeResp).NotTo(BeNil(), "no response on connection 0 carries the node Read's request id")
			g.Expect(nodeResp.Results).To(HaveLen(1))
			g.Expect(nodeResp.Results[0].Value.Value()).To(Equal(any(specNodeValue)), "the recorder saw the wrong value")
		}).WithTimeout(specWait).Should(Succeed())

		tampered := recorder.Requests()
		tampered[0].Connection = 99
		Expect(recorder.Requests()[0].Connection).To(Equal(0), "Requests() returned the recorder's internal slice")
	})

	It("interleaves both directions by Order and records HEL then ACK", func() {
		client, _, recorder, nodeID := connectThroughRelay()
		readNode(client, nodeID)

		log := recorder.Log()
		Expect(log).NotTo(BeEmpty())
		for i := 1; i < len(log); i++ {
			Expect(orderOf(log[i-1])).To(BeNumerically("<", orderOf(log[i])), "log entries %d and %d are not in Order", i-1, i)
		}

		transport := recorder.Transport()
		var hel, ack *TransportRecord
		for i := range transport {
			if transport[i].Connection != 0 {
				continue
			}
			switch transport[i].Type {
			case HEL:
				if hel == nil {
					hel = &transport[i]
				}
			case ACK:
				if ack == nil {
					ack = &transport[i]
				}
			}
		}
		Expect(hel).NotTo(BeNil(), "connection 0 shows no HEL")
		Expect(ack).NotTo(BeNil(), "connection 0 shows no ACK")
		Expect(hel.Err).To(BeNil())
		Expect(ack.Err).To(BeNil())
		Expect(hel.Order).To(BeNumerically("<", ack.Order), "the ACK does not follow the HEL in Order")

		requestOrders := map[uint32]int{}
		for _, r := range recorder.Requests() {
			if r.Connection == 0 {
				requestOrders[r.RequestID] = r.Order
			}
		}
		paired := 0
		for _, r := range recorder.Responses() {
			if r.Connection != 0 {
				continue
			}
			requestOrder, ok := requestOrders[r.RequestID]
			if !ok {
				continue
			}
			paired++
			Expect(r.Order).To(BeNumerically(">", requestOrder), "the response to request %d does not follow it in Order", r.RequestID)
		}
		Expect(paired).To(BeNumerically(">=", 3), "too few request-response pairs on connection 0")

		tamperedResponses := recorder.Responses()
		tamperedResponses[0].Connection = 99
		Expect(recorder.Responses()[0].Connection).To(Equal(0), "Responses() returned the recorder's internal slice")
		tamperedTransport := recorder.Transport()
		tamperedTransport[0].Connection = 99
		Expect(recorder.Transport()[0].Connection).To(Equal(0), "Transport() returned the recorder's internal slice")
	})

	It("records a WriteRequest that spans several chunks as one record", func() {
		client, _, recorder, nodeID := connectThroughRelay(opcua.SendBufferSize(8192))
		payload := make([]byte, 64*1024)
		for i := range payload {
			payload[i] = byte(i % 251)
		}
		variant, err := ua.NewVariant(payload)
		Expect(err).NotTo(HaveOccurred(), "building the 64 KiB ByteString variant failed")
		writeCtx, writeCancel := context.WithTimeout(context.Background(), specWait)
		defer writeCancel()
		writeResp, err := client.Write(writeCtx, &ua.WriteRequest{
			NodesToWrite: []*ua.WriteValue{{
				NodeID:      nodeID,
				AttributeID: ua.AttributeIDValue,
				Value:       &ua.DataValue{EncodingMask: ua.DataValueValue, Value: variant},
			}},
		})
		Expect(err).NotTo(HaveOccurred(), "the chunked Write failed at transport level")
		Expect(writeResp).NotTo(BeNil())
		Expect(writeResp.Results).To(HaveLen(1), "the server answered the chunked Write with %d results, want 1", len(writeResp.Results))
		Expect(writeResp.Results[0]).To(Equal(ua.StatusGood), "the server rejected the chunked Write")

		var writes []ServiceRecord[ua.Request]
		for _, r := range recorder.Requests() {
			if r.Connection != 0 {
				continue
			}
			if m, ok := r.Message(); ok {
				if _, isWrite := m.(*ua.WriteRequest); isWrite {
					writes = append(writes, r)
				}
			}
		}
		Expect(writes).To(HaveLen(1), "the recorder saw %d WriteRequests, want exactly 1", len(writes))
		write := writes[0]
		Expect(recorder.chunkCount(0, clientToServer, write.RequestID)).To(BeNumerically(">=", 2),
			"the WriteRequest was sent in %d chunks, want at least 2", recorder.chunkCount(0, clientToServer, write.RequestID))
		m, ok := write.Message()
		Expect(ok).To(BeTrue())
		writeMessage, isWrite := m.(*ua.WriteRequest)
		Expect(isWrite).To(BeTrue(), "the WriteRequest record holds a %T", m)
		Expect(writeMessage.NodesToWrite).To(HaveLen(1))
		Expect(writeMessage.NodesToWrite[0].Value.Value.Value()).To(Equal(any(payload)),
			"the recorded WriteRequest does not hold the value the client wrote")
	})

	It("cuts the connection and the client reconnects on a new connection", func() {
		client, relay, recorder, nodeID := connectThroughRelay()
		readNode(client, nodeID)
		Expect(relay.ConnectionCount()).To(Equal(1))
		relay.Cut()

		Eventually(func(g Gomega) {
			g.Expect(relay.ConnectionCount()).To(Equal(2), "the client did not reconnect after the cut")
			activated := false
			for _, r := range recorder.Requests() {
				if r.Connection != 1 {
					continue
				}
				if m, ok := r.Message(); ok {
					if _, isActivate := m.(*ua.ActivateSessionRequest); isActivate {
						activated = true
					}
				}
			}
			g.Expect(activated).To(BeTrue(), "connection 1 carries no ActivateSession")
		}).WithTimeout(15 * time.Second).Should(Succeed())

		resp := readNode(client, nodeID)
		Expect(resp.Results[0].Value.Value()).To(Equal(any(specNodeValue)), "the client cannot read through the reconnected relay")
	})

	It("marks bytes a cut leaves behind as Truncated through the live relay", func() {
		t := GinkgoT()
		serverAddr, _, _ := startSpecServer()
		relay, recorder := newRelay(t, serverAddr)

		firstConn, err := net.Dial("tcp", relay.address())
		Expect(err).NotTo(HaveOccurred(), "dialing the relay failed")
		defer func() { _ = firstConn.Close() }()
		firstPending, err := messageChunk(uacp.ChunkTypeIntermediate, 7, []byte{1, 2, 3})
		Expect(err).NotTo(HaveOccurred(), "encoding the first intermediate chunk failed")
		_, err = firstConn.Write(firstPending)
		Expect(err).NotTo(HaveOccurred(), "sending the first intermediate chunk failed")
		Eventually(func() int { return recorder.chunkCount(0, clientToServer, 7) }).WithTimeout(specWait).Should(Equal(1),
			"the relay never observed the first intermediate chunk")
		relay.Cut()

		Eventually(func(g Gomega) {
			var found bool
			for _, record := range recorder.Requests() {
				if record.Connection != 0 || record.RequestID != 7 {
					continue
				}
				found = true
				g.Expect(record.Fate).To(Equal(Truncated), "the partial request the cut left behind is not Truncated")
				_, ok := record.Message()
				g.Expect(ok).To(BeFalse(), "a Truncated record must not yield its message")
			}
			g.Expect(found).To(BeTrue(), "the cut left no Truncated record for the partial request")
		}).WithTimeout(specWait).Should(Succeed())

		secondConn, err := net.Dial("tcp", relay.address())
		Expect(err).NotTo(HaveOccurred(), "dialing the relay after the cut failed")
		defer func() { _ = secondConn.Close() }()
		secondPending, err := messageChunk(uacp.ChunkTypeIntermediate, 8, []byte{4, 5, 6})
		Expect(err).NotTo(HaveOccurred(), "encoding the second intermediate chunk failed")
		_, err = secondConn.Write(secondPending)
		Expect(err).NotTo(HaveOccurred(), "sending the second intermediate chunk failed")
		Eventually(func() int { return recorder.chunkCount(1, clientToServer, 8) }).WithTimeout(specWait).Should(Equal(1),
			"the relay never observed the second intermediate chunk")
		relay.Cut()

		Eventually(func(g Gomega) {
			var found bool
			for _, record := range recorder.Requests() {
				if record.Connection != 1 {
					continue
				}
				g.Expect(record.RequestID).NotTo(Equal(uint32(7)), "connection 1 reuses the partial request of connection 0")
				if record.RequestID != 8 {
					continue
				}
				found = true
				g.Expect(record.Fate).To(Equal(Truncated), "the second partial request is not Truncated")
			}
			g.Expect(found).To(BeTrue(), "the second cut left no Truncated record for connection 1")
		}).WithTimeout(specWait).Should(Succeed())
	})
})

var _ = Describe("Recorder injected bytes", func() {
	It("fails the next accessor after a message it cannot decode", func() {
		ft := &fakeT{}
		recorder := newRecorder(ft)
		recorder.observe(0, clientToServer, garbageMessage())

		recorder.Requests()
		Expect(ft.fatals).To(HaveLen(1), "Requests() did not fail exactly once")
		Expect(ft.fatals[0]).To(HavePrefix("spectest: connection 0, client-to-server direction, byte offset 0:"))

		for _, cleanup := range ft.cleanups {
			cleanup()
		}
		Expect(ft.fatals).To(HaveLen(1), "the cleanup failed again although an accessor already reported the error")
	})

	It("fails the cleanup when no accessor reported the undecodable message", func() {
		ft := &fakeT{}
		recorder := newRecorder(ft)
		recorder.observe(0, clientToServer, garbageMessage())

		for _, cleanup := range ft.cleanups {
			cleanup()
		}
		Expect(ft.fatals).To(HaveLen(1), "the cleanup did not fail on the undecodable message")
		Expect(ft.fatals[0]).To(HavePrefix("spectest: connection 0, client-to-server direction, byte offset 0:"))
	})

	It("records a truncated tail at close as Truncated, never as an error", func() {
		ft := &fakeT{}
		recorder := newRecorder(ft)
		pending, err := messageChunk(uacp.ChunkTypeIntermediate, 7, []byte{1, 2, 3})
		Expect(err).NotTo(HaveOccurred(), "encoding the intermediate chunk failed")
		recorder.observe(0, clientToServer, pending)
		recorder.observe(0, clientToServer, []byte("HELF\x00\x00\x00"))
		recorder.observeClose(0, clientToServer)

		reqs := recorder.Requests()
		Expect(ft.fatals).To(BeEmpty(), "a truncated tail raised a harness error")
		Expect(reqs).To(HaveLen(2))
		Expect(reqs[0].Fate).To(Equal(Truncated))
		Expect(reqs[0].RequestID).To(Equal(uint32(7)))
		Expect(reqs[1].Fate).To(Equal(Truncated))
		Expect(reqs[1].RequestID).To(Equal(uint32(0)))
		Expect(reqs[0].Order).To(BeNumerically("<", reqs[1].Order))
		for _, r := range reqs {
			_, ok := r.Message()
			Expect(ok).To(BeFalse(), "a Truncated record must not yield its message")
		}
		Expect(recorder.Log()).To(HaveLen(2), "the truncated records are missing from the log")
	})

	It("records an abort chunk as an Aborted message", func() {
		ft := &fakeT{}
		recorder := newRecorder(ft)
		abortBody, err := (&uasc.MessageAbort{ErrorCode: uint32(ua.StatusBadTimeout), Reason: "spectest abort"}).Encode()
		Expect(err).NotTo(HaveOccurred(), "encoding the abort body failed")
		abortWire, err := messageChunk(uacp.ChunkTypeAbort, 42, abortBody)
		Expect(err).NotTo(HaveOccurred(), "encoding the abort chunk failed")
		recorder.observe(0, clientToServer, abortWire)

		reqs := recorder.Requests()
		Expect(ft.fatals).To(BeEmpty())
		Expect(reqs).To(HaveLen(1))
		Expect(reqs[0].Fate).To(Equal(Aborted))
		Expect(reqs[0].RequestID).To(Equal(uint32(42)))
		_, ok := reqs[0].Message()
		Expect(ok).To(BeFalse(), "an Aborted record must not yield its message")
	})

	It("records an injected ERR in Transport with its status and reason", func() {
		ft := &fakeT{}
		recorder := newRecorder(ft)
		errorBody, err := (&uacp.Error{ErrorCode: uint32(ua.StatusBadTimeout), Reason: "the connection timed out"}).Encode()
		Expect(err).NotTo(HaveOccurred(), "encoding the ERR body failed")
		errorWire, err := transportMessage(uacp.MessageTypeError, errorBody)
		Expect(err).NotTo(HaveOccurred(), "encoding the ERR message failed")
		recorder.observe(0, serverToClient, errorWire)

		transport := recorder.Transport()
		Expect(ft.fatals).To(BeEmpty())
		Expect(transport).To(HaveLen(1))
		Expect(transport[0].Type).To(Equal(ERR))
		Expect(transport[0].Connection).To(Equal(0))
		Expect(transport[0].Err).NotTo(BeNil())
		Expect(transport[0].Err.Status).To(Equal(ua.StatusBadTimeout))
		Expect(transport[0].Err.Reason).To(Equal("the connection timed out"))
	})

	It("fails the next accessor when a request arrives on the server-to-client direction", func() {
		ft := &fakeT{}
		recorder := newRecorder(ft)
		requestWire, err := readRequestWire(5)
		Expect(err).NotTo(HaveOccurred(), "encoding the request failed")
		recorder.observe(0, serverToClient, requestWire)

		recorder.Responses()
		Expect(ft.fatals).To(HaveLen(1), "Responses() did not fail on the request traveling the wrong direction")
		Expect(ft.fatals[0]).To(HavePrefix("spectest: connection 0, server-to-client direction, byte offset 0:"))
		Expect(ft.fatals[0]).To(ContainSubstring("*ua.ReadRequest"))

		for _, cleanup := range ft.cleanups {
			cleanup()
		}
		Expect(ft.fatals).To(HaveLen(1), "the cleanup failed again although an accessor already reported the error")
	})
})

var _ = Describe("Recorder per-direction reassembly", func() {
	It("reassembles each direction separately, so one request id is Truncated one way and complete the other", func() {
		ft := &fakeT{}
		recorder := newRecorder(ft)
		intermediate, err := messageChunk(uacp.ChunkTypeIntermediate, 9, []byte{1, 2, 3})
		Expect(err).NotTo(HaveOccurred(), "encoding the intermediate chunk failed")
		final, err := readResponseWire(9)
		Expect(err).NotTo(HaveOccurred(), "encoding the final response chunk failed")
		recorder.observe(0, clientToServer, intermediate)
		recorder.observe(0, serverToClient, final)
		recorder.observeClose(0, clientToServer)
		recorder.observeClose(0, serverToClient)

		responses := recorder.Responses()
		Expect(ft.fatals).To(BeEmpty(), "per-direction reassembly raised a harness error")
		Expect(responses).To(HaveLen(1), "the final chunk on the server-to-client direction did not reassemble")
		Expect(responses[0].RequestID).To(Equal(uint32(9)))
		Expect(responses[0].Fate).To(Equal(Forwarded))
		message, ok := responses[0].Message()
		Expect(ok).To(BeTrue(), "the reassembled response is not forwarded")
		readResponse, isRead := message.(*ua.ReadResponse)
		Expect(isRead).To(BeTrue(), "the reassembled response is a %T, want a *ua.ReadResponse", message)
		Expect(readResponse.Results).To(BeEmpty())

		requests := recorder.Requests()
		Expect(requests).To(HaveLen(1), "the intermediate chunk on the client-to-server direction left no Truncated record")
		Expect(requests[0].RequestID).To(Equal(uint32(9)))
		Expect(requests[0].Fate).To(Equal(Truncated))
		_, ok = requests[0].Message()
		Expect(ok).To(BeFalse(), "a Truncated record must not yield its message")
	})
})

var _ = Describe("Relay close propagation", func() {
	It("closes the other side of a connection when one side closes", func() {
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

		relay, _ := newRelay(t, upstreamListener.Addr().String())
		clientConn, err := net.Dial("tcp", relay.address())
		Expect(err).NotTo(HaveOccurred(), "dialing the relay failed")
		var upstream net.Conn
		Eventually(upstreamConns).WithTimeout(specWait).Should(Receive(&upstream), "the relay never dialed the fake upstream")

		Expect(upstream.Close()).To(Succeed())
		clientClosed := make(chan error, 1)
		go func() {
			_, err := clientConn.Read(make([]byte, 1))
			clientClosed <- err
		}()
		var clientReadErr error
		Eventually(clientClosed).WithTimeout(specWait).Should(Receive(&clientReadErr), "the client connection never saw the upstream close")
		Expect(clientReadErr).To(HaveOccurred(), "the client connection is still readable after the upstream closed")

		secondClient, err := net.Dial("tcp", relay.address())
		Expect(err).NotTo(HaveOccurred(), "dialing the relay a second time failed")
		var secondUpstream net.Conn
		Eventually(upstreamConns).WithTimeout(specWait).Should(Receive(&secondUpstream), "the relay never dialed the fake upstream a second time")

		Expect(secondClient.Close()).To(Succeed())
		upstreamClosed := make(chan error, 1)
		go func() {
			_, err := secondUpstream.Read(make([]byte, 1))
			upstreamClosed <- err
		}()
		var upstreamReadErr error
		Eventually(upstreamClosed).WithTimeout(specWait).Should(Receive(&upstreamReadErr), "the fake upstream never saw the client close")
		Expect(upstreamReadErr).To(HaveOccurred(), "the upstream connection is still readable after the client closed")
	})
})

var _ = Describe("Relay upstream faults", func() {
	It("keeps accepting after a failed upstream dial and reports it as a harness fault", func() {
		portListener, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred(), "reserving the upstream port failed")
		upstreamAddress := portListener.Addr().String()
		Expect(portListener.Close()).To(Succeed())

		ft := &fakeT{}
		relay, recorder := newRelay(ft, upstreamAddress)

		firstConn, err := net.Dial("tcp", relay.address())
		Expect(err).NotTo(HaveOccurred(), "dialing the relay failed")
		firstClosed := make(chan error, 1)
		go func() {
			_, err := firstConn.Read(make([]byte, 1))
			firstClosed <- err
		}()
		var firstErr error
		Eventually(firstClosed).WithTimeout(specWait).Should(Receive(&firstErr), "the client connection was not closed after the failed upstream dial")
		Expect(firstErr).To(HaveOccurred(), "the client connection is still readable after the failed upstream dial")

		secondConn, err := net.Dial("tcp", relay.address())
		Expect(err).NotTo(HaveOccurred(), "the relay stopped accepting after a failed upstream dial")
		secondClosed := make(chan error, 1)
		go func() {
			_, err := secondConn.Read(make([]byte, 1))
			secondClosed <- err
		}()
		var secondErr error
		Eventually(secondClosed).WithTimeout(specWait).Should(Receive(&secondErr), "the second client connection was not closed after the failed upstream dial")
		Expect(secondErr).To(HaveOccurred(), "the second client connection is still readable after the failed upstream dial")

		recorder.Requests()
		Expect(ft.fatals).To(HaveLen(1), "the failed upstream dial did not reach the harness error")
		Expect(ft.fatals[0]).To(HavePrefix("spectest:"))
		Expect(ft.fatals[0]).To(ContainSubstring(upstreamAddress))

		for _, cleanup := range ft.cleanups {
			cleanup()
		}
	})
})
