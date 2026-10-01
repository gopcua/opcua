package spectest

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"sync"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
)

// T is the testing handle spectest helpers use to fail the running
// spec and register cleanup functions.
type T interface {
	Helper()
	Fatalf(format string, args ...any)
	Cleanup(func())
}

// Fate says what the relay did with a service message.
type Fate int

const (
	// Forwarded marks a message the relay passed through unchanged.
	Forwarded Fate = iota
	// Dropped is reserved for the drop path a later rung adds;
	// nothing produces it yet.
	Dropped
	// Truncated marks bytes that never became a complete message
	// before their connection closed.
	Truncated
	// Aborted marks a message the sender cancelled with an abort
	// chunk.
	Aborted
)

// ServiceRecord is one service message the relay observed on one
// connection; only a Forwarded record holds its decoded message.
type ServiceRecord[M any] struct {
	Order      int
	Connection int
	RequestID  uint32
	Fate       Fate
	message    M
}

// Message returns the decoded service and whether the relay
// forwarded it.
func (r ServiceRecord[M]) Message() (M, bool) {
	if r.Fate != Forwarded {
		var zero M
		return zero, false
	}
	return r.message, true
}

// TransportType names one OPC UA transport message.
type TransportType int

const (
	// HEL is the client's first transport handshake message.
	HEL TransportType = iota
	// ACK is the server's answer to a HEL.
	ACK
	// ERR reports a transport-level failure.
	ERR
)

// TransportError is the payload an ERR transport message carries.
type TransportError struct {
	Status ua.StatusCode
	Reason string
}

// TransportRecord is one transport handshake or error message the
// relay observed.
type TransportRecord struct {
	Order      int
	Connection int
	Type       TransportType
	Err        *TransportError
}

type flow int

const (
	clientToServer flow = iota
	serverToClient
)

func (f flow) String() string {
	if f == serverToClient {
		return "server-to-client"
	}
	return "client-to-server"
}

type streamKey struct {
	connection int
	flow       flow
}

type chunkKey struct {
	connection int
	flow       flow
	requestID  uint32
}

type streamState struct {
	reassembler reassembleChunks
	rest        []byte
	consumed    int
}

type harnessError struct {
	connection int
	flow       flow
	offset     int
	err        error
}

func (e *harnessError) Error() string {
	return fmt.Sprintf("connection %d, %s direction, byte offset %d: %v", e.connection, e.flow, e.offset, e.err)
}

// Recorder collects every message the relay observes. All methods are
// safe for concurrent use and return copies.
type Recorder struct {
	t           T
	mu          sync.Mutex
	order       int
	requests    []ServiceRecord[ua.Request]
	responses   []ServiceRecord[ua.Response]
	transport   []TransportRecord
	log         []any
	streams     map[streamKey]*streamState
	chunkCounts map[chunkKey]int
	err         error
	reported    bool
}

func newRecorder(t T) *Recorder {
	recorder := &Recorder{
		t:           t,
		streams:     make(map[streamKey]*streamState),
		chunkCounts: make(map[chunkKey]int),
	}
	t.Cleanup(func() {
		recorder.mu.Lock()
		defer recorder.mu.Unlock()
		recorder.raiseLocked()
	})
	return recorder
}

// Requests returns a copy of the recorded client-to-server messages,
// oldest first.
func (r *Recorder) Requests() []ServiceRecord[ua.Request] {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.raiseLocked()
	return slices.Clone(r.requests)
}

// Responses returns a copy of the recorded server-to-client messages,
// oldest first.
func (r *Recorder) Responses() []ServiceRecord[ua.Response] {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.raiseLocked()
	return slices.Clone(r.responses)
}

// Transport returns a copy of the recorded transport messages, oldest
// first.
func (r *Recorder) Transport() []TransportRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.raiseLocked()
	return slices.Clone(r.transport)
}

// Log returns a copy of every recorded message, in Order.
func (r *Recorder) Log() []any {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.raiseLocked()
	return slices.Clone(r.log)
}

