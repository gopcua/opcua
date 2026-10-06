package spectest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
	ginkgo "github.com/onsi/ginkgo/v2"
)

const (
	defaultPublishingInterval = 100 * time.Millisecond
	defaultReconnectInterval  = 50 * time.Millisecond
	clientRequestTimeout      = 5 * time.Second
	startTimeout              = 15 * time.Second
	teardownWait              = 10 * time.Second
	statePollInterval         = 10 * time.Millisecond
	notificationBuffer        = 64
	lifetimeCount             = 1_000_000
	maxKeepAliveCount         = 1000
	monitorClientHandle       = 1
	publishingIntervalEnv     = "SPECTEST_PUBLISHING_INTERVAL"
	reconnectIntervalEnv      = "SPECTEST_RECONNECT_INTERVAL"
)

const valueBeforeCut int32 = 9001

// Option changes how Start builds the Environment.
type Option func(*options)

type options struct {
	clientOptions         []opcua.Option
	publishingInterval    time.Duration
	publishingIntervalSet bool
	retention             bool
	notificationBuffer    int
	drainGap              time.Duration
}

// WithClientOptions appends client options after the ones Start sets itself.
func WithClientOptions(opts ...opcua.Option) Option {
	return func(o *options) {
		o.clientOptions = append(o.clientOptions, opts...)
	}
}

// WithRetentionQueue makes every server Start or StartServer creates
// keep each value it answers in the subscription's retransmission
// queue until a Publish acknowledges it, as Part 4 §5.14.1 describes.
// Without the option an answered value is gone from the queue
// immediately.
func WithRetentionQueue() Option {
	return func(o *options) {
		o.retention = true
	}
}

// WithNotificationBuffer sets the capacity of the channel the client
// delivers the subscription's notifications to. With a small buffer and
// WithSlowConsumer the channel fills and the client blocks writing to
// it, which ConsumerBlocked reports.
func WithNotificationBuffer(n int) Option {
	return func(o *options) {
		o.notificationBuffer = n
	}
}

// WithSlowConsumer makes the environment's notification drain read one
// notification per gap, so notifications pile up in the channel.
func WithSlowConsumer(gap time.Duration) Option {
	return func(o *options) {
		o.drainGap = gap
	}
}

// WithPublishingInterval sets the publishing interval of the subscription
// Start creates.
func WithPublishingInterval(d time.Duration) Option {
	return func(o *options) {
		o.publishingInterval = d
		o.publishingIntervalSet = true
	}
}

// Environment is a client connected to a scripted server through the
// relay, with one subscription monitored.
type Environment struct {
	Client   *opcua.Client
	Relay    *Relay
	Server   *ScriptedServer
	Recorder *Recorder

	t                  T
	mu                 sync.Mutex
	clientSubscription *opcua.Subscription
	received           []int32
	receivedErrors     []error
	states             []opcua.ConnState
	retention          bool
	drainGap           time.Duration
	consumerBlocked    bool
	publishingInterval time.Duration
	reconnectInterval  time.Duration
	receivedSignal     chan struct{}
	everConnected      bool
	cutStates          int
	waitTimeout        time.Duration
	servers            []*ScriptedServer
	onClientClosed     func()
}

