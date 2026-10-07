package harness

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gopcua/opcua/tests/spec/internal/harnessfault"
	"github.com/gopcua/opcua/tests/spec/internal/message"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
)

// T is the testing handle harness helpers use to fail the running
// spec and register cleanup functions. Fatalf must not return: it
// stops the running spec.
type T interface {
	Helper()
	Fatalf(format string, args ...any)
	Cleanup(func())
}

// harnessFault returns the failure message for a fault in the harness
// itself, carrying the spectest: prefix. knowndefectgate classifies a
// message starting with this prefix as a harness fault, not an
// implementation defect.
func harnessFault(format string, args ...any) string {
	return harnessfault.Prefix + fmt.Sprintf(format, args...)
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
	// Stalled marks a message the relay read on a stalled connection,
	// or an ACK a DiscardNextACK armed discarded, and never wrote; the
	// record still holds its decoded message.
	Stalled
)

// ServiceRecord is one service message the relay observed on one
// connection; a Forwarded or Dropped record holds its decoded
// message, the other Fates hold none. ReadAt is the moment the relay
// read the message's final bytes and WrittenAt the moment it wrote
// them upstream, so the two bracket a DelayAt hold.
type ServiceRecord[M any] struct {
	Order      int
	Connection int
	RequestID  uint32
	Fate       Fate
	ReadAt     time.Time
	WrittenAt  time.Time
	message    M
}

