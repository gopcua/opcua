package monitor

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/ua"
)

var (
	// DefaultCallbackBufferLen is the size of the internal buffer when using a callback-based subscription
	DefaultCallbackBufferLen = 8192

	// ErrSlowConsumer is returned when a subscriber does not keep up with the incoming messages
	ErrSlowConsumer = errors.New("slow consumer. messages may be dropped")
)

// ErrHandler is a function that is called when there is an out of band issue with delivery
type ErrHandler func(*opcua.Client, *Subscription, error)

// MsgHandler is a function that is called for each new DataValue
type MsgHandler func(*Subscription, *DataChangeMessage)

// DataChangeMessage represents the changed DataValue from the server. It also includes a reference
// to the sending NodeID and error (if any)
type DataChangeMessage struct {
	*ua.DataValue
	Error  error
	NodeID *ua.NodeID
}

// NodeMonitor creates new subscriptions
type NodeMonitor struct {
	client           *opcua.Client
	nextClientHandle uint32
	errHandlerCB     ErrHandler
}

// internal struct to manage various ids
type itemIDs struct {
	handle uint32 // client-provided
	id     uint32 // from server
}

// Subscription is an instance of an active subscription.
// Nodes can be added and removed concurrently.
type Subscription struct {
	monitor          *NodeMonitor
	sub              *opcua.Subscription
	internalNotifyCh chan *opcua.PublishNotificationData
	delivered        uint64
	dropped          uint64
	closed           chan struct{}
	mu               sync.RWMutex
	handles          map[uint32]*ua.NodeID
	nodeLookup       map[string]*itemIDs
}

// NewNodeMonitor creates a new NodeMonitor
func NewNodeMonitor(client *opcua.Client) (*NodeMonitor, error) {
	m := &NodeMonitor{
		client:           client,
		nextClientHandle: 100,
	}

	return m, nil
}

func newSubscription(m *NodeMonitor, params *opcua.SubscriptionParameters, notifyChanLength int, nodes ...string) (*Subscription, error) {
	if params == nil {
		params = &opcua.SubscriptionParameters{}
	}

	s := &Subscription{
		monitor:          m,
		closed:           make(chan struct{}),
		internalNotifyCh: make(chan *opcua.PublishNotificationData, notifyChanLength),
		handles:          make(map[uint32]*ua.NodeID),
		nodeLookup:       make(map[string]*itemIDs),
	}

	var err error
	if s.sub, err = m.client.Subscribe(params, s.internalNotifyCh); err != nil {
		return nil, err
	}

	if err = s.AddNodes(nodes...); err != nil {
		return nil, err
	}

	return s, nil
}

// SetErrorHandler sets an optional callback for async errors
func (m *NodeMonitor) SetErrorHandler(cb ErrHandler) {
	m.errHandlerCB = cb
}

// Subscribe creates a new callback-based subscription and an optional list of nodes.
// The caller must call `Unsubscribe` to stop and clean up resources. Canceling the context
// will also cause the subscription to stop, but `Unsubscribe` must still be called.
func (m *NodeMonitor) Subscribe(ctx context.Context, params *opcua.SubscriptionParameters, cb MsgHandler, nodes ...string) (*Subscription, error) {
	sub, err := newSubscription(m, params, DefaultCallbackBufferLen, nodes...)
	if err != nil {
		return nil, err
	}

	go sub.pump(ctx, nil, cb)
	go sub.sub.Run(ctx)

	return sub, nil
}

// ChanSubscribe creates a new channel-based subscription and an optional list of nodes.
// The channel should be deep enough to allow some buffering, otherwise `ErrSlowConsumer` is sent
// via the monitor's `ErrHandler`.
// The caller must call `Unsubscribe` to stop and clean up resources. Canceling the context
// will also cause the subscription to stop, but `Unsubscribe` must still be called.
func (m *NodeMonitor) ChanSubscribe(ctx context.Context, params *opcua.SubscriptionParameters, ch chan<- *DataChangeMessage, nodes ...string) (*Subscription, error) {
	sub, err := newSubscription(m, params, 16, nodes...)
	if err != nil {
		return nil, err
	}

	go sub.pump(ctx, ch, nil)
	go sub.sub.Run(ctx)

	return sub, nil
}

func (s *Subscription) sendError(err error) {
	if err != nil && s.monitor.errHandlerCB != nil {
		go s.monitor.errHandlerCB(s.monitor.client, s, err)
	}
}