func (r *Recorder) raiseLocked() {
	if r.err != nil && !r.reported {
		r.reported = true
		r.t.Fatalf("spectest: %v", r.err)
	}
}

func (r *Recorder) nextOrderLocked() int {
	r.order++
	return r.order
}

func (r *Recorder) setErrorLocked(connection int, flow flow, offset int, err error) {
	if r.err == nil {
		r.err = &harnessError{connection: connection, flow: flow, offset: offset, err: err}
	}
}

func (r *Recorder) setRelayError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err == nil {
		r.err = err
	}
}

func (r *Recorder) observe(connection int, flow flow, data []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, _ = r.observeLocked(connection, flow, data)
}

// The write stays outside the lock so a peer that stops reading
// blocks only its own direction.
func (r *Recorder) forward(connection int, flow flow, data []byte, write func([]byte) error) error {
	r.mu.Lock()
	messages, err := r.observeLocked(connection, flow, data)
	r.mu.Unlock()
	for _, message := range messages {
		if writeErr := write(message); writeErr != nil {
			return writeErr
		}
	}
	return err
}

func (r *Recorder) observeClose(connection int, flow flow) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := streamKey{connection, flow}
	state := r.streams[key]
	if state == nil {
		return
	}
	for _, requestID := range state.reassembler.truncatedAtClose() {
		r.appendService(r.nextOrderLocked(), connection, flow, requestID, Truncated, nil)
	}
	if len(state.rest) > 0 {
		r.appendService(r.nextOrderLocked(), connection, flow, 0, Truncated, nil)
	}
	delete(r.streams, key)
}

func (r *Recorder) observeLocked(connection int, flow flow, data []byte) ([][]byte, error) {
	key := streamKey{connection, flow}
	state := r.streams[key]
	if state == nil {
		state = &streamState{}
		r.streams[key] = state
	}
	stream := append(state.rest, data...)
	messages, rest, err := splitMessages(stream)
	state.rest = bytes.Clone(rest)
	for _, message := range messages {
		offset := state.consumed
		state.consumed += len(message)
		order := r.nextOrderLocked()
		if recordErr := r.recordLocked(order, connection, flow, state, message); recordErr != nil {
			r.setErrorLocked(connection, flow, offset, recordErr)
		}
	}
	if err != nil {
		r.setErrorLocked(connection, flow, state.consumed, err)
		return messages, err
	}
	return messages, nil
}

func (r *Recorder) recordLocked(order, connection int, flow flow, state *streamState, message []byte) error {
	decoded, err := decodeFrame(message)
	if err != nil {
		return err
	}
	switch frame := decoded.(type) {
	case *transportFrame:
		record := transportRecord(order, connection, frame)
		r.transport = append(r.transport, record)
		r.log = append(r.log, record)
		return nil
	case *chunkFrame:
		r.chunkCounts[chunkKey{connection, flow, frame.requestID}]++
		assembled, complete, err := state.reassembler.add(frame)
		if err != nil {
			return err
		}
		if !complete {
			return nil
		}
		switch msg := assembled.(type) {
		case *completeMessage:
			if flowErr := serviceDirectionError(flow, msg.service); flowErr != nil {
				return flowErr
			}
			r.appendService(order, connection, flow, msg.requestID, Forwarded, msg.service)
			return nil
		case *abortedMessage:
			r.appendService(order, connection, flow, msg.requestID, Aborted, nil)
			return nil
		}
	}
	return nil
}

func serviceDirectionError(flow flow, service any) error {
	var ok bool
	if flow == clientToServer {
		_, ok = service.(ua.Request)
	} else {
		_, ok = service.(ua.Response)
	}
	if ok {
		return nil
	}
	return fmt.Errorf("a %T arrived on the %s direction", service, flow)
}

func transportRecord(order, connection int, frame *transportFrame) TransportRecord {
	record := TransportRecord{Order: order, Connection: connection}
	switch frame.messageType {
	case uacp.MessageTypeHello:
		record.Type = HEL
	case uacp.MessageTypeAcknowledge:
		record.Type = ACK
	case uacp.MessageTypeError:
		record.Type = ERR
		if body, ok := frame.body.(*uacp.Error); ok {
			record.Err = &TransportError{Status: ua.StatusCode(body.ErrorCode), Reason: body.Reason}
		}
	}
	return record
}