// Message returns the decoded service and whether the relay
// forwarded it, a cut CutAt armed dropped it, or it arrived on a
// connection whose forwarding a link fault stopped.
func (r ServiceRecord[M]) Message() (M, bool) {
	if r.Fate != Forwarded && r.Fate != Dropped && r.Fate != Stalled {
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
// relay observed; Fate says whether the relay forwarded it or a cut
// CutAt armed dropped it. ReadAt is the moment the relay read the
// message and WrittenAt the moment it wrote it, so the two bracket a
// DelayAt hold.
type TransportRecord struct {
	Order      int
	Connection int
	Type       TransportType
	Fate       Fate
	ReadAt     time.Time
	WrittenAt  time.Time
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
	index    int
	closed   bool
	upstream string
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
	t                   T
	relay               *Relay
	mu                  sync.Mutex
	order               int
	requests            []ServiceRecord[ua.Request]
	responses           []ServiceRecord[ua.Response]
	transport           []TransportRecord
	log                 []any
	streams             map[streamKey]*streamState
	chunkCounts         map[chunkKey]int
	intermediateWritten map[chunkKey]time.Time
	connections         map[string]*connectionEntry
	err                 error
	reported            bool
}

func newRecorder(t T) *Recorder {
	recorder := &Recorder{
		t:                   t,
		streams:             make(map[streamKey]*streamState),
		chunkCounts:         make(map[chunkKey]int),
		intermediateWritten: make(map[chunkKey]time.Time),
		connections:         make(map[string]*connectionEntry),
	}
	t.Cleanup(func() {
		recorder.mu.Lock()
		defer recorder.mu.Unlock()
		recorder.raiseLocked()
	})
	return recorder
}

// Notification is one data change notification the recorder saw inside
// a forwarded Publish or Republish response, with the wire order and
// connection it arrived on, the subscription it belongs to and the
// value it carried. Republished marks a Republish answer; its
// SubscriptionID is the SubscriptionID of the Republish request the
// response answered.
type Notification struct {
	Order          int
	Connection     int
	SubscriptionID uint32
	SequenceNumber uint32
	Republished    bool
	Value          int32
}

// Notifications returns the data change notifications the recorder saw
// inside forwarded Publish and Republish responses, in wire order.
func (r *Recorder) Notifications() []Notification {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.raiseLocked()
	answered := answeredPublishes(r.requests, r.responses)
	notifications := make([]Notification, len(answered))
	for i, publish := range answered {
		notifications[i] = Notification{
			Order:          publish.order,
			Connection:     publish.connection,
			SubscriptionID: publish.subscriptionID,
			SequenceNumber: publish.sequenceNumber,
			Republished:    publish.republished,
			Value:          publish.value,
		}
	}
	return notifications
}

func (r *Recorder) answeredPublishes() []answeredPublish {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.raiseLocked()
	return answeredPublishes(r.requests, r.responses)
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

// Upstream returns the upstream address the relay dials for
// connections it accepts from now on.
func (r *Relay) Upstream() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.upstream
}

// UpstreamOf returns the upstream address the relay dialled for the
// connection with the given index, or "" for a connection that never
// dialled one.
func (r *Recorder) UpstreamOf(index int) string {
	return r.connectionUpstreamOf(index)
}

// ConnectionStateOf returns the state of the relay connection with the
// given index.
func (r *Recorder) ConnectionStateOf(index int) ConnectionState {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, entry := range r.connections {
		if entry.index != index {
			continue
		}
		if entry.closed {
			return Closed
		}
		return Open
	}
	return Unknown
}

// ConnectionOfOrder returns the relay connection index the record with
// the given Order rode, for a rule that pairs a held request with the
// later requests on its connection.
func (r *Recorder) ConnectionOfOrder(order int) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, record := range r.requests {
		if record.Order == order {
			return record.Connection
		}
	}
	for _, record := range r.responses {
		if record.Order == order {
			return record.Connection
		}
	}
	return -1
}

// TimesOf returns when the relay read the message of a record and when
// it wrote it upstream, for a Forwarded record. The bool is false for
// a record whose message was never written, or that the relay never
// read through forward.
func (r *Recorder) TimesOf(order int) (read time.Time, written time.Time, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, candidate := range r.requests {
		if candidate.Order == order {
			return candidate.ReadAt, candidate.WrittenAt, !candidate.WrittenAt.IsZero()
		}
	}
	for _, candidate := range r.responses {
		if candidate.Order == order {
			return candidate.ReadAt, candidate.WrittenAt, !candidate.WrittenAt.IsZero()
		}
	}
	return time.Time{}, time.Time{}, false
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

func (r *Recorder) observeUpstream(index int, upstream string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, entry := range r.connections {
		if entry.index == index {
			entry.upstream = upstream
		}
	}
}

func (r *Recorder) connectionUpstreamOf(index int) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, entry := range r.connections {
		if entry.index == index {
			return entry.upstream
		}
	}
	return ""
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
		r.t.Fatalf("%s", harnessFault("%v", r.err))
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
	_, _, _, _ = r.observeLocked(connection, flow, data, false)
}

func (r *Recorder) forward(connection int, flow flow, data []byte, write func([]byte) error, finishAfterResponse func(armedCut) error) error {
	r.mu.Lock()
	messages, cut, hold, err := r.observeLocked(connection, flow, data, true)
	r.mu.Unlock()
	// The writes stay outside the recorder lock: a write to a peer
	// that stopped reading blocks, and every connection's pump
	// shares this lock.
	for i, msg := range messages {
		if hold.index == i {
			time.Sleep(time.Until(hold.releaseAt))
		}
		if cut.index == i && cut.drop {
			r.relay.closeConnection(connection, cut.claim)
			return err
		}
		if r.relay.discards(connection, flow, msg) {
			r.markStalled(connection, flow, msg)
			continue
		}
		if writeErr := write(msg); writeErr != nil {
			if cut.index >= 0 {
				r.relay.release(cut.claim)
			}
			return writeErr
		}
		r.noteWritten(connection, flow, msg)
		if cut.index == i {
			if finishErr := finishAfterResponse(cut.claim); finishErr != nil {
				r.setRelayError(fmt.Errorf("the relay could not half-close the client side after the response it wrote: %w", finishErr))
			}
			r.relay.disarm(cut.claim)
			return err
		}
	}
	return err
}

func (r *Recorder) markStalled(connection int, flow flow, message []byte) {
	decoded, err := decodeFrame(message)
	if err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	switch frame := decoded.(type) {
	case *transportFrame:
		for i := range r.transport {
			record := &r.transport[i]
			if record.Connection == connection && record.Fate == Forwarded && record.WrittenAt.IsZero() && matchesTransport(record, frame) {
				record.Fate = Stalled
				return
			}
		}
	case *chunkFrame:
		if frame.chunkType == uacp.ChunkTypeIntermediate {
			return
		}
		if flow == clientToServer {
			for i := range r.requests {
				record := &r.requests[i]
				if record.Connection == connection && record.RequestID == frame.requestID && record.Fate == Forwarded && record.WrittenAt.IsZero() {
					record.Fate = Stalled
					return
				}
			}
			return
		}
		for i := range r.responses {
			record := &r.responses[i]
			if record.Connection == connection && record.RequestID == frame.requestID && record.Fate == Forwarded && record.WrittenAt.IsZero() {
				record.Fate = Stalled
				return
			}
		}
	}
}

// dialledUpstream says whether the relay dialled the upstream server
// for the connection with the given index, including a dial whose
// connection it then closed.
func (r *Relay) dialledUpstream(index int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.dialled[index]
}

// discards says whether the relay must not write this message: the
// connection is stalled, or the message is the ACK a DiscardNextACK
// armed discards, whose claim it consumes.
func (r *Relay) discards(connection int, flow flow, message []byte) bool {
	r.mu.Lock()
	stalled := r.stalled[connection]
	discardACKConn := r.discardACKConn
	r.mu.Unlock()
	if stalled {
		return true
	}
	if discardACKConn != connection || flow != serverToClient {
		return false
	}
	decoded, err := decodeFrame(message)
	if err != nil {
		return false
	}
	frame, isTransport := decoded.(*transportFrame)
	if !isTransport || frame.messageType != uacp.MessageTypeAcknowledge {
		return false
	}
	r.mu.Lock()
	if r.discardACKConn == connection {
		r.discardACKConn = -1
	}
	r.mu.Unlock()
	return true
}

func (r *Recorder) noteWritten(connection int, flow flow, message []byte) {
	header, _, splitErr := splitMessages(message)
	if splitErr != nil || len(header) != 1 {
		return
	}
	decoded, err := decodeFrame(header[0])
	if err != nil {
		return
	}
	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()
	switch frame := decoded.(type) {
	case *transportFrame:
		for i := range r.transport {
			record := &r.transport[i]
			if record.Connection == connection && record.WrittenAt.IsZero() && matchesTransport(record, frame) {
				record.WrittenAt = now
				return
			}
		}
	case *chunkFrame:
		key := chunkKey{connection, flow, frame.requestID}
		if frame.chunkType == uacp.ChunkTypeIntermediate {
			r.intermediateWritten[key] = now
			return
		}
		if r.intermediateWritten[key].After(now) {
			now = r.intermediateWritten[key]
		}
		delete(r.intermediateWritten, key)
		if flow == clientToServer {
			for i := range r.requests {
				record := &r.requests[i]
				if record.Connection == connection && record.RequestID == frame.requestID && record.WrittenAt.IsZero() {
					record.WrittenAt = now
					return
				}
			}
			return
		}
		for i := range r.responses {
			record := &r.responses[i]
			if record.Connection == connection && record.RequestID == frame.requestID && record.WrittenAt.IsZero() {
				record.WrittenAt = now
				return
			}
		}
	}
}

func matchesTransport(record *TransportRecord, frame *transportFrame) bool {
	switch record.Type {
	case HEL:
		return frame.messageType == uacp.MessageTypeHello
	case ACK:
		return frame.messageType == uacp.MessageTypeAcknowledge
	case ERR:
		return frame.messageType == uacp.MessageTypeError
	}
	return false
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

func (r *Recorder) observeLocked(connection int, flow flow, data []byte, claimCuts bool) ([][]byte, firedCut, heldCut, error) {
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
	hold := heldCut{index: -1}
	readAt := time.Now()
	for i, msg := range messages {
		offset := state.consumed
		state.consumed += len(msg)
		order := r.nextOrderLocked()
		claim, dropped, afterResponse, delay, recordErr := r.recordLocked(order, connection, flow, state, msg, claimCuts)
		r.stampReadAtLocked(connection, flow, msg, readAt)
		if recordErr != nil {
			r.setErrorLocked(connection, flow, offset, recordErr)
		}
		if delay > 0 && hold.index < 0 {
			hold = heldCut{index: i, releaseAt: readAt.Add(delay)}
		}
		if (dropped || afterResponse) && cut.index < 0 {
			cut = firedCut{index: i, drop: dropped, claim: claim}
			messages = messages[:i+1]
			break
		}
	}
	if splitErr != nil {
		r.setErrorLocked(connection, flow, state.consumed, splitErr)
		return messages, cut, hold, splitErr
	}
	return messages, cut, hold, nil
}

func (r *Recorder) stampReadAtLocked(connection int, flow flow, msg []byte, readAt time.Time) {
	decoded, err := decodeFrame(msg)
	if err != nil {
		return
	}
	switch frame := decoded.(type) {
	case *transportFrame:
		for i := range r.transport {
			record := &r.transport[i]
			if record.Connection == connection && record.ReadAt.IsZero() && matchesTransport(record, frame) {
				record.ReadAt = readAt
				return
			}
		}
	case *chunkFrame:
		if flow == clientToServer {
			for i := range r.requests {
				record := &r.requests[i]
				if record.Connection == connection && record.RequestID == frame.requestID && record.ReadAt.IsZero() {
					record.ReadAt = readAt
					return
				}
			}
			return
		}
		for i := range r.responses {
			record := &r.responses[i]
			if record.Connection == connection && record.RequestID == frame.requestID && record.ReadAt.IsZero() {
				record.ReadAt = readAt
				return
			}
		}
	}
}

func (r *Recorder) recordLocked(order, connection int, flow flow, state *streamState, msg []byte, claimCuts bool) (claimed armedCut, dropped bool, afterResponse bool, delay time.Duration, err error) {
	decoded, err := decodeFrame(msg)
	if err != nil {
		return armedCut{}, false, false, 0, err
	}
	switch frame := decoded.(type) {
	case *transportFrame:
		fate := Forwarded
		var claim armedCut
		dropped = false
		afterResponse = false
		if claimCuts && flow == clientToServer && frame.messageType == uacp.MessageTypeHello {
			if cut, fired := r.relay.takeArmed(func(c armedCut) bool {
				return c.moment == BeforeRequestReachesServer && c.message == message.HEL && !r.hasTransportLocked(connection, HEL)
			}); fired {
				claim, dropped, fate = cut, true, Dropped
			}
		}
		if claimCuts && flow == serverToClient && frame.messageType == uacp.MessageTypeAcknowledge {
			if cut, fired := r.relay.takeArmed(func(c armedCut) bool {
				return isResponseMoment(c.moment) && c.message == message.HEL && r.hasTransportLocked(connection, HEL)
			}); fired {
				claim = cut
				if cut.moment == ResponseNeverReachesClient {
					dropped, fate = true, Dropped
				} else {
					afterResponse = true
				}
			}
		}
		if claimCuts && flow == clientToServer && frame.messageType == uacp.MessageTypeHello {
			if d, armed := r.relay.takeDelay(message.HEL); armed {
				delay = d
			}
		}
		record := transportRecord(order, connection, frame)
		record.Fate = fate
		r.transport = append(r.transport, record)
		r.log = append(r.log, record)
		return claim, dropped, afterResponse, delay, nil
	case *chunkFrame:
		r.chunkCounts[chunkKey{connection, flow, frame.requestID}]++
		service, complete, err := state.reassembler.add(frame)
		if err != nil {
			return armedCut{}, false, false, 0, err
		}
		if !complete {
			return armedCut{}, false, false, 0, nil
		}
		switch svc := service.(type) {
		case *completeMessage:
			if flowErr := serviceDirectionError(flow, svc.service); flowErr != nil {
				return armedCut{}, false, false, 0, flowErr
			}
			if claimCuts && flow == clientToServer {
				if target, named := MessageOf(svc.service); named {
					if cut, fired := r.relay.takeArmed(func(c armedCut) bool {
						return c.moment == BeforeRequestReachesServer && c.message == target
					}); fired {
						r.appendService(order, connection, flow, svc.requestID, Dropped, svc.service)
						return cut, true, false, 0, nil
					}
					if d, armed := r.relay.takeDelay(target); armed {
						delay = d
					}
				}
			}
			if claimCuts && flow == serverToClient {
				if cut, fired := r.relay.takeArmed(func(c armedCut) bool {
					return isResponseMoment(c.moment) && r.hasRecordedRequestLocked(connection, svc.requestID, c.message)
				}); fired {
					if cut.moment == ResponseNeverReachesClient {
						r.appendService(order, connection, flow, svc.requestID, Dropped, svc.service)
						return cut, true, false, 0, nil
					}
					r.appendService(order, connection, flow, svc.requestID, Forwarded, svc.service)
					return cut, false, true, 0, nil
				}
				if d, armed := r.relay.takeResponseHold(); armed {
					delay = d
				}
			}
			r.appendService(order, connection, flow, svc.requestID, Forwarded, svc.service)
			return armedCut{}, false, false, delay, nil
		case *abortedMessage:
			r.appendService(order, connection, flow, svc.requestID, Aborted, nil)
			return armedCut{}, false, false, 0, nil
		}
	}
	return armedCut{}, false, false, 0, nil
}

func (r *Recorder) lastClientHandleOf(subscriptionID uint32) (handle uint32, found bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.requests) - 1; i >= 0; i-- {
		record := r.requests[i]
		if record.message == nil {
			continue
		}
		create, isCreate := record.message.(*ua.CreateMonitoredItemsRequest)
		if !isCreate || create.SubscriptionID != subscriptionID {
			continue
		}
		for _, item := range create.ItemsToCreate {
			if item != nil && item.RequestedParameters != nil {
				return item.RequestedParameters.ClientHandle, true
			}
		}
		return 0, true
	}
	return 0, false
}

func (r *Recorder) hasRecordedRequestLocked(connection int, requestID uint32, message message.Message) bool {
	for _, record := range r.requests {
		if record.Connection != connection || record.RequestID != requestID {
			continue
		}
		if record.message != nil {
			if named, ok := MessageOf(record.message); ok && named == message {
				return true
			}
		}
	}
	return false
}

func (r *Recorder) hasTransportLocked(connection int, typ TransportType) bool {
	for _, record := range r.transport {
		if record.Connection == connection && record.Type == typ {
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
	// ResponseNeverReachesClient fires once the matching response is
	// complete, without writing it to the client; the relay records it
	// Dropped and closes the connection like BeforeRequestReachesServer
	// does.
	ResponseNeverReachesClient
)

func isResponseMoment(moment Moment) bool {
	return moment == AfterResponseReachesClient || moment == ResponseNeverReachesClient
}

// MessageOf maps a decoded client request to the message.Message
// naming it. The bool is false for a request the catalogue does not
// name.
func MessageOf(service any) (message.Message, bool) {
	switch service.(type) {
	case *ua.OpenSecureChannelRequest:
		return message.OpenSecureChannel, true
	case *ua.CloseSecureChannelRequest:
		return message.CloseSecureChannel, true
	case *ua.CreateSessionRequest:
		return message.CreateSession, true
	case *ua.ActivateSessionRequest:
		return message.ActivateSession, true
	case *ua.CloseSessionRequest:
		return message.CloseSession, true
	case *ua.ReadRequest:
		return message.Read, true
	case *ua.CreateSubscriptionRequest:
		return message.CreateSubscription, true
	case *ua.CreateMonitoredItemsRequest:
		return message.CreateMonitoredItems, true
	case *ua.DeleteSubscriptionsRequest:
		return message.DeleteSubscriptions, true
	case *ua.PublishRequest:
		return message.Publish, true
	case *ua.RepublishRequest:
		return message.Republish, true
	case *ua.TransferSubscriptionsRequest:
		return message.TransferSubscriptions, true
	}
	return 0, false
}

type armedCut struct {
	id      int
	moment  Moment
	message message.Message
	firing  bool
}

type firedCut struct {
	index int
	drop  bool
	claim armedCut
}

type heldCut struct {
	index     int
	releaseAt time.Time
}

func (c armedCut) describe() string {
	switch c.moment {
	case AfterResponseReachesClient:
		return fmt.Sprintf("after a %s response reaches the client", c.message)
	case ResponseNeverReachesClient:
		return fmt.Sprintf("the %s response never reaches the client", c.message)
	}
	return fmt.Sprintf("before a %s request reaches the server", c.message)
}

// Relay accepts client connections and forwards them to a fixed
// upstream server while a Recorder observes the traffic.
type Relay struct {
	listener        net.Listener
	recorder        *Recorder
	t               T
	mu              sync.Mutex
	connections     []relayConn
	count           int
	wg              sync.WaitGroup
	closed          bool
	onCut           func()
	nextCutID       int
	armed           []armedCut
	armedDelay      *armedDelay
	responseHold    *time.Duration
	draining        map[int]bool
	stalled         map[int]bool
	dialled         map[int]bool
	closeNextAccept int
	discardACKConn  int
	upstream        string
}

type armedDelay struct {
	message message.Message
	hold    time.Duration
	firing  bool
}

type relayConn struct {
	index  int
	client net.Conn
	server net.Conn
}

func (r *Relay) address() string {
	return r.listener.Addr().String()
}

// markCut records the cut position before any socket of the cut
// connection closes, so the client's reaction to the close lands above
// the cut position WaitUntilReconnected searches from.
func (r *Relay) markCut() {
	r.mu.Lock()
	onCut := r.onCut
	r.mu.Unlock()
	if onCut != nil {
		onCut()
	}
}

// Cut closes both sides of every live connection. The listener stays
// open, so the next dial is accepted immediately.
func (r *Relay) Cut() {
	r.markCut()
	r.mu.Lock()
	conns := r.connections
	r.connections = nil
	r.mu.Unlock()
	for _, conn := range conns {
		_ = conn.client.Close()
		_ = conn.server.Close()
		r.recorder.observeConnectionClosed(conn.index)
	}
}

// CutAt arms one cut: a Moment and a message.Message the cut fires on,
// on the next message that matches both. A cut armed for
// BeforeRequestReachesServer drops the matching request, records it
// Dropped and closes the connection it rode on. A cut armed for
// AfterResponseReachesClient writes the matching response to the
// client, half-closes the client side, closes the server side and
// keeps reading and discarding what the client sends for 2 s, then
// closes the client side. A cut armed for ResponseNeverReachesClient
// drops the matching response without writing it, records it Dropped
// and closes the connection like BeforeRequestReachesServer does.
// Transport messages pair by position on their connection: a HEL pairs
// with the ACK that answers it and a message.OpenSecureChannel request
// with its response, so a cut armed for a transport message or its
// moment fires on that pair. The CloseSecureChannel the client sends
// at Close gets no response, so a cut armed on it for a response
// moment is a harness fault, as are an unknown Moment and a Message
// the catalogue does not name.
func (r *Relay) CutAt(moment Moment, msg message.Message) {
	if moment != BeforeRequestReachesServer && !isResponseMoment(moment) {
		r.t.Fatalf("%s", harnessFault("CutAt received an unknown Moment %d", int(moment)))
		return
	}
	if msg < message.HEL || msg > message.TransferSubscriptions {
		r.t.Fatalf("%s", harnessFault("CutAt received an unknown Message %d", int(msg)))
		return
	}
	if isResponseMoment(moment) && msg == message.CloseSecureChannel {
		r.t.Fatalf("%s", harnessFault("CloseSecureChannel has no response a response-moment cut could fire on"))
		return
	}
	r.mu.Lock()
	r.armed = append(r.armed, armedCut{id: r.nextCutID, moment: moment, message: msg})
	r.nextCutID++
	r.mu.Unlock()
}

// DelayAt arms one hold: the request of the first matching message
// after arming, client-to-server only, is held for d before the relay
// writes it upstream. Every later message in that direction on that
// connection waits behind it, and all are released in their original
// order at one moment. The service of a request is known only at its
// last chunk, so earlier chunks of the held request are already
// forwarded; the hold starts at the last chunk.
func (r *Relay) DelayAt(msg message.Message, d time.Duration) {
	if msg == message.CloseSecureChannel {
		r.t.Fatalf("%s", harnessFault("the client sends CloseSecureChannel only while closing, so no request of it can arrive to delay"))
		return
	}
	if msg < message.HEL || msg > message.TransferSubscriptions {
		r.t.Fatalf("%s", harnessFault("DelayAt received an unknown Message %d", int(msg)))
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.armedDelay = &armedDelay{message: msg, hold: d}
}

// HoldNextResponse holds the next server-to-client service response
// for d before the relay writes it to the client; later messages in
// that direction on that connection wait behind it and all are
// released in their original order at one moment.
func (r *Relay) HoldNextResponse(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	hold := d
	r.responseHold = &hold
}

// takeResponseHold reports and consumes the armed response hold.
func (r *Relay) takeResponseHold() (time.Duration, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.responseHold == nil {
		return 0, false
	}
	hold := *r.responseHold
	r.responseHold = nil
	return hold, true
}

// takeDelay reports the hold armed for message, marking it firing so
// one armed hold delays one matching request only.
func (r *Relay) takeDelay(msg message.Message) (time.Duration, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.armedDelay == nil || r.armedDelay.firing || r.armedDelay.message != msg {
		return 0, false
	}
	r.armedDelay.firing = true
	return r.armedDelay.hold, true
}

// ArmedCuts returns a readable description of every cut CutAt armed
// whose position the relay has not marked yet, in arming order. A cut
// the relay is firing stays listed until markCut has returned.
func (r *Relay) ArmedCuts() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	descriptions := make([]string, len(r.armed))
	for i, cut := range r.armed {
		descriptions[i] = cut.describe()
	}
	return descriptions
}

// afterResponseCutPending reports whether a connection an
// after-response cut fired on is still without its recorded close: the
// relay half-closed it and keeps discarding what the client sends for
// its drain window before it records the close.
func (r *Relay) afterResponseCutPending() bool {
	r.mu.Lock()
	indexes := make([]int, 0, len(r.draining))
	for index := range r.draining {
		indexes = append(indexes, index)
	}
	r.mu.Unlock()
	for _, index := range indexes {
		if r.recorder.ConnectionStateOf(index) != Closed {
			return true
		}
	}
	return false
}

func (r *Relay) takeArmed(match func(armedCut) bool) (armedCut, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, cut := range r.armed {
		if cut.firing || !match(cut) {
			continue
		}
		r.armed[i].firing = true
		return cut, true
	}
	return armedCut{}, false
}

func (r *Relay) disarm(cut armedCut) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, armed := range r.armed {
		if armed.id == cut.id {
			r.armed = append(r.armed[:i], r.armed[i+1:]...)
			return
		}
	}
}

func (r *Relay) release(cut armedCut) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, armed := range r.armed {
		if armed.id == cut.id {
			r.armed[i].firing = false
			return
		}
	}
}

func (r *Relay) closeConnection(index int, firedCut armedCut) {
	r.markCut()
	r.disarm(firedCut)
	r.closeConnectionSockets(index)
}

func (r *Relay) closeConnectionSockets(index int) {
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
}

const clientDrainTimeout = 2 * time.Second

func (r *Relay) cutAfterResponse(connection int, clientConn, serverConn net.Conn, claim armedCut) error {
	r.markCut()
	r.disarm(claim)
	r.mu.Lock()
	r.draining[connection] = true
	r.mu.Unlock()
	closeErr := halfClose(clientConn)
	_ = serverConn.Close()
	if closeErr == nil {
		drainClient(clientConn)
	}
	_ = clientConn.Close()
	r.closeConnectionSockets(connection)
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
	if closer, ok := conn.(interface{ CloseWrite() error }); ok {
		return closer.CloseWrite()
	}
	return nil
}

// ConnectionCount returns the number of connections the relay has
// accepted since it was created.
func (r *Relay) ConnectionCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.count
}

