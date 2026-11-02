// Package reconnect holds the gopcua reconnect tests: one behaviour per file,
// with the clause it checks in the file's opening comment. This file is the
// fixture they all run against.
//
// The fixture is a scripted OPC UA server. It drops the transport on command,
// can refuse every connection after the drop, and records what it decoded off
// the wire and what the client's callback received.
//
// The lost notification is constructed. The server answers the stream's first
// Publish and holds every later one: the notification gets a sequence number
// but is never sent. Once a Publish is held, the scenario resets the connection
// it is outstanding on, so the held notification cannot have reached the
// client. Nothing tells the session its transport is gone.
//
// The sequence counter is global and never resets. A subscription the client
// recreates after the drop starts at 3, which looks like a gap to a client that
// detects loss by sequence gaps.
package reconnect

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/uasc"
)

// publishPacing is how long the scripted server waits before answering a
// Publish. Without it the client would re-publish as fast as it is answered
// and flood the request record.
const publishPacing = 200 * time.Millisecond

// settleDefault is how long a run goes on after the drop when the test sets no
// Settle.
const settleDefault = 5 * time.Second

// SubID is the subscription id the scripted server hands out; the scenario
// has one subscription.
const SubID uint32 = 42

// monitoredNode is the node the scenario monitors. The scripted server never
// reads it; it manufactures notifications instead.
var monitoredNode = ua.NewStringNodeID(1, "reconnect.value")

// requestRecord is one service request the scripted server decoded off the
// wire. Epoch is the connection generation it arrived on, counting from 1.
// record.DropEpoch says which generation was dropped. RepublishSeq is the
// sequence number a RepublishRequest asked for and is meaningless on any
// other service.
type requestRecord struct {
	Epoch        int
	Service      string
	RepublishSeq uint32
}

// deliveryRecord is one notification the client's application callback
// received, carrying the sequence number the server encoded into the value.
type deliveryRecord struct {
	Seq uint32
	At  time.Time
}

// recorder collects what both ends saw while the run is live. Safe for
// concurrent use: the server serves each connection on its own goroutine and
// the drop watcher runs beside them.
type recorder struct {
	mu       sync.Mutex
	requests []requestRecord
	accepts  []time.Time // when each connection was accepted, refused ones included

	// dropStarted is set just before the connection is closed, so a client
	// that redials the instant it sees the reset is already refused.
	dropStarted bool
	// dropApplied is set once the close has succeeded; the scenario waits for it.
	dropApplied bool

	lastSeq   uint32 // highest notification sequence created so far
	heldSeq   uint32 // sequence of the first held Publish; 0 = none
	heldEpoch int    // connection the held Publish is outstanding on; 0 = none
	answered  bool   // the stream's first Publish has been answered

	// activatedEpochs holds the connections ActivateSession has arrived on. It
	// is per connection because a reconnect must activate the session again.
	activatedEpochs map[int]bool
	// anyMonitored is state of the whole fixture, unlike activatedEpochs: the
	// scenario has one subscription. CreateSubscription clears it and
	// CreateMonitoredItems sets it.
	anyMonitored bool
}

// recordRequest takes the decoded request, so the call site does not need to
// know which services carry something worth keeping.
func (o *recorder) recordRequest(epoch int, req ua.Request) {
	rr := requestRecord{Epoch: epoch, Service: serviceName(req)}
	if r, ok := req.(*ua.RepublishRequest); ok {
		rr.RepublishSeq = r.RetransmitSequenceNumber
	}
	o.mu.Lock()
	if rr.Service == "ActivateSessionRequest" {
		if o.activatedEpochs == nil {
			o.activatedEpochs = map[int]bool{}
		}
		o.activatedEpochs[epoch] = true
	}
	o.requests = append(o.requests, rr)
	o.mu.Unlock()
}

func (o *recorder) recordConn() {
	o.mu.Lock()
	o.accepts = append(o.accepts, time.Now())
	o.mu.Unlock()
}

func (o *recorder) isDropStarted() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.dropStarted
}

func (o *recorder) recordDropStarted() {
	o.mu.Lock()
	o.dropStarted = true
	o.mu.Unlock()
}

