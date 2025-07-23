package monitor

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

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
type MsgHandler func(*Subscription, Message)

// Message is an interface that can represent either a DataChangeMessage or an EventMessage
type Message interface {
	isMessage()
}

type DataChangeMessage struct {
	*ua.DataValue
	Error  error
	NodeID *ua.NodeID
}

func (DataChangeMessage) isMessage() {}

type EventMessage struct {
	EventFields []*ua.DataValue
	Error       error
}

func (EventMessage) isMessage() {}

// NodeMonitor creates new subscriptions
type NodeMonitor struct {
	client           *opcua.Client
	nextClientHandle uint32
	errHandlerCB     ErrHandler
}

// Item is a struct to manage Monitored Items
type Item struct {
	id     uint32     // from server
	nodeID *ua.NodeID // from request
	handle uint32     // client provided
}

// ID returns the MonitorItemID set by the server
func (m *Item) ID() uint32 {
	return m.id
}

// NodeID returns the NodeID for the Item
func (m *Item) NodeID() *ua.NodeID {
	return m.nodeID
}

// Request is a struct to manage a request to monitor a node or modify a monitored node
type Request struct {
	NodeID               *ua.NodeID
	MonitoringMode       ua.MonitoringMode
	MonitoringParameters *ua.MonitoringParameters
	handle               uint32
}

// Subscription is an instance of an active subscription.
// Nodes can be added and removed concurrently.
type Subscription struct {
	delivered        uint64
	dropped          uint64
	monitor          *NodeMonitor
	sub              *opcua.Subscription
	internalNotifyCh chan *opcua.PublishNotificationData
	closed           chan struct{}
	mu               sync.RWMutex
	handles          map[uint32]*ua.NodeID
	itemLookup       map[uint32]Item
}

// NewNodeMonitor creates a new NodeMonitor
func NewNodeMonitor(client *opcua.Client) (*NodeMonitor, error) {
	m := &NodeMonitor{
		client:           client,
		nextClientHandle: 100,
	}

	return m, nil
}