// RedirectTo makes every connection the relay accepts from now on dial
// address instead of the server it was created with. Connection
// indexes continue from where they were.
func (r *Relay) RedirectTo(address string) {
	r.mu.Lock()
	r.upstream = strings.TrimPrefix(address, "opc.tcp://")
	r.mu.Unlock()
}

func newRelay(t T, upstream string, onCut func()) (*Relay, *Recorder) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("%s", harnessFault("the relay could not listen on 127.0.0.1: %v", err))
	}
	relay := &Relay{
		listener:       listener,
		t:              t,
		onCut:          onCut,
		draining:       make(map[int]bool),
		stalled:        make(map[int]bool),
		dialled:        make(map[int]bool),
		discardACKConn: -1,
		upstream:       strings.TrimPrefix(upstream, "opc.tcp://"),
	}
	relay.wg.Add(1)
	recorder := newRecorder(t)
	relay.recorder = recorder
	recorder.relay = relay
	t.Cleanup(relay.close)
	go relay.acceptLoop(recorder)
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

// CloseNextAccepts makes the relay close the next n accepted
// connections before it dials the upstream server, so the client sees
// EOF on a connection that never carried its traffic. The closed
// connections still count in ConnectionCount.
func (r *Relay) CloseNextAccepts(n int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closeNextAccept += n
}