// Start creates and returns a running Environment (see the Environment
// type) and answers the first held Publish request (see HeldPublish)
// with valueBeforeCut.
func Start(t T, opts ...Option) *Environment {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	publishing, reconnect, err := resolveIntervals(o, os.Getenv)
	if err != nil {
		t.Fatalf("%s", harnessFault("%v", err))
		return nil
	}
	if publishing*time.Duration(lifetimeCount) < time.Hour {
		t.Fatalf("%s", harnessFault("publishing interval %s times lifetime count %d stays below one hour, so the server's subscription service could delete the subscription before the test ends", publishing, lifetimeCount))
		return nil
	}
	e := &Environment{t: t, receivedSignal: make(chan struct{}, 1), retention: o.retention, drainGap: o.drainGap, publishingInterval: publishing, reconnectInterval: reconnect}
	e.Server = newScriptedServer(t)
	e.Server.retention = o.retention
	e.Relay, e.Recorder = newRelay(t, e.Server.Address(), e.noteCut)
	e.Server.mu.Lock()
	e.Server.recorder = e.Recorder
	e.Server.mu.Unlock()

	buffer := notificationBuffer
	if o.notificationBuffer > 0 {
		buffer = o.notificationBuffer
	}
	states := make(chan opcua.ConnState, notificationBuffer)
	notifications := make(chan *opcua.PublishNotificationData, buffer)
	drained := make(chan struct{})
	go e.drain(notifications, drained)
	go e.drainStates(states, drained)
	t.Cleanup(func() { e.teardown(drained) })

	clientOptions := append([]opcua.Option{
		opcua.SecurityMode(ua.MessageSecurityModeNone),
		opcua.AutoReconnect(true),
		opcua.ReconnectInterval(reconnect),
		opcua.RequestTimeout(clientRequestTimeout),
		opcua.StateChangedCh(states),
	}, o.clientOptions...)
	client, err := opcua.NewClient("opc.tcp://"+e.Relay.address(), clientOptions...)
	if err != nil {
		t.Fatalf("%s", harnessFault("creating the client failed: %v", err))
		return nil
	}
	e.Client = client

	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	connectErr := client.Connect(ctx)
	cancel()
	if connectErr != nil {
		t.Fatalf("the client never connected through the relay: %v", connectErr)
		return nil
	}
	e.everConnected = true

	ctx, cancel = context.WithTimeout(context.Background(), startTimeout)
	subscription, err := client.Subscribe(ctx, &opcua.SubscriptionParameters{
		Interval:          publishing,
		LifetimeCount:     lifetimeCount,
		MaxKeepAliveCount: maxKeepAliveCount,
	}, notifications)
	cancel()
	if err != nil {
		t.Fatalf("the client created no subscription: %v", err)
		return nil
	}
	e.clientSubscription = subscription
	ctx, cancel = context.WithTimeout(context.Background(), startTimeout)
	_, monitorErr := subscription.Monitor(ctx, ua.TimestampsToReturnBoth,
		opcua.NewMonitoredItemCreateRequestWithDefaults(e.Server.node, ua.AttributeIDValue, monitorClientHandle))
	cancel()
	if monitorErr != nil {
		t.Fatalf("the client monitored no node: %v", monitorErr)
		return nil
	}
	e.Server.WaitHeldPublish().Answer(e.Subscription(), valueBeforeCut)

	timer := time.NewTimer(startTimeout)
	defer timer.Stop()
	for len(e.Received()) == 0 {
		if errs := e.ReceivedErrors(); len(errs) > 0 {
			t.Fatalf("the client did not deliver the pre-cut value within %s; errors: %v", startTimeout, errs)
		}
		select {
		case <-e.receivedSignal:
		case <-timer.C:
			t.Fatalf("the client did not deliver the pre-cut value within %s", startTimeout)
		}
	}
	e.checkLastSequenceNumber()
	return e
}

// Mark holds the position of the recorder's records, the relay's
// accepted connections and the Environment's received values, received
// errors and reported states, so the Since methods return only what
// arrived after it.
type Mark struct {
	order          int
	connections    int
	received       int
	receivedErrors int
	states         int
}

// Mark returns the current position of every stream. Mark reads each
// position separately; take it while no traffic flows, as every spec
// does before a cut.
func (e *Environment) Mark() Mark {
	e.mu.Lock()
	defer e.mu.Unlock()
	return Mark{
		order:          e.Recorder.position(),
		connections:    e.Relay.ConnectionCount(),
		received:       len(e.received),
		receivedErrors: len(e.receivedErrors),
		states:         len(e.states),
	}
}

// Subscription returns the subscription the client created before
// Start answered the first held Publish request (HeldPublish).
func (e *Environment) Subscription() Subscription {
	e.Server.mu.Lock()
	defer e.Server.mu.Unlock()
	if e.Server.first == nil {
		e.t.Fatalf("the client created no subscription")
	}
	return Subscription{server: e.Server, sub: e.Server.first}
}

// ClientSubscription returns the subscription Start created on the
// client, whose Notifs channel feeds the Environment's received values.
func (e *Environment) ClientSubscription() *opcua.Subscription {
	return e.clientSubscription
}

// Servers returns every server the environment created: its own and
// every one StartServer made, in creation order.
func (e *Environment) Servers() []*ScriptedServer {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]*ScriptedServer{e.Server}, e.servers...)
}

// UpstreamServer returns the scripted server the relay's next
// connection dials: the server a RedirectTo pointed at, or the
// environment's own when no redirect is in place.
func (e *Environment) UpstreamServer() *ScriptedServer {
	upstream := strings.TrimPrefix(e.Relay.Upstream(), "opc.tcp://")
	for _, server := range e.Servers() {
		if strings.TrimPrefix(server.Address(), "opc.tcp://") == upstream {
			return server
		}
	}
	return e.Server
}