func newSubscription(ctx context.Context, m *NodeMonitor, params *opcua.SubscriptionParameters, notifyChanLength int, eventSub bool, filter *ua.ExtensionObject, nodes ...string) (*Subscription, error) {
	if params == nil {
		params = &opcua.SubscriptionParameters{}
	}

	s := &Subscription{
		monitor:          m,
		closed:           make(chan struct{}),
		internalNotifyCh: make(chan *opcua.PublishNotificationData, notifyChanLength),
		handles:          make(map[uint32]*ua.NodeID),
		itemLookup:       make(map[uint32]Item),
	}

	var err error
	if s.sub, err = m.client.Subscribe(ctx, params, s.internalNotifyCh); err != nil {
		return nil, err
	}

	if err = s.AddNodes(ctx, eventSub, filter, nodes...); err != nil {
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
func (m *NodeMonitor) Subscribe(ctx context.Context, params *opcua.SubscriptionParameters, cb MsgHandler, eventSub bool, filter *ua.ExtensionObject, nodes ...string) (*Subscription, error) {
	sub, err := newSubscription(ctx, m, params, DefaultCallbackBufferLen, eventSub, filter, nodes...)
	if err != nil {
		return nil, err
	}

	go sub.pump(ctx, nil, cb, true)

	return sub, nil
}

// ChanSubscribe creates a new channel-based subscription and an optional list of nodes.
// The channel should be deep enough to allow some buffering, otherwise `ErrSlowConsumer` is sent
// via the monitor's `ErrHandler`.
// The caller must call `Unsubscribe` to stop and clean up resources. Canceling the context
// will also cause the subscription to stop, but `Unsubscribe` must still be called.
func (m *NodeMonitor) ChanSubscribe(ctx context.Context, params *opcua.SubscriptionParameters, ch chan<- Message, eventSub bool, filter *ua.ExtensionObject, nodes ...string) (*Subscription, error) {
	sub, err := newSubscription(ctx, m, params, 16, eventSub, filter, nodes...)
	if err != nil {
		return nil, err
	}

	go sub.pump(ctx, ch, nil, true)

	return sub, nil
}

func (s *Subscription) sendError(err error) {
	if err != nil && s.monitor.errHandlerCB != nil {
		go s.monitor.errHandlerCB(s.monitor.client, s, err)
	}
}

// internal func to read from internal channel and write to client provided channel
func (s *Subscription) pump(ctx context.Context, notifyCh chan<- Message, cb MsgHandler, sub bool) {
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

			case *ua.EventNotificationList:
				for _, item := range v.Events {
					s.mu.RLock()
					_, ok := s.handles[item.ClientHandle]
					s.mu.RUnlock()

					out := &EventMessage{}
					if !ok {
						out.Error = errors.Errorf("handle %d not found", item.ClientHandle)
						// TODO: should the error also propagate via the monitor callback?
					} else {
						// Initialize the EventFields slice with the correct size
						out.EventFields = make([]*ua.DataValue, len(item.EventFields))

						for i, field := range item.EventFields {
							// Create a new DataValue
							dataValue := &ua.DataValue{
								Value:           field,
								Status:          ua.StatusOK,
								SourceTimestamp: time.Now(),
								ServerTimestamp: time.Now(),
							}
							out.EventFields[i] = dataValue
						}
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

// Modify modifies the subscription settings
func (s *Subscription) Modify(ctx context.Context, params *opcua.SubscriptionParameters) error {
	_, err := s.sub.ModifySubscription(ctx, *params)
	return err
}

// Unsubscribe removes the subscription interests and cleans up any resources
func (s *Subscription) Unsubscribe(ctx context.Context) error {
	// TODO: make idempotent
	close(s.closed)
	return s.sub.Cancel(ctx)
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
func (s *Subscription) AddNodes(ctx context.Context, eventSub bool, filter *ua.ExtensionObject, nodes ...string) error {
	nodeIDs, err := parseNodeSlice(nodes...)
	if err != nil {
		return err
	}
	return s.AddNodeIDs(ctx, eventSub, filter, nodeIDs...)
}

// AddNodeIDs adds nodes
func (s *Subscription) AddNodeIDs(ctx context.Context, eventSub bool, filter *ua.ExtensionObject, nodes ...*ua.NodeID) error {
	requests := make([]Request, len(nodes))

	for i, node := range nodes {
		requests[i] = Request{
			NodeID:         node,
			MonitoringMode: ua.MonitoringModeReporting,
		}
	}
	var err error
	if eventSub {
		_, err = s.AddMonitorEvents(ctx, filter, requests...)
	} else {
		_, err = s.AddMonitorItems(ctx, requests...)
	}
	return err
}

// AddMonitorItems adds nodes with monitoring parameters to the subscription
func (s *Subscription) AddMonitorItems(ctx context.Context, nodes ...Request) ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(nodes) == 0 {
		// some server implementations allow an empty monitoreditemrequest, some don't.
		// better to just return
		return nil, nil
	}

	toAdd := make([]*ua.MonitoredItemCreateRequest, 0)

	// Add handles and make requests
	for i, node := range nodes {
		handle := atomic.AddUint32(&s.monitor.nextClientHandle, 1)
		s.handles[handle] = nodes[i].NodeID
		nodes[i].handle = handle

		request := opcua.NewDefaultMonitoredItemCreateRequest(opcua.MonitoredItemCreateRequestArgs{
			NodeID:       node.NodeID,
			AttributeID:  ua.AttributeIDValue,
			ClientHandle: handle,
		})
		request.MonitoringMode = node.MonitoringMode

		if node.MonitoringParameters != nil {
			request.RequestedParameters = node.MonitoringParameters
			request.RequestedParameters.ClientHandle = handle
		}
		toAdd = append(toAdd, request)
	}
	resp, err := s.sub.Monitor(ctx, ua.TimestampsToReturnBoth, toAdd...)
	if err != nil {
		return nil, err
	}

	if resp.ResponseHeader.ServiceResult != ua.StatusOK {
		return nil, resp.ResponseHeader.ServiceResult
	}

	if len(resp.Results) != len(toAdd) {
		return nil, errors.Errorf("monitor items response length mismatch")
	}
	var monitoredItems []Item
	for i, res := range resp.Results {
		if res.StatusCode != ua.StatusOK {
			return nil, res.StatusCode
		}
		mn := Item{
			id:     res.MonitoredItemID,
			handle: nodes[i].handle,
			nodeID: toAdd[i].ItemToMonitor.NodeID,
		}
		s.itemLookup[res.MonitoredItemID] = mn
		monitoredItems = append(monitoredItems, mn)
	}

	return monitoredItems, nil
}

// AddMonitorEvents adds nodes with monitoring parameters to the subscription
func (s *Subscription) AddMonitorEvents(ctx context.Context, filter *ua.ExtensionObject, nodes ...Request) ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(nodes) == 0 {
		// some server implementionations allow an empty monitoreditemrequest, some don't.
		// beter to just return
		return nil, nil
	}

	toAdd := make([]*ua.MonitoredItemCreateRequest, 0)

	// Add handles and make requests
	for i, node := range nodes {
		handle := atomic.AddUint32(&s.monitor.nextClientHandle, 1)
		s.handles[handle] = nodes[i].NodeID
		nodes[i].handle = handle

		request := opcua.NewDefaultMonitoredItemCreateRequest(opcua.MonitoredItemCreateRequestArgs{
			NodeID:       node.NodeID,
			AttributeID:  ua.AttributeIDEventNotifier,
			ClientHandle: handle,
			Filter:       filter,
		})
		request.MonitoringMode = node.MonitoringMode

		if node.MonitoringParameters != nil {
			request.RequestedParameters = node.MonitoringParameters
			request.RequestedParameters.ClientHandle = handle
		}
		toAdd = append(toAdd, request)
	}
	resp, err := s.sub.Monitor(ctx, ua.TimestampsToReturnBoth, toAdd...)
	if err != nil {
		return nil, err
	}

	if resp.ResponseHeader.ServiceResult != ua.StatusOK {
		return nil, resp.ResponseHeader.ServiceResult
	}

	if len(resp.Results) != len(toAdd) {
		return nil, errors.Errorf("monitor items response length mismatch")
	}
	var monitoredItems []Item
	for i, res := range resp.Results {
		if res.StatusCode != ua.StatusOK {
			return nil, res.StatusCode
		}
		mn := Item{
			id:     res.MonitoredItemID,
			handle: nodes[i].handle,
			nodeID: toAdd[i].ItemToMonitor.NodeID,
		}
		s.itemLookup[res.MonitoredItemID] = mn
		monitoredItems = append(monitoredItems, mn)
	}

	return monitoredItems, nil
}

// RemoveNodes removes nodes defined by their string representation
func (s *Subscription) RemoveNodes(ctx context.Context, nodes ...string) error {
	nodeIDs, err := parseNodeSlice(nodes...)
	if err != nil {
		return err
	}
	return s.RemoveNodeIDs(ctx, nodeIDs...)
}

// RemoveNodeIDs removes nodes
func (s *Subscription) RemoveNodeIDs(ctx context.Context, nodes ...*ua.NodeID) error {
	if len(nodes) == 0 {
		return nil
	}

	var toRemove []Item
	for _, node := range nodes {
		for _, item := range s.itemLookup {
			if item.nodeID.String() == node.String() {
				toRemove = append(toRemove, item)
				break
			}
		}
	}

	return s.RemoveMonitorItems(ctx, toRemove...)
}

// RemoveMonitorItems removes nodes
func (s *Subscription) RemoveMonitorItems(ctx context.Context, items ...Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(items) == 0 {
		return nil
	}

	var toRemove []uint32
	for _, item := range items {
		_, ok := s.itemLookup[item.id]
		if !ok {
			return errors.Errorf("item not found: %s", item.id)
		}
		delete(s.itemLookup, item.id)
		delete(s.handles, item.handle)
		toRemove = append(toRemove, item.id)
	}

	resp, err := s.sub.Unmonitor(ctx, toRemove...)
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

// ModifyMonitorItems modifies nodes with monitoring parameters to the subscription
func (s *Subscription) ModifyMonitorItems(ctx context.Context, nodes ...Request) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(nodes) == 0 {
		return nil
	}

	toModify := make([]*ua.MonitoredItemModifyRequest, 0)

	for _, node := range nodes {
		for _, item := range s.itemLookup {
			if item.nodeID.String() != node.NodeID.String() {
				continue
			}

			if node.MonitoringParameters == nil {
				break
			}

			request := &ua.MonitoredItemModifyRequest{
				MonitoredItemID:     item.id,
				RequestedParameters: node.MonitoringParameters,
			}
			request.RequestedParameters.ClientHandle = item.handle
			toModify = append(toModify, request)
			break
		}
	}

	resp, err := s.sub.ModifyMonitoredItems(ctx, ua.TimestampsToReturnBoth, toModify...)
	if err != nil {
		return err
	}

	if resp.ResponseHeader.ServiceResult != ua.StatusOK {
		return resp.ResponseHeader.ServiceResult
	}

	if len(resp.Results) != len(toModify) {
		return errors.Errorf("modify monitored items response length mismatch")
	}

	for _, res := range resp.Results {
		if res.StatusCode != ua.StatusOK {
			return res.StatusCode
		}
	}

	return nil
}

// SetMonitoringModeForNodes sets the monitoring mode for nodes defined by their string representation
func (s *Subscription) SetMonitoringModeForNodes(ctx context.Context, monitoringMode ua.MonitoringMode, nodes ...string) error {
	nodeIDs, err := parseNodeSlice(nodes...)
	if err != nil {
		return err
	}

	return s.SetMonitoringModeForNodeIDs(ctx, monitoringMode, nodeIDs...)
}

// SetMonitoringModeForNodeIDs sets the monitoring mode for nodes
func (s *Subscription) SetMonitoringModeForNodeIDs(ctx context.Context, monitoringMode ua.MonitoringMode, nodes ...*ua.NodeID) error {
	if len(nodes) == 0 {
		return nil
	}

	var toSet []Item
	for _, node := range nodes {
		for _, item := range s.itemLookup {
			if item.nodeID.String() == node.String() {
				toSet = append(toSet, item)
				break
			}
		}
	}

	return s.SetMonitoringMode(ctx, monitoringMode, toSet...)
}

// SetMonitoringMode sets the monitoring mode for nodes
func (s *Subscription) SetMonitoringMode(ctx context.Context, monitoringMode ua.MonitoringMode, items ...Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(items) == 0 {
		return nil
	}

	toSet := make([]uint32, 0)
	for _, item := range items {
		toSet = append(toSet, item.id)
	}

	resp, err := s.sub.SetMonitoringMode(ctx, monitoringMode, toSet...)
	if err != nil {
		return err
	}

	if resp.ResponseHeader.ServiceResult != ua.StatusOK {
		return resp.ResponseHeader.ServiceResult
	}

	if len(resp.Results) != len(toSet) {
		return errors.Errorf("set monitoring mode response length mismatch")
	}

	for _, statusCode := range resp.Results {
		if statusCode != ua.StatusOK {
			return statusCode
		}
	}

	return nil
}

// Stats returns statistics for the subscription
func (s *Subscription) Stats(ctx context.Context) (*ua.SubscriptionDiagnosticsDataType, error) {
	return s.sub.Stats(ctx)
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