// CloseListenerFor closes the relay's listener now and reopens a
// listener on the same address after d, so dials in between are
// refused and the address stays the same.
func (r *Relay) CloseListenerFor(d time.Duration) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	address := r.listener.Addr().String()
	_ = r.listener.Close()
	r.mu.Unlock()
	time.AfterFunc(d, func() {
		listener, err := net.Listen("tcp", address)
		if err != nil {
			r.t.Fatalf("%s", harnessFault("the relay could not reopen its listener on %s: %v", address, err))
			return
		}
		r.mu.Lock()
		reopened := !r.closed
		if reopened {
			r.listener = listener
		}
		r.mu.Unlock()
		if !reopened {
			_ = listener.Close()
		}
	})
}

// Stall stops the relay forwarding on every currently open
// connection, in both directions: it keeps reading and recording what
// each side sends, records it Stalled and writes nothing, while the
// sockets stay open. Connections accepted later forward normally.
func (r *Relay) Stall() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, conn := range r.connections {
		r.stalled[conn.index] = true
	}
}

// DiscardNextACK makes the relay discard the ACK the server sends on
// the next accepted connection, after forwarding that connection's
// HEL: the client's handshake never completes while the connection
// stays open.
func (r *Relay) DiscardNextACK() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.discardACKConn = -2
}

