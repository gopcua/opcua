package spectest

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
)

// T is the testing handle spectest helpers use to fail the running
// spec and register cleanup functions. Fatalf must not return: it
// stops the running spec.
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
	// Dropped marks a request a cut CutAt armed discarded instead of
	// forwarding it; the record still holds its decoded message.
	Dropped
	// Truncated marks bytes left over when a connection closes: a
	// message that never became complete.
	Truncated
	// Aborted marks a message the sender cancelled with an abort
	// chunk.
	Aborted
)

// ServiceRecord is one service message the relay observed on one
// connection; a Forwarded or Dropped record holds its decoded
// message, the other Fates hold none.
type ServiceRecord[M any] struct {
	Order      int
	Connection int
	RequestID  uint32
	Fate       Fate
	message    M
}

// Message returns the decoded service and whether the relay
// forwarded it or a cut CutAt armed dropped it.
func (r ServiceRecord[M]) Message() (M, bool) {
	if r.Fate != Forwarded && r.Fate != Dropped {
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

// ConnectionState says whether the relay connection a server-side
// address belongs to is still open.
type ConnectionState int

const (
	// Unknown means the relay never accepted a connection with that
	// address.
	Unknown ConnectionState = iota
	// Open means the relay still forwards the connection the address
	// belongs to.
	Open
	// Closed means the relay closed the connection the address
	// belongs to.
	Closed
)

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

type connectionEntry struct {
	index  int
	closed bool
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
	relay       *Relay
	mu          sync.Mutex
	order       int
	requests    []ServiceRecord[ua.Request]
	responses   []ServiceRecord[ua.Response]
	transport   []TransportRecord
	log         []any
	streams     map[streamKey]*streamState
	chunkCounts map[chunkKey]int
	connections map[string]*connectionEntry
	err         error
	reported    bool
}

func newRecorder(t T) *Recorder {
	recorder := &Recorder{
		t:           t,
		streams:     make(map[streamKey]*streamState),
		chunkCounts: make(map[chunkKey]int),
		connections: make(map[string]*connectionEntry),
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

// RequestsSince returns a copy of the recorded client-to-server
// messages with a higher Order than m captured, oldest first.
func (r *Recorder) RequestsSince(m Mark) []ServiceRecord[ua.Request] {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.raiseLocked()
	return sinceOrder(r.requests, m.order)
}

// ResponsesSince returns a copy of the recorded server-to-client
// messages with a higher Order than m captured, oldest first.
func (r *Recorder) ResponsesSince(m Mark) []ServiceRecord[ua.Response] {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.raiseLocked()
	return sinceOrder(r.responses, m.order)
}

func sinceOrder[M any](records []ServiceRecord[M], order int) []ServiceRecord[M] {
	var since []ServiceRecord[M]
	for _, record := range records {
		if record.Order > order {
			since = append(since, record)
		}
	}
	return since
}

// ConnectionOf maps a server-side address, what the server's
// SecureChannel.RemoteAddr returns, to its relay connection index and
// state.
func (r *Recorder) ConnectionOf(addr net.Addr) (int, ConnectionState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, known := r.connections[addr.String()]
	if !known {
		return 0, Unknown
	}
	if entry.closed {
		return entry.index, Closed
	}
	return entry.index, Open
}

func (r *Recorder) position() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.order
}

func (r *Recorder) observeConnection(index int, serverAddr net.Addr) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.connections[serverAddr.String()] = &connectionEntry{index: index}
}

func (r *Recorder) observeConnectionClosed(index int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, entry := range r.connections {
		if entry.index == index {
			entry.closed = true
		}
	}
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
	_, _, _ = r.observeLocked(connection, flow, data, false)
}

// The write stays outside the lock so a peer that stops reading
// blocks only its own direction.
func (r *Recorder) forward(connection int, flow flow, data []byte, write func([]byte) error, finishAfterResponse func() error) error {
	r.mu.Lock()
	messages, cut, err := r.observeLocked(connection, flow, data, true)
	r.mu.Unlock()
	for i, message := range messages {
		if cut.index == i && cut.drop {
			r.relay.closeConnection(connection)
			return err
		}
		if writeErr := write(message); writeErr != nil {
			return writeErr
		}
		if cut.index == i {
			if finishErr := finishAfterResponse(); finishErr != nil {
				r.setRelayError(fmt.Errorf("spectest: the relay could not half-close the client side after the response it wrote: %w", finishErr))
			}
			return err
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

func (r *Recorder) observeLocked(connection int, flow flow, data []byte, claimCuts bool) ([][]byte, firedCut, error) {
	key := streamKey{connection, flow}
	state := r.streams[key]
	if state == nil {
		state = &streamState{}
		r.streams[key] = state
	}
	stream := append(state.rest, data...)
	messages, rest, splitErr := splitMessages(stream)
	state.rest = bytes.Clone(rest)
	cut := firedCut{index: -1}
	for i, message := range messages {
		offset := state.consumed
		state.consumed += len(message)
		order := r.nextOrderLocked()
		dropped, afterResponse, recordErr := r.recordLocked(order, connection, flow, state, message, claimCuts)
		if recordErr != nil {
			r.setErrorLocked(connection, flow, offset, recordErr)
		}
		if (dropped || afterResponse) && cut.index < 0 {
			cut = firedCut{index: i, drop: dropped}
			messages = messages[:i+1]
			break
		}
	}
	if splitErr != nil {
		r.setErrorLocked(connection, flow, state.consumed, splitErr)
		return messages, cut, splitErr
	}
	return messages, cut, nil
}

func (r *Recorder) recordLocked(order, connection int, flow flow, state *streamState, message []byte, claimCuts bool) (dropped bool, afterResponse bool, err error) {
	decoded, err := decodeFrame(message)
	if err != nil {
		return false, false, err
	}
	switch frame := decoded.(type) {
	case *transportFrame:
		record := transportRecord(order, connection, frame)
		r.transport = append(r.transport, record)
		r.log = append(r.log, record)
		return false, false, nil
	case *chunkFrame:
		r.chunkCounts[chunkKey{connection, flow, frame.requestID}]++
		assembled, complete, err := state.reassembler.add(frame)
		if err != nil {
			return false, false, err
		}
		if !complete {
			return false, false, nil
		}
		switch msg := assembled.(type) {
		case *completeMessage:
			if flowErr := serviceDirectionError(flow, msg.service); flowErr != nil {
				return false, false, flowErr
			}
			if claimCuts && flow == clientToServer && r.relay.takeArmed(func(c armedCut) bool {
				return c.moment == BeforeRequestReachesServer && c.service.matches(msg.service)
			}) {
				r.appendService(order, connection, flow, msg.requestID, Dropped, msg.service)
				return true, false, nil
			}
			if claimCuts && flow == serverToClient && r.relay.takeArmed(func(c armedCut) bool {
				return c.moment == AfterResponseReachesClient && r.hasRecordedRequest(connection, msg.requestID, c.service)
			}) {
				r.appendService(order, connection, flow, msg.requestID, Forwarded, msg.service)
				return false, true, nil
			}
			r.appendService(order, connection, flow, msg.requestID, Forwarded, msg.service)
			return false, false, nil
		case *abortedMessage:
			r.appendService(order, connection, flow, msg.requestID, Aborted, nil)
			return false, false, nil
		}
	}
	return false, false, nil
}

func (r *Recorder) hasRecordedRequest(connection int, requestID uint32, service Service) bool {
	for _, record := range r.requests {
		if record.Connection != connection || record.RequestID != requestID {
			continue
		}
		if record.message != nil && service.matches(record.message) {
			return true
		}
	}
	return false
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

// Moment names the point in a request's round trip a cut CutAt
// armed waits for before it fires.
type Moment int

const (
	// BeforeRequestReachesServer fires while the relay still holds the
	// matching request, before it writes the request upstream.
	BeforeRequestReachesServer Moment = iota
	// AfterResponseReachesClient fires once the matching response is
	// complete, after the relay wrote it to the client.
	AfterResponseReachesClient
)

// Service names the kind of client request an armed cut fires on.
type Service int

const (
	// Republish is the client's RepublishRequest.
	Republish Service = iota
	// Read is the client's ReadRequest.
	Read
)

func (s Service) name() string {
	switch s {
	case Republish:
		return "Republish"
	case Read:
		return "Read"
	}
	return ""
}

type armedCut struct {
	moment  Moment
	service Service
}

type firedCut struct {
	index int
	drop  bool
}

func (c armedCut) describe() string {
	if c.moment == AfterResponseReachesClient {
		return fmt.Sprintf("after a %s response reaches the client", c.service.name())
	}
	return fmt.Sprintf("before a %s request reaches the server", c.service.name())
}

func (s Service) matches(message any) bool {
	switch s {
	case Republish:
		_, ok := message.(*ua.RepublishRequest)
		return ok
	case Read:
		_, ok := message.(*ua.ReadRequest)
		return ok
	}
	return false
}

// Relay accepts client connections and forwards them to a fixed
// upstream server while a Recorder observes the traffic.
type Relay struct {
	listener    net.Listener
	recorder    *Recorder
	t           T
	mu          sync.Mutex
	connections []relayConn
	count       int
	wg          sync.WaitGroup
	closed      bool
	onCut       func()
	armed       []armedCut
	draining    map[int]bool
}

type relayConn struct {
	index  int
	client net.Conn
	server net.Conn
}

func (r *Relay) address() string {
	return r.listener.Addr().String()
}

// Cut closes both sides of every live connection. The listener stays
// open, so the next dial is accepted immediately.
func (r *Relay) Cut() {
	r.mu.Lock()
	conns := r.connections
	r.connections = nil
	r.mu.Unlock()
	for _, conn := range conns {
		_ = conn.client.Close()
		_ = conn.server.Close()
		r.recorder.observeConnectionClosed(conn.index)
	}
	if r.onCut != nil {
		r.onCut()
	}
}

// CutAt arms one cut: a Moment and a Service the cut fires on, on
// the next message that matches both. A cut armed for
// BeforeRequestReachesServer drops the matching request, records it
// Dropped and closes the connection it rode on. A cut armed for
// AfterResponseReachesClient writes the matching response to the
// client, half-closes the client side, closes the server side and
// keeps reading and discarding what the client sends for 2 s, then
// closes the client side.
func (r *Relay) CutAt(moment Moment, service Service) {
	if moment != BeforeRequestReachesServer && moment != AfterResponseReachesClient {
		r.t.Fatalf("spectest: CutAt received an unknown Moment %d", int(moment))
		return
	}
	if service != Republish && service != Read {
		r.t.Fatalf("spectest: CutAt received an unknown Service %d", int(service))
		return
	}
	r.mu.Lock()
	r.armed = append(r.armed, armedCut{moment: moment, service: service})
	r.mu.Unlock()
}

// ArmedCuts returns a readable description of every cut CutAt armed
// and has not fired yet, in arming order.
func (r *Relay) ArmedCuts() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	descriptions := make([]string, len(r.armed))
	for i, cut := range r.armed {
		descriptions[i] = cut.describe()
	}
	return descriptions
}

func (r *Relay) takeArmed(match func(armedCut) bool) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, cut := range r.armed {
		if match(cut) {
			r.armed = append(r.armed[:i], r.armed[i+1:]...)
			return true
		}
	}
	return false
}

func (r *Relay) closeConnection(index int) {
	r.mu.Lock()
	var kept []relayConn
	var conn relayConn
	found := false
	for _, c := range r.connections {
		if c.index == index {
			conn = c
			found = true
			continue
		}
		kept = append(kept, c)
	}
	r.connections = kept
	r.mu.Unlock()
	if found {
		_ = conn.client.Close()
		_ = conn.server.Close()
		r.recorder.observeConnectionClosed(index)
	}
	if r.onCut != nil {
		r.onCut()
	}
}

const clientDrainTimeout = 2 * time.Second

func (r *Relay) cutAfterResponse(connection int, clientConn, serverConn net.Conn) error {
	r.mu.Lock()
	r.draining[connection] = true
	r.mu.Unlock()
	closeErr := halfClose(clientConn)
	_ = serverConn.Close()
	if closeErr == nil {
		drainClient(clientConn)
	}
	_ = clientConn.Close()
	r.closeConnection(connection)
	return closeErr
}

func drainClient(clientConn net.Conn) {
	_ = clientConn.SetReadDeadline(time.Now().Add(clientDrainTimeout))
	buffer := make([]byte, 4096)
	for {
		if _, err := clientConn.Read(buffer); err != nil {
			return
		}
	}
}

func halfClose(conn net.Conn) error {
	return conn.(*net.TCPConn).CloseWrite()
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
	relay := &Relay{listener: listener, t: t, draining: make(map[int]bool)}
	relay.wg.Add(1)
	recorder := newRecorder(t)
	relay.recorder = recorder
	recorder.relay = relay
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
		r.connections = append(r.connections, relayConn{index: connection, client: clientConn, server: serverConn})
		r.wg.Add(2)
		r.mu.Unlock()
		recorder.observeConnection(connection, serverConn.LocalAddr())
		go r.pump(recorder, connection, clientToServer, clientConn, serverConn)
		go r.pump(recorder, connection, serverToClient, clientConn, serverConn)
	}
}

func (r *Relay) pump(recorder *Recorder, connection int, flow flow, clientConn, serverConn net.Conn) {
	defer r.wg.Done()
	defer recorder.observeConnectionClosed(connection)
	defer recorder.observeClose(connection, flow)
	defer func() {
		r.mu.Lock()
		draining := r.draining[connection]
		r.mu.Unlock()
		if !draining {
			_ = clientConn.Close()
		}
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
			r.mu.Lock()
			draining := r.draining[connection]
			r.mu.Unlock()
			if draining {
				continue
			}
			if forwardErr := recorder.forward(connection, flow, buffer[:n], func(message []byte) error {
				_, err := destination.Write(message)
				return err
			}, func() error {
				return r.cutAfterResponse(connection, clientConn, serverConn)
			}); forwardErr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}