func (o *recorder) recordDropApplied() {
	o.mu.Lock()
	o.dropApplied = true
	o.mu.Unlock()
}

func (o *recorder) isActivated(epoch int) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.activatedEpochs[epoch]
}

func (o *recorder) setAnyMonitored(v bool) {
	o.mu.Lock()
	o.anyMonitored = v
	o.mu.Unlock()
}

func (o *recorder) isAnyMonitored() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.anyMonitored
}

// peekNextSeq is the sequence number the next notification will carry.
// Reading it does not use it up, so a keep-alive can carry it.
func (o *recorder) peekNextSeq() uint32 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.lastSeq + 1
}

func (o *recorder) highestSeq() uint32 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.lastSeq
}

func (o *recorder) heldPublishSeq() uint32 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.heldSeq
}

// availableSeqs is what the server says it can still resend. An empty list
// claims it keeps no retransmission queue at all (OPC 10000-4 §5.14.1.1), so
// while a notification is held back the list has to name it. Every Publish
// answer and every TransferResult carries this, and they have to agree.
func (o *recorder) availableSeqs() []uint32 {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.heldSeq == 0 {
		return []uint32{}
	}
	return []uint32{o.heldSeq}
}

func (o *recorder) heldPublishEpoch() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.heldEpoch
}

// publishStep answers the stream's first Publish and holds every later one.
// After the drop it answers again with fresh data, so a client that notices a
// gap only when the next notification arrives has one to notice it in.
// Every call uses up a sequence number, held or not.
func (o *recorder) publishStep(epoch int) (seq uint32, hold bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.lastSeq++
	if o.dropApplied {
		return o.lastSeq, false
	}
	if o.answered {
		if o.heldSeq == 0 {
			o.heldSeq = o.lastSeq
		}
		if o.heldEpoch == 0 {
			o.heldEpoch = epoch
		}
		return o.lastSeq, true
	}
	o.answered = true
	return o.lastSeq, false
}

// snapshot returns copies of the request and accept logs, with the drop and held state.
func (o *recorder) snapshot() (requests []requestRecord, accepts []time.Time, dropApplied bool, held uint32) {
	o.mu.Lock()
	defer o.mu.Unlock()
	requests = append([]requestRecord(nil), o.requests...)
	accepts = append([]time.Time(nil), o.accepts...)
	return requests, accepts, o.dropApplied, o.heldSeq
}

// record is what one scenario run reduces to: what both ends saw, frozen when
// the run ends. A test decides pass or fail from this value alone.
type record struct {
	// Requests is every service request the server decoded, each with the
	// epoch of the connection it arrived on.
	Requests []requestRecord
	// Deliveries is what the application's own callback received.
	Deliveries []deliveryRecord
	// HeldSeq is the sequence number of the notification the server held back.
	HeldSeq uint32
	// LastSeq is the highest sequence number the server created.
	LastSeq uint32
	// DropApplied is whether the commanded transport drop fired, at DropAt.
	DropApplied bool
	DropAt      time.Time
	// DropEpoch is the connection generation the drop was applied to. A
	// request on a higher one arrived on a reconnect; a fixed 2 would miscount
	// a client that dialled twice before the drop.
	DropEpoch int
	// Accepts is when each connection was accepted, refused ones included. A
	// refused connection is served by nobody, so it shows up here and in no
	// request's epoch.
	Accepts []time.Time
	// HarnessErr is a stage the scenario could not reach; the test reports it
	// as a setup failure.
	HarnessErr string
}

// setupFailed reports why the run never reached the point the test is about,
// or "" if it did. Whatever caused it, no test here can speak past it.
//
// Every check of this kind is named …SetupFailed, takes the run, returns the
// reason or "", and is reported as "could not set the test up: %s". A row
// consults one up front when nothing it scores could be true without it, and
// as the last arm of its switch when the violation it scores would show up
// anyway; either way the reason is the row's, and the wording is the
// package's.
func setupFailed(run record) string {
	switch {
	case run.HarnessErr != "":
		return run.HarnessErr
	case !run.DropApplied:
		return "the connection was never dropped"
	}
	return ""
}