func (r *Relay) acceptLoop(recorder *Recorder) {
	defer r.wg.Done()
	for {
		r.mu.Lock()
		listener := r.listener
		closed := r.closed
		r.mu.Unlock()
		if closed {
			return
		}
		clientConn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				// The listener was closed for a window or the relay is
				// closing; re-check which one it was.
				time.Sleep(5 * time.Millisecond)
				continue
			}
			recorder.setRelayError(fmt.Errorf("the relay stopped accepting client connections: %w", err))
			_ = r.listener.Close()
			return
		}
		r.mu.Lock()
		if r.closed {
			r.mu.Unlock()
			_ = clientConn.Close()
			return
		}
		connection := r.count
		r.count++
		if r.closeNextAccept > 0 {
			r.closeNextAccept--
			r.mu.Unlock()
			_ = clientConn.Close()
			continue
		}
		if r.discardACKConn == -2 {
			r.discardACKConn = connection
		}
		addr := r.upstream
		r.mu.Unlock()
		serverConn, err := net.Dial("tcp", addr)
		if err != nil {
			_ = clientConn.Close()
			recorder.setRelayError(fmt.Errorf("the relay could not connect to the upstream server at %s: %w", addr, err))
			continue
		}
		r.mu.Lock()
		r.dialled[connection] = true
		r.mu.Unlock()
		r.mu.Lock()
		if r.closed {
			r.mu.Unlock()
			_ = clientConn.Close()
			_ = serverConn.Close()
			return
		}
		r.connections = append(r.connections, relayConn{index: connection, client: clientConn, server: serverConn})
		r.wg.Add(2)
		r.mu.Unlock()
		recorder.observeConnection(connection, serverConn.LocalAddr())
		recorder.observeUpstream(connection, addr)
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
			}, func(claim armedCut) error {
				return r.cutAfterResponse(connection, clientConn, serverConn, claim)
			}); forwardErr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}
