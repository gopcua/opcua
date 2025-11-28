package server

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server/attrs"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/ualog"
)

// the base "node-centric" namespace
type NodeNameSpace struct {
	srv             *Server
	name            string
	mu              sync.RWMutex
	nodes           []*Node
	m               map[string]*Node
	id              uint16
	nodeid_sequence uint32

	ExternalNotification chan *ua.NodeID

	logAttributes ualog.Attr
}

func (ns *NodeNameSpace) GetNextNodeID() uint32 {
	if ns.nodeid_sequence < 100 {
		ns.nodeid_sequence = 100
	}
	return atomic.AddUint32(&(ns.nodeid_sequence), 1)
}

func NewNodeNameSpace(srv *Server, name string) *NodeNameSpace {
	ns := &NodeNameSpace{
		srv:                  srv,
		name:                 name,
		nodes:                make([]*Node, 0),
		m:                    make(map[string]*Node),
		ExternalNotification: make(chan *ua.NodeID),
		logAttributes:        ualog.GroupAttrs("namespace", ualog.String("name", name), ualog.String("type", "node")),
	}
	srv.AddNamespace(ns)

	//objectsNode := NewFolderNode(ua.NewNumericNodeID(ns.id, id.ObjectsFolder), ns.name)
	oid := ua.NewNumericNodeID(ns.ID(), id.ObjectsFolder)
	//eoid := ua.NewNumericExpandedNodeID(ns.ID(), id.ObjectsFolder)
	typedef := ua.NewNumericExpandedNodeID(0, id.ObjectsFolder)
	//reftype := ua.NewTwoByteNodeID(uint8(id.HasComponent)) // folder
	objectsNode := NewNode(
		oid,
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDNodeClass:     DataValueFromValue(uint32(ua.NodeClassObject)),
			ua.AttributeIDBrowseName:    DataValueFromValue(attrs.BrowseName(ns.name)),
			ua.AttributeIDDisplayName:   DataValueFromValue(attrs.DisplayName(ns.name, ns.name)),
			ua.AttributeIDDescription:   DataValueFromValue(uint32(ua.NodeClassObject)),
			ua.AttributeIDDataType:      DataValueFromValue(typedef),
			ua.AttributeIDEventNotifier: DataValueFromValue(int16(0)),
		},
		[]*ua.ReferenceDescription{},
		nil,
	)

	ns.AddNode(objectsNode)

	return ns

}

// This function is to notify opc subscribers if a node was changed
// without using the SetAttribute method
func (s *NodeNameSpace) ChangeNotification(ctx context.Context, nodeid *ua.NodeID) {
	s.srv.ChangeNotification(ctx, nodeid)
}

func (ns *NodeNameSpace) Name() string {
	return ns.name
}

func NewNameSpace(name string) *NodeNameSpace {
	return &NodeNameSpace{name: name, m: map[string]*Node{}}
}

func (as *NodeNameSpace) AddNode(n *Node) *Node {
	as.mu.Lock()
	defer as.mu.Unlock()

	/*
		nn := &Node{
			id:   n.id,
			attr: maps.Clone(n.attr),
			refs: slices.Clone(n.refs),
			val:  n.val,
			ns:   as,
		}
	*/

	// todo(fs): this is wrong since this leaves the old node in the list.
	as.nodes = append(as.nodes, n)
	k := n.ID().String()

	as.m[k] = n
	return n
}

func (as *NodeNameSpace) AddNewVariableNode(name string, value any) *Node {
	n := NewVariableNode(ua.NewNumericNodeID(as.id, as.GetNextNodeID()), name, value)
	as.AddNode(n)
	return n
}
func (as *NodeNameSpace) AddNewVariableStringNode(name string, value any) *Node {
	n := NewVariableNode(ua.NewStringNodeID(as.id, name), name, value)
	as.AddNode(n)
	return n
}