// reconnectSetupFailed reports why the run never got the client back onto an
// activated Session, or "" if it did. A client that never re-activated was
// never offered the Session's subscriptions back, so nothing it did or did
// not do with them is evidence.
func reconnectSetupFailed(run record) string {
	for _, r := range afterDrop(run) {
		if r.Service == "ActivateSessionRequest" {
			return ""
		}
	}
	return "the client never activated a Session after the drop, so the Session's " +
		"subscriptions were never offered back to it"
}

// afterDrop is every request the client sent on a connection it opened after
// the drop.
func afterDrop(run record) []requestRecord {
	var out []requestRecord
	for _, r := range run.Requests {
		if r.Epoch > run.DropEpoch {
			out = append(out, r)
		}
	}
	return out
}

// server is the scripted fixture the client under test talks to.
type server struct {
	rec      *recorder
	ln       *uacp.Listener
	endpoint string

	dropCh          chan struct{} // the scenario's one-shot drop command
	refuseAfterDrop bool          // close every connection accepted once the drop has started

	mu    sync.Mutex         // guards conns
	conns map[int]*uacp.Conn // by epoch
}

// newServer starts the scripted server on an OS-chosen port.
func newServer(ctx context.Context, refuseAfterDrop bool) (*server, error) {
	ln, err := uacp.Listen(ctx, "opc.tcp://127.0.0.1:0", nil)
	if err != nil {
		return nil, err
	}
	s := &server{
		rec:             &recorder{},
		ln:              ln,
		endpoint:        "opc.tcp://" + ln.Addr().String(),
		dropCh:          make(chan struct{}, 1),
		conns:           map[int]*uacp.Conn{},
		refuseAfterDrop: refuseAfterDrop,
	}
	go s.acceptLoop(ctx)
	go s.dropWatcher()
	return s, nil
}

// drop commands a TCP reset of the connection the held Publish is outstanding
// on. The session is not told.
func (s *server) drop() {
	select {
	case s.dropCh <- struct{}{}:
	default:
	}
}

// dropWatcher applies the commanded drop to the connection pinned by heldEpoch;
// the newest connection could already be a redial.
func (s *server) dropWatcher() {
	<-s.dropCh
	epoch := s.rec.heldPublishEpoch()
	s.mu.Lock()
	conn := s.conns[epoch]
	s.mu.Unlock()
	applied := false
	if conn != nil {
		// Marked before the close, so a redial after the reset is refused.
		s.rec.recordDropStarted()
		_ = conn.SetLinger(0)
		applied = conn.Close() == nil
	}
	if applied {
		s.rec.recordDropApplied()
	}
}

func (s *server) acceptLoop(ctx context.Context) {
	for {
		c, err := s.ln.Accept(ctx)
		if err != nil {
			return
		}
		s.rec.recordConn()
		// The Hello/Acknowledge handshake is already done, so closing here
		// fails the client's OpenSecureChannel.
		if s.refuseAfterDrop && s.rec.isDropStarted() {
			_ = c.Close()
			continue
		}
		s.mu.Lock()
		epoch := len(s.conns) + 1
		s.conns[epoch] = c
		s.mu.Unlock()
		go s.serve(ctx, epoch, c)
	}
}

func (s *server) serve(ctx context.Context, epoch int, c *uacp.Conn) {
	cfg := &uasc.Config{
		SecurityPolicyURI: ua.SecurityPolicyURINone,
		SecurityMode:      ua.MessageSecurityModeNone,
		Lifetime:          uint32(time.Hour / time.Millisecond),
	}
	errch := make(chan error, 8)
	// Channel, sequence and token ids differ per connection.
	sc, err := uasc.NewServerSecureChannel(s.endpoint, c, cfg, errch, uint32(1000+epoch), uint32(100*epoch+1), uint32(2000+epoch))
	if err != nil {
		return
	}
	defer sc.Close()

	for {
		msg := sc.Receive(ctx)
		if msg == nil || msg.Err != nil {
			return
		}
		req := msg.Request()
		if req == nil {
			continue
		}
		s.rec.recordRequest(epoch, req)

		resp := s.answer(epoch, req)
		if resp == nil {
			continue
		}
		_ = sc.SendResponseWithContext(ctx, msg.RequestID, resp)
	}
}