// StartServer starts a second, independent scripted server with the
// same namespace and node as the Environment's own, sharing the
// Environment's recorder. Its traffic flows through the relay only
// after RedirectTo sends new connections to it. The Environment's
// teardown closes it, after the client and the relay.
func (e *Environment) StartServer() *ScriptedServer {
	second := newScriptedServer(e.t)
	second.retention = e.retention
	second.mu.Lock()
	second.recorder = e.Recorder
	second.mu.Unlock()
	e.mu.Lock()
	e.servers = append(e.servers, second)
	e.mu.Unlock()
	return second
}

// WaitUntilReconnected waits for the client to pass through
// Reconnecting back to Connected after the most recent cut the relay
// made, by Relay.Cut or by a fired CutAt, searching the states
// recorded since that cut. The deadline starts when this method is
// called, and the Connected wait restarts its deadline once
// Reconnecting appears.
func (e *Environment) WaitUntilReconnected() {
	e.mu.Lock()
	cut := e.cutStates
	waitTimeout := e.waitTimeout
	e.mu.Unlock()
	if waitTimeout == 0 {
		waitTimeout = startTimeout
	}
	if !e.hasStateSince(cut, opcua.Reconnecting) {
		deadline := time.Now().Add(waitTimeout)
		for !e.hasStateSince(cut, opcua.Reconnecting) {
			if !time.Now().Before(deadline) {
				e.t.Fatalf("client never entered Reconnecting within %s of the cut", waitTimeout)
				return
			}
			time.Sleep(statePollInterval)
		}
	}
	reconnecting := e.lastStateIndex(opcua.Reconnecting)
	if e.hasStateSince(reconnecting+1, opcua.Connected) {
		return
	}
	deadline := time.Now().Add(waitTimeout)
	for !e.hasStateSince(reconnecting+1, opcua.Connected) {
		if !time.Now().Before(deadline) {
			e.t.Fatalf("client entered Reconnecting but did not reach Connected within %s; states: %v", waitTimeout, e.statesFrom(reconnecting+1))
			return
		}
		time.Sleep(statePollInterval)
	}
}

// TryWaitUntilReconnected waits up to timeout for the client to pass
// through Reconnecting back to Connected after the most recent cut the
// relay made, by Relay.Cut or by a fired CutAt, and reports whether it
// did: the failure-matrix workloads bound every wait instead of
// failing the spec.
func (e *Environment) TryWaitUntilReconnected(timeout time.Duration) bool {
	e.mu.Lock()
	cut := e.cutStates
	e.mu.Unlock()
	if !e.hasStateSince(cut, opcua.Reconnecting) {
		deadline := time.Now().Add(timeout)
		for !e.hasStateSince(cut, opcua.Reconnecting) {
			if !time.Now().Before(deadline) {
				return false
			}
			time.Sleep(statePollInterval)
		}
	}
	reconnecting := e.lastStateIndex(opcua.Reconnecting)
	if e.hasStateSince(reconnecting+1, opcua.Connected) {
		return true
	}
	deadline := time.Now().Add(timeout)
	for !e.hasStateSince(reconnecting+1, opcua.Connected) {
		if !time.Now().Before(deadline) {
			return false
		}
		time.Sleep(statePollInterval)
	}
	return true
}

func (e *Environment) lastStateIndex(state opcua.ConnState) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i := len(e.states) - 1; i >= 0; i-- {
		if e.states[i] == state {
			return i
		}
	}
	return -1
}

func (e *Environment) noteCut() {
	e.mu.Lock()
	e.cutStates = len(e.states)
	e.mu.Unlock()
}

// Received returns the int32 values of the data change notifications the
// client delivered, in delivery order.
func (e *Environment) Received() []int32 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.received)
}

// ReceivedSince returns the int32 values the client delivered after m,
// in delivery order.
func (e *Environment) ReceivedSince(m Mark) []int32 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.received[min(m.received, len(e.received)):])
}

// ReceivedErrors returns the errors the client delivered, in delivery
// order.
func (e *Environment) ReceivedErrors() []error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.receivedErrors)
}

// ReceivedErrorsSince returns the errors the client delivered after m,
// in delivery order.
func (e *Environment) ReceivedErrorsSince(m Mark) []error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.receivedErrors[min(m.receivedErrors, len(e.receivedErrors)):])
}

// ConnectionsSince returns how many connections the relay accepted
// after m.
func (e *Environment) ConnectionsSince(m Mark) int {
	return e.Relay.ConnectionCount() - m.connections
}

// States returns the client connection states the client reported, in
// the order it reported them.
func (e *Environment) States() []opcua.ConnState {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.states)
}

// StatesSince returns the client connection states the client reported
// after m, in the order it reported them.
func (e *Environment) StatesSince(m Mark) []opcua.ConnState {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.states[min(m.states, len(e.states)):])
}