func (as *NodeNameSpace) Attribute(ctx context.Context, id *ua.NodeID, attr ua.AttributeID) *ua.DataValue {
	ctx = ualog.WithAttrs(ctx, as.logAttributes)
	ualog.Debug(ctx, "read node attribute",
		ualog.Any(ualog.NodeIdKey, id), ualog.Any("attr", attr),
	)

	n := as.Node(id)
	if n == nil {
		return &ua.DataValue{
			EncodingMask:    ua.DataValueServerTimestamp | ua.DataValueStatusCode,
			ServerTimestamp: time.Now(),
			Status:          ua.StatusBadNodeIDUnknown,
		}
	}

	if !n.Access(ua.AccessLevelTypeCurrentRead) {
		return &ua.DataValue{
			EncodingMask:    ua.DataValueServerTimestamp | ua.DataValueStatusCode,
			ServerTimestamp: time.Now(),
			Status:          ua.StatusBadUserAccessDenied,
		}
	}

	var err error
	var a *AttrValue

	switch attr {
	case ua.AttributeIDNodeID:
		a = &AttrValue{Value: DataValueFromValue(id)}
	case ua.AttributeIDEventNotifier:
		// TODO: this is a hack to force the EventNotifier to false for everything.
		// If at some point someone or something needs to use this, this will have to go away and be
		// fixed properly.
		a = &AttrValue{Value: DataValueFromValue(byte(0))}
	case ua.AttributeIDNodeClass:
		a, err = n.Attribute(attr)
		if err != nil {
			return &ua.DataValue{
				EncodingMask:    ua.DataValueServerTimestamp | ua.DataValueStatusCode,
				ServerTimestamp: time.Now(),
				Status:          ua.StatusBadAttributeIDInvalid,
			}
		}
		// TODO: we need int32 instead of uint32 here.  this isn't the right place to fix it, but it is a bandaid
		x, ok := a.Value.Value.Value().(uint32)
		if ok {
			a.Value.Value = ua.MustVariant(int32(x))
		}
	default:
		a, err = n.Attribute(attr)
	}

	if err != nil {
		return &ua.DataValue{
			EncodingMask:    ua.DataValueServerTimestamp | ua.DataValueStatusCode,
			ServerTimestamp: time.Now(),
			Status:          ua.StatusBadAttributeIDInvalid,
		}
	}
	return a.Value
}

func (as *NodeNameSpace) Node(id *ua.NodeID) *Node {
	as.mu.RLock()
	defer as.mu.RUnlock()
	if id == nil {
		return nil
	}
	k := id.String()

	n := as.m[k]
	if n == nil {
		return nil
	}
	return n
}

func (as *NodeNameSpace) Objects() *Node {
	of := ua.NewNumericNodeID(as.id, id.ObjectsFolder)
	return as.Node(of)
}

func (as *NodeNameSpace) Root() *Node {
	return as.Node(RootFolder)
}

func (ns *NodeNameSpace) Browse(ctx context.Context, bd *ua.BrowseDescription) *ua.BrowseResult {
	ualog.Debug(ctx, "browse", ns.logAttributes,
		ualog.Any(ualog.NodeIdKey, bd.NodeID), ualog.Bitmask("mask", bd.ResultMask),
	)

	ns.mu.RLock()
	defer ns.mu.RUnlock()

	n := ns.Node(bd.NodeID)
	if n == nil {
		return &ua.BrowseResult{StatusCode: ua.StatusBadNodeIDUnknown}
	}

	refs := make([]*ua.ReferenceDescription, 0, len(n.refs))

	for i := range n.refs {
		r := n.refs[i]
		// we can't have nils in these or the encoder will fail.
		if r.NodeID == nil || r.BrowseName == nil || r.DisplayName == nil || r.TypeDefinition == nil {
			continue
		}

		// see if this is a ref the client was interested in.
		if !suitableRef(ctx, ns.srv, bd, r) {
			continue
		}

		td := ns.srv.Node(r.NodeID.NodeID)

		rf := &ua.ReferenceDescription{
			ReferenceTypeID: r.ReferenceTypeID,
			IsForward:       r.IsForward,
			NodeID:          r.NodeID,
			BrowseName:      r.BrowseName,
			DisplayName:     r.DisplayName,
			NodeClass:       r.NodeClass,
			TypeDefinition:  td.DataType(),
		}

		if rf.ReferenceTypeID.IntID() == id.HasTypeDefinition && rf.IsForward {
			// this one has to be first!
			refs = append([]*ua.ReferenceDescription{rf}, refs...)
		} else {
			refs = append(refs, rf)
		}
	}

	return &ua.BrowseResult{
		StatusCode: ua.StatusGood,
		References: refs,
	}
}

func (ns *NodeNameSpace) ID() uint16 {
	return ns.id
}

func (ns *NodeNameSpace) SetID(id uint16) {
	ns.id = id
}
func (as *NodeNameSpace) SetAttribute(ctx context.Context, id *ua.NodeID, attr ua.AttributeID, val *ua.DataValue) ua.StatusCode {
	ctx = ualog.WithAttrs(ctx, as.logAttributes)
	ualog.Debug(ctx, "write node attribute", ualog.Any(ualog.NodeIdKey, id), ualog.Any("attr", attr))

	n := as.Node(id)
	if n == nil {
		return ua.StatusBadNodeIDUnknown
	}

	if !n.Access(ua.AccessLevelTypeCurrentWrite) {
		return ua.StatusBadUserAccessDenied
	}

	err := n.SetAttribute(attr, val)
	if err != nil {
		return ua.StatusBadAttributeIDInvalid
	}
	as.srv.ChangeNotification(ctx, id)
	select {
	case as.ExternalNotification <- id:
	default:
	}

	return ua.StatusOK
}