func (s *server) answer(epoch int, req ua.Request) ua.Response {
	h := req.Header().RequestHandle
	switch r := req.(type) {
	case *ua.CreateSessionRequest:
		return &ua.CreateSessionResponse{
			ResponseHeader:        newResponseHeader(h, ua.StatusOK),
			SessionID:             ua.NewNumericNodeID(1, 6000),
			AuthenticationToken:   ua.NewNumericNodeID(0, 7000),
			RevisedSessionTimeout: 60000,
			ServerEndpoints:       []*ua.EndpointDescription{anonymousAuthEndpoint(s.endpoint)},
			ServerSignature:       &ua.SignatureData{},
		}

	case *ua.ActivateSessionRequest:
		// The session survives the drop, so re-activation succeeds.
		return &ua.ActivateSessionResponse{
			ResponseHeader: newResponseHeader(h, ua.StatusOK),
			ServerNonce:    nil,
			Results:        []ua.StatusCode{},
		}

	case *ua.CloseSessionRequest:
		return &ua.CloseSessionResponse{ResponseHeader: newResponseHeader(h, ua.StatusOK)}

	case *ua.ReadRequest:
		return &ua.ReadResponse{
			ResponseHeader: newResponseHeader(h, ua.StatusOK),
			Results: []*ua.DataValue{{
				EncodingMask: ua.DataValueValue,
				Value:        ua.MustVariant([]string{"http://opcfoundation.org/UA/"}),
			}},
		}

	case *ua.CreateSubscriptionRequest:
		s.rec.setAnyMonitored(false)
		return &ua.CreateSubscriptionResponse{
			ResponseHeader:            newResponseHeader(h, ua.StatusOK),
			SubscriptionID:            SubID,
			RevisedPublishingInterval: r.RequestedPublishingInterval,
			RevisedLifetimeCount:      1000,
			RevisedMaxKeepAliveCount:  10,
		}

	case *ua.CreateMonitoredItemsRequest:
		if r.SubscriptionID == SubID {
			s.rec.setAnyMonitored(true)
		}
		results := make([]*ua.MonitoredItemCreateResult, len(r.ItemsToCreate))
		for i := range r.ItemsToCreate {
			results[i] = &ua.MonitoredItemCreateResult{
				StatusCode:              ua.StatusOK,
				MonitoredItemID:         uint32(i + 1),
				RevisedSamplingInterval: 200,
				RevisedQueueSize:        1,
			}
		}
		return &ua.CreateMonitoredItemsResponse{
			ResponseHeader:  newResponseHeader(h, ua.StatusOK),
			Results:         results,
			DiagnosticInfos: []*ua.DiagnosticInfo{},
		}

	case *ua.PublishRequest:
		if !s.rec.isActivated(epoch) {
			return newServiceFault(h, ua.StatusBadSessionNotActivated)
		}
		// With nothing monitored there is no data, only keep-alives.
		if !s.rec.isAnyMonitored() {
			time.Sleep(publishPacing)
			return newKeepAliveResponse(s.rec.peekNextSeq(), s.rec.availableSeqs())
		}
		seq, hold := s.rec.publishStep(epoch)
		if hold {
			return nil
		}
		time.Sleep(publishPacing)
		return newPublishResponse(seq, s.rec.availableSeqs())

	case *ua.TransferSubscriptionsRequest:
		results := make([]*ua.TransferResult, len(r.SubscriptionIDs))
		for i := range r.SubscriptionIDs {
			results[i] = &ua.TransferResult{
				StatusCode:               ua.StatusOK,
				AvailableSequenceNumbers: s.rec.availableSeqs(),
			}
		}
		return &ua.TransferSubscriptionsResponse{
			ResponseHeader:  newResponseHeader(h, ua.StatusOK),
			Results:         results,
			DiagnosticInfos: []*ua.DiagnosticInfo{},
		}

	case *ua.RepublishRequest:
		// The server never has a message to give back. OPC 10000-4 §6.7 names
		// Bad_MessageNotAvailable as the status that ends the Republish loop.
		// Whether the client asked is read from the request record.
		return newServiceFault(h, ua.StatusBadMessageNotAvailable)
	}
	// Services the scenario does not script fault.
	return newServiceFault(h, ua.StatusBadServiceUnsupported)
}