// LastSequenceNumber returns the highest sequence number of the data
// change notifications the recorder saw the server deliver, in a
// Publish response or a Republish response.
func (e *Environment) LastSequenceNumber() uint32 {
	var highest uint32
	for _, candidate := range e.Recorder.answeredPublishes() {
		if candidate.sequenceNumber > highest {
			highest = candidate.sequenceNumber
		}
	}
	return highest
}

func (e *Environment) checkLastSequenceNumber() {
	var last answeredPublish
	seen := false
	for _, candidate := range e.Recorder.answeredPublishes() {
		if !candidate.republished {
			last = candidate
			seen = true
		}
	}
	if !seen {
		e.t.Fatalf("%s", harnessFault("the recorder saw no answered Publish response"))
		return
	}
	received := e.Received()
	if len(received) == 0 || last.value != received[len(received)-1] {
		e.t.Fatalf("%s", harnessFault("the last answered Publish response carries %d, want the value the client delivered", last.value))
	}
}

func resolveIntervals(o options, getenv func(string) string) (time.Duration, time.Duration, error) {
	publishing := o.publishingInterval
	if !o.publishingIntervalSet {
		if value := getenv(publishingIntervalEnv); value != "" {
			parsed, err := time.ParseDuration(value)
			if err != nil {
				return 0, 0, fmt.Errorf("%s: %w", publishingIntervalEnv, err)
			}
			publishing = parsed
		} else {
			publishing = defaultPublishingInterval
		}
	}
	reconnect := defaultReconnectInterval
	if value := getenv(reconnectIntervalEnv); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return 0, 0, fmt.Errorf("%s: %w", reconnectIntervalEnv, err)
		}
		if parsed <= 0 {
			return 0, 0, fmt.Errorf("%s: %s is not a positive duration", reconnectIntervalEnv, value)
		}
		reconnect = parsed
	}
	return publishing, reconnect, nil
}

func (e *Environment) hasStateSince(start int, want opcua.ConnState) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, state := range e.states[min(start, len(e.states)):] {
		if state == want {
			return true
		}
	}
	return false
}

func (e *Environment) statesFrom(start int) []opcua.ConnState {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.states[min(start, len(e.states)):])
}

func (e *Environment) drain(notifications <-chan *opcua.PublishNotificationData, drained <-chan struct{}) {
	noteIfFull := func() {
		if cap(notifications) > 0 && len(notifications) == cap(notifications) {
			e.mu.Lock()
			e.consumerBlocked = true
			e.mu.Unlock()
		}
	}
	for {
		select {
		case <-drained:
			return
		case notification := <-notifications:
			e.accept(notification)
			noteIfFull()
			if e.drainGap > 0 {
				select {
				case <-drained:
					return
				case <-time.After(e.drainGap):
				}
				noteIfFull()
			}
		}
	}
}

// ConsumerBlocked reports whether the environment's notification drain
// ever saw the notification channel full, which means the client was
// blocked writing to it.
func (e *Environment) ConsumerBlocked() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.consumerBlocked
}

// RequestTimeout returns the request timeout the client runs with.
func (e *Environment) RequestTimeout() time.Duration {
	return e.Client.RequestTimeout()
}

// PublishTimeout returns the Publish timeout the client computes for
// its subscription: the keep-alive count times the publishing
// interval, at least the request timeout.
func (e *Environment) PublishTimeout() time.Duration {
	timeout := time.Duration(maxKeepAliveCount) * e.publishingInterval
	if timeout > uasc.MaxTimeout {
		return uasc.MaxTimeout
	}
	if timeout < e.RequestTimeout() {
		return e.RequestTimeout()
	}
	return timeout
}

// ReconnectInterval returns the interval the client waits between two
// reconnection attempts.
func (e *Environment) ReconnectInterval() time.Duration {
	return e.reconnectInterval
}

func (e *Environment) drainStates(states <-chan opcua.ConnState, drained <-chan struct{}) {
	for {
		select {
		case <-drained:
			return
		case state := <-states:
			e.mu.Lock()
			e.states = append(e.states, state)
			e.mu.Unlock()
		}
	}
}