func (r *Recorder) appendService(order, connection int, flow flow, requestID uint32, fate Fate, service any) {
	if flow == clientToServer {
		request, _ := service.(ua.Request)
		record := ServiceRecord[ua.Request]{Order: order, Connection: connection, RequestID: requestID, Fate: fate, message: request}
		r.requests = append(r.requests, record)
		r.log = append(r.log, record)
		return
	}
	response, _ := service.(ua.Response)
	record := ServiceRecord[ua.Response]{Order: order, Connection: connection, RequestID: requestID, Fate: fate, message: response}
	r.responses = append(r.responses, record)
	r.log = append(r.log, record)
}

func (r *Recorder) chunkCount(connection int, flow flow, requestID uint32) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.chunkCounts[chunkKey{connection, flow, requestID}]
}

// Relay accepts client connections and forwards them to a fixed
// upstream server while a Recorder observes the traffic.
type Relay struct {
	listener    net.Listener
	mu          sync.Mutex
	connections []net.Conn
	count       int
	wg          sync.WaitGroup
	closed      bool
}

func (r *Relay) address() string {
	return r.listener.Addr().String()
}

// Cut closes both sides of every connection in r.connections, live or
// already closed. The listener stays open, so the next dial is accepted
// immediately.
func (r *Relay) Cut() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, conn := range r.connections {
		_ = conn.Close()
	}
	r.connections = nil
}

// ConnectionCount returns the number of connections the relay has
// accepted since it was created.
func (r *Relay) ConnectionCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.count
}

func newRelay(t T, upstream string) (*Relay, *Recorder) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("spectest: the relay could not listen on 127.0.0.1: %v", err)
	}
	relay := &Relay{listener: listener}
	relay.wg.Add(1)
	recorder := newRecorder(t)
	t.Cleanup(relay.close)
	go relay.acceptLoop(upstream, recorder)
	return relay, recorder
}

func (r *Relay) close() {
	_ = r.listener.Close()
	r.mu.Lock()
	r.closed = true
	r.mu.Unlock()
	r.Cut()
	r.wg.Wait()
}

func (r *Relay) acceptLoop(upstream string, recorder *Recorder) {
	defer r.wg.Done()
	addr := strings.TrimPrefix(upstream, "opc.tcp://")
	for {
		clientConn, err := r.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			recorder.setRelayError(fmt.Errorf("the relay stopped accepting client connections: %w", err))
			_ = r.listener.Close()
			return
		}
		serverConn, err := net.Dial("tcp", addr)
		if err != nil {
			_ = clientConn.Close()
			recorder.setRelayError(fmt.Errorf("the relay could not connect to the upstream server at %s: %w", addr, err))
			continue
		}
		r.mu.Lock()
		if r.closed {
			r.mu.Unlock()
			_ = clientConn.Close()
			_ = serverConn.Close()
			return
		}
		connection := r.count
		r.count++
		r.connections = append(r.connections, clientConn, serverConn)
		r.wg.Add(2)
		r.mu.Unlock()
		go r.pump(recorder, connection, clientToServer, clientConn, serverConn)
		go r.pump(recorder, connection, serverToClient, clientConn, serverConn)
	}
}

func (r *Relay) pump(recorder *Recorder, connection int, flow flow, clientConn, serverConn net.Conn) {
	defer r.wg.Done()
	defer recorder.observeClose(connection, flow)
	defer func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
	}()
	buffer := make([]byte, 64*1024)
	source, destination := clientConn, serverConn
	if flow == serverToClient {
		source, destination = serverConn, clientConn
	}
	for {
		n, err := source.Read(buffer)
		if n > 0 {
			if forwardErr := recorder.forward(connection, flow, buffer[:n], func(message []byte) error {
				_, err := destination.Write(message)
				return err
			}); forwardErr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}