func serviceName(v any) string {
	return strings.TrimPrefix(fmt.Sprintf("%T", v), "*ua.")
}

func newResponseHeader(handle uint32, status ua.StatusCode) *ua.ResponseHeader {
	return &ua.ResponseHeader{
		Timestamp:          time.Now(),
		RequestHandle:      handle,
		ServiceResult:      status,
		ServiceDiagnostics: &ua.DiagnosticInfo{},
		StringTable:        []string{},
		AdditionalHeader:   ua.NewExtensionObject(nil),
	}
}

func newServiceFault(handle uint32, status ua.StatusCode) ua.Response {
	return &ua.ServiceFault{ResponseHeader: newResponseHeader(handle, status)}
}

// newNotification builds a notification whose value is its own sequence
// number, so a delivery record shows which one the client received.
func newNotification(seq uint32) *ua.NotificationMessage {
	return &ua.NotificationMessage{
		SequenceNumber: seq,
		PublishTime:    time.Now(),
		NotificationData: []*ua.ExtensionObject{
			ua.NewExtensionObject(&ua.DataChangeNotification{
				MonitoredItems: []*ua.MonitoredItemNotification{{
					ClientHandle: 1,
					Value:        &ua.DataValue{EncodingMask: ua.DataValueValue, Value: ua.MustVariant(int32(seq))},
				}},
				DiagnosticInfos: []*ua.DiagnosticInfo{},
			}),
		},
	}
}

func newPublishResponse(seq uint32, available []uint32) *ua.PublishResponse {
	return &ua.PublishResponse{
		ResponseHeader:           newResponseHeader(0, ua.StatusOK),
		SubscriptionID:           SubID,
		AvailableSequenceNumbers: available,
		MoreNotifications:        false,
		NotificationMessage:      newNotification(seq),
		Results:                  []ua.StatusCode{},
		DiagnosticInfos:          []*ua.DiagnosticInfo{},
	}
}

// newKeepAliveResponse answers a Publish with no data. It carries the sequence
// number the next notification will use.
func newKeepAliveResponse(nextSeq uint32, available []uint32) *ua.PublishResponse {
	return &ua.PublishResponse{
		ResponseHeader: newResponseHeader(0, ua.StatusOK),
		SubscriptionID: SubID,
		NotificationMessage: &ua.NotificationMessage{
			SequenceNumber:   nextSeq,
			PublishTime:      time.Now(),
			NotificationData: []*ua.ExtensionObject{},
		},
		AvailableSequenceNumbers: available,
		Results:                  []ua.StatusCode{},
		DiagnosticInfos:          []*ua.DiagnosticInfo{},
	}
}

func anonymousAuthEndpoint(url string) *ua.EndpointDescription {
	return &ua.EndpointDescription{
		EndpointURL: url,
		Server: &ua.ApplicationDescription{
			ApplicationURI:  "urn:reconnect:server",
			ProductURI:      "urn:reconnect:server",
			ApplicationName: ua.NewLocalizedText("reconnect"),
			ApplicationType: ua.ApplicationTypeServer,
			DiscoveryURLs:   []string{url},
		},
		SecurityMode:      ua.MessageSecurityModeNone,
		SecurityPolicyURI: ua.SecurityPolicyURINone,
		UserIdentityTokens: []*ua.UserTokenPolicy{{
			PolicyID:  "anonymous_none",
			TokenType: ua.UserTokenTypeAnonymous,
		}},
		TransportProfileURI: "http://opcfoundation.org/UA-Profile/Transport/uatcp-uasc-uabinary",
	}
}

// scenarioOptions configures one scenario run.
type scenarioOptions struct {
	// RefuseAfterDrop closes every connection accepted once the drop has started.
	RefuseAfterDrop bool
	// ReconnectInterval is the client's wait between reconnection attempts;
	// zero means 100ms.
	ReconnectInterval time.Duration
	// Settle is how long to wait after the drop before snapshotting; zero means
	// settleDefault.
	Settle time.Duration
}