func (e *Environment) accept(notification *opcua.PublishNotificationData) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if notification.Error != nil {
		e.receivedErrors = append(e.receivedErrors, notification.Error)
	} else if change, isDataChange := notification.Value.(*ua.DataChangeNotification); isDataChange {
		for _, item := range change.MonitoredItems {
			if item == nil || item.Value == nil {
				continue
			}
			if item.Value.Value == nil {
				e.receivedErrors = append(e.receivedErrors, fmt.Errorf("the client delivered a data change with no value"))
				continue
			}
			value := item.Value.Value.Value()
			if number, isInt32 := value.(int32); isInt32 {
				e.received = append(e.received, number)
			} else {
				e.receivedErrors = append(e.receivedErrors, fmt.Errorf("the client delivered a data change carrying %T, want int32", value))
			}
		}
	} else {
		e.receivedErrors = append(e.receivedErrors, fmt.Errorf("the client delivered a %T, want a data change notification", notification.Value))
	}
	select {
	case e.receivedSignal <- struct{}{}:
	default:
	}
}

func (e *Environment) teardown(drained chan struct{}) {
	if e.Client == nil {
		close(drained)
		return
	}
	if e.everConnected && e.Client.State() != opcua.Closed {
		deadline := time.Now().Add(teardownWait)
		for e.Client.State() != opcua.Connected && time.Now().Before(deadline) {
			time.Sleep(statePollInterval)
		}
		if e.Client.State() != opcua.Connected {
			ginkgo.GinkgoWriter.Printf("the client did not reach the connected state within %s before teardown\n", teardownWait)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), teardownWait)
	defer cancel()
	if err := e.Client.Close(ctx); err != nil {
		ginkgo.GinkgoWriter.Printf("closing the client failed: %v\n", err)
	}
	if e.onClientClosed != nil {
		e.onClientClosed()
	}
	e.Relay.close()
	e.Server.close()
	e.mu.Lock()
	servers := e.servers
	e.mu.Unlock()
	for _, extra := range servers {
		extra.close()
	}
	var faults []error
	for _, scripted := range append([]*ScriptedServer{e.Server}, servers...) {
		scripted.mu.Lock()
		faultErr := scripted.faultErr
		scripted.mu.Unlock()
		if faultErr != nil {
			faults = append(faults, faultErr)
		}
	}
	close(drained)
	if len(faults) > 0 {
		e.t.Fatalf("%s", errors.Join(faults...))
	}
}

type answeredPublish struct {
	order                    int
	connection               int
	subscriptionID           uint32
	sequenceNumber           uint32
	republished              bool
	value                    int32
	results                  []ua.StatusCode
	availableSequenceNumbers []uint32
}

func answeredPublishes(requests []ServiceRecord[ua.Request], responses []ServiceRecord[ua.Response]) []answeredPublish {
	var answered []answeredPublish
	for _, record := range responses {
		message, forwarded := record.Message()
		if !forwarded {
			continue
		}
		var notification *ua.NotificationMessage
		var subscriptionID uint32
		var results []ua.StatusCode
		var availableSequenceNumbers []uint32
		republished := false
		switch response := message.(type) {
		case *ua.PublishResponse:
			notification = response.NotificationMessage
			subscriptionID = response.SubscriptionID
			results = response.Results
			availableSequenceNumbers = response.AvailableSequenceNumbers
		case *ua.RepublishResponse:
			notification = response.NotificationMessage
			subscriptionID = republishSubscriptionID(requests, record)
			republished = true
		default:
			continue
		}
		value, carries := dataChangeValue(notification)
		if !carries {
			continue
		}
		valueInt32, _ := value.(int32)
		answered = append(answered, answeredPublish{
			order:                    record.Order,
			connection:               record.Connection,
			subscriptionID:           subscriptionID,
			sequenceNumber:           notification.SequenceNumber,
			republished:              republished,
			value:                    valueInt32,
			results:                  results,
			availableSequenceNumbers: availableSequenceNumbers,
		})
	}
	return answered
}

func republishSubscriptionID(requests []ServiceRecord[ua.Request], response ServiceRecord[ua.Response]) uint32 {
	for _, record := range requests {
		if record.Connection != response.Connection || record.RequestID != response.RequestID {
			continue
		}
		message, forwarded := record.Message()
		if !forwarded {
			continue
		}
		if request, isRepublish := message.(*ua.RepublishRequest); isRepublish {
			return request.SubscriptionID
		}
	}
	return 0
}

func dataChangeValue(message *ua.NotificationMessage) (any, bool) {
	if message == nil {
		return nil, false
	}
	for _, data := range message.NotificationData {
		if data == nil || data.Value == nil {
			continue
		}
		change, isDataChange := data.Value.(*ua.DataChangeNotification)
		if isDataChange && len(change.MonitoredItems) > 0 &&
			change.MonitoredItems[0].Value != nil && change.MonitoredItems[0].Value.Value != nil {
			return change.MonitoredItems[0].Value.Value.Value(), true
		}
	}
	return nil, false
}