// internal func to read from internal channel and write to client provided channel
func (s *Subscription) pump(ctx context.Context, notifyCh chan<- *DataChangeMessage, cb MsgHandler) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.closed:
			return
		case msg := <-s.internalNotifyCh:
			if msg.Error != nil {
				// TODO: is it possible to have an error _and_ some DataChangeNotification values?
				s.sendError(msg.Error)
				continue
			}

			if msg.SubscriptionID != s.sub.SubscriptionID {
				s.sendError(errors.Errorf("message sub id %v does not match sub id %v", msg.SubscriptionID, s.sub.SubscriptionID))
				continue
			}

			// this is sort of a hack to emulate an `ErrSlowConsumer` error from the underlying subscription
			// we check to see if the channel is "full" from the outside, and bail if it is.
			if cb != nil && cap(s.internalNotifyCh) > 0 {
				if len(s.internalNotifyCh) == cap(s.internalNotifyCh) {
					s.sendError(ErrSlowConsumer)
					atomic.AddUint64(&s.dropped, 1)
					continue
				}
			}

			switch v := msg.Value.(type) {
			case *ua.DataChangeNotification:
				for _, item := range v.MonitoredItems {
					s.mu.RLock()
					nid, ok := s.handles[item.ClientHandle]
					s.mu.RUnlock()

					out := &DataChangeMessage{}

					if !ok {
						out.Error = errors.Errorf("handle %d not found", item.ClientHandle)
						// TODO: should the error also propagate via the monitor callback?
					} else {
						out.NodeID = nid
						out.DataValue = item.Value
					}

					if notifyCh != nil {
						select {
						case notifyCh <- out:
							atomic.AddUint64(&s.delivered, 1)
						default:
							atomic.AddUint64(&s.dropped, 1)
							s.sendError(ErrSlowConsumer)
						}
					} else if cb != nil {
						cb(s, out)
						atomic.AddUint64(&s.delivered, 1)
					} else {
						panic("notifyCh or cb must be set")
					}
				}
			default:
				s.sendError(errors.Errorf("unknown message type: %T", msg.Value))
			}
		}
	}
}

// Unsubscribe removes the subscription interests and cleans up any resources
func (s *Subscription) Unsubscribe() error {
	// TODO: make idempotent
	close(s.closed)
	return s.sub.Cancel()
}

// Subscribed returns the number of currently subscribed to nodes
func (s *Subscription) Subscribed() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.handles)
}

// SubscriptionID returns the underlying subscription id
func (s *Subscription) SubscriptionID() uint32 {
	return s.sub.SubscriptionID
}

// Delivered returns the number of DataChangeMessages delivered
func (s *Subscription) Delivered() uint64 {
	return atomic.LoadUint64(&s.delivered)
}

// Dropped returns the number of DataChangeMessages dropped due to a slow consumer
func (s *Subscription) Dropped() uint64 {
	return atomic.LoadUint64(&s.dropped)
}

// AddNodes adds nodes defined by their string representation
func (s *Subscription) AddNodes(nodes ...string) error {
	nodeIDs, err := parseNodeSlice(nodes...)
	if err != nil {
		return err
	}
	return s.AddNodeIDs(nodeIDs...)
}

// AddNodeIDs adds nodes
func (s *Subscription) AddNodeIDs(nodes ...*ua.NodeID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(nodes) == 0 {
		// some server implementionations allow an empty monitoreditemrequest, some don't.
		// beter to just return
		return nil
	}

	toAdd := make([]*ua.MonitoredItemCreateRequest, 0)

	for _, node := range nodes {
		handle := atomic.AddUint32(&s.monitor.nextClientHandle, 1)

		s.handles[handle] = node
		s.nodeLookup[node.String()] = &itemIDs{
			handle: handle,
		}

		// log.Printf("node=%s handle=%d", node.String(), handle)

		toAdd = append(toAdd, opcua.NewMonitoredItemCreateRequestWithDefaults(node, ua.AttributeIDValue, handle))
	}

	resp, err := s.sub.Monitor(ua.TimestampsToReturnBoth, toAdd...)
	if err != nil {
		return err
	}

	if resp.ResponseHeader.ServiceResult != ua.StatusOK {
		return resp.ResponseHeader.ServiceResult
	}

	if len(resp.Results) != len(toAdd) {
		return errors.Errorf("monitor items response length mismatch")
	}

	for i, res := range resp.Results {
		if res.StatusCode != ua.StatusOK {
			return res.StatusCode
		}
		// note: this works _iff_ the order of the response is the same as the request
		sid := toAdd[i].ItemToMonitor.NodeID.String()
		s.nodeLookup[sid].id = res.MonitoredItemID
	}

	return nil
}

// RemoveNodes removes nodes defined by their string representation
func (s *Subscription) RemoveNodes(nodes ...string) error {
	nodeIDs, err := parseNodeSlice(nodes...)
	if err != nil {
		return err
	}
	return s.RemoveNodeIDs(nodeIDs...)
}

// RemoveNodeIDs removes nodes
func (s *Subscription) RemoveNodeIDs(nodes ...*ua.NodeID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(nodes) == 0 {
		return nil
	}

	toRemove := make([]uint32, len(nodes))

	for i, node := range nodes {
		sid := node.String()
		ids, ok := s.nodeLookup[sid]
		if !ok {
			return errors.Errorf("node not found: %s", sid)
		}
		delete(s.nodeLookup, sid)
		delete(s.handles, ids.handle)

		toRemove[i] = ids.id
	}

	resp, err := s.sub.Unmonitor(toRemove...)
	if err != nil {
		return err
	}

	if resp.ResponseHeader.ServiceResult != ua.StatusOK {
		return resp.ResponseHeader.ServiceResult
	}

	if len(resp.Results) != len(toRemove) {
		return errors.Errorf("unmonitor items response length mismatch")
	}

	for _, res := range resp.Results {
		if res != ua.StatusOK {
			return res
		}
	}

	return nil
}

func (s *Subscription) Stats() (*ua.SubscriptionDiagnosticsDataType, error) {
	return s.sub.Stats()
}

func parseNodeSlice(nodes ...string) ([]*ua.NodeID, error) {
	var err error

	nodeIDs := make([]*ua.NodeID, len(nodes))

	for i, node := range nodes {
		if nodeIDs[i], err = ua.ParseNodeID(node); err != nil {
			return nil, err
		}
	}

	return nodeIDs, nil
}