// runScenario drives one reconnect scenario against the scripted server and
// returns the record the test reads. It never aborts: a stage it could not
// reach is recorded in HarnessErr.
func runScenario(opts scenarioOptions) record {
	if opts.Settle == 0 {
		opts.Settle = settleDefault
	}
	if opts.ReconnectInterval == 0 {
		opts.ReconnectInterval = 100 * time.Millisecond
	}
	var out record

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, err := newServer(ctx, opts.RefuseAfterDrop)
	if err != nil {
		out.HarnessErr = fmt.Sprintf("the scripted server could not start: %v", err)
		return out
	}
	defer func() { srv.ln.Close() }()

	c, err := opcua.NewClient(srv.endpoint,
		opcua.SecurityMode(ua.MessageSecurityModeNone),
		opcua.AutoReconnect(true),
		opcua.ReconnectInterval(opts.ReconnectInterval),
		opcua.RequestTimeout(3*time.Second),
		opcua.DialTimeout(3*time.Second),
	)
	if err != nil {
		out.HarnessErr = fmt.Sprintf("constructing the client: %v", err)
		return out
	}
	if err := c.Connect(ctx); err != nil {
		out.HarnessErr = fmt.Sprintf("the initial connect failed: %v", err)
		return out
	}
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = c.Close(closeCtx)
		closeCancel()
	}()

	// Records what the application's own notification callback receives.
	var deliveriesMu sync.Mutex
	var deliveries []deliveryRecord
	notifs := make(chan *opcua.PublishNotificationData, 32)
	sub, err := c.Subscribe(ctx, &opcua.SubscriptionParameters{Interval: 200 * time.Millisecond}, notifs)
	if err != nil {
		out.HarnessErr = fmt.Sprintf("subscribing: %v", err)
		return out
	}
	go func() {
		for n := range notifs {
			dcn, ok := n.Value.(*ua.DataChangeNotification)
			if !ok || len(dcn.MonitoredItems) == 0 {
				continue
			}
			if seq, ok := dcn.MonitoredItems[0].Value.Value.Value().(int32); ok {
				deliveriesMu.Lock()
				deliveries = append(deliveries, deliveryRecord{Seq: uint32(seq), At: time.Now()})
				deliveriesMu.Unlock()
			}
		}
	}()

	item := opcua.NewMonitoredItemCreateRequestWithDefaults(monitoredNode, ua.AttributeIDValue, 1)
	if _, err := sub.Monitor(ctx, ua.TimestampsToReturnBoth, item); err != nil {
		out.HarnessErr = fmt.Sprintf("monitoring the item: %v", err)
		return out
	}

	// Wait until the stream is live: the application's callback has received
	// a notification.
	deadline := time.After(10 * time.Second)
	for {
		deliveriesMu.Lock()
		n := len(deliveries)
		deliveriesMu.Unlock()
		if n > 0 {
			break
		}
		select {
		case <-deadline:
			out.HarnessErr = "no notification reached the application before the drop command"
			return out
		case <-time.After(20 * time.Millisecond):
		}
	}

	// Wait until the server is holding a Publish, so that its notification
	// cannot reach the client before the drop.
	heldDeadline := time.After(10 * time.Second)
	for {
		if srv.rec.heldPublishSeq() != 0 {
			break
		}
		select {
		case <-heldDeadline:
			out.HarnessErr = "the client never left a publish outstanding for the server to hold; the loss could not be constructed"
			return out
		case <-time.After(10 * time.Millisecond):
		}
	}

	dropAt := time.Now()
	srv.drop()
	dropDeadline := time.After(2 * time.Second)
	for {
		_, _, dropApplied, _ := srv.rec.snapshot()
		if dropApplied {
			out.DropApplied = true
			out.DropAt = dropAt
			out.DropEpoch = srv.rec.heldPublishEpoch()
			break
		}
		select {
		case <-dropDeadline:
			out.HarnessErr = "the commanded transport drop never applied"
			return out
		case <-time.After(10 * time.Millisecond):
		}
	}

	time.Sleep(opts.Settle)

	deliveriesMu.Lock()
	out.Deliveries = append([]deliveryRecord(nil), deliveries...)
	deliveriesMu.Unlock()
	requests, accepts, _, held := srv.rec.snapshot()
	out.Requests = requests
	out.Accepts = accepts
	out.HeldSeq = held
	out.LastSeq = srv.rec.highestSeq()
	return out
}
