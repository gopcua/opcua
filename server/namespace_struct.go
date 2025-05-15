package server

import (
	"reflect"
	"sync"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server/attrs"
	"github.com/gopcua/opcua/ua"
)

// This namespaces give a convenient way to have data mapped to the OPC server
// without having to map your application data to the OCP-UA data abstraction
//
// # It (currently) supports a single level of field access so only atomic types are allowed in the struct
//
// To notify subscribers of changes, be sure to call ChangeNotification(FieldName) after changing a value.
// To be notified of changes from the opc-ua server to the struct, receive on ExternalNotification channel
type StructNamespace[T any] struct {
	srv  *Server
	name string
	mu   sync.RWMutex
	data T

	// This can be used to be alerted when a value is changed from the opc server
	ExternalNotification chan string

	id uint16
}

// This function is used to notify OPC UA subscribers if a key was changed without using the
// SetValue() function
func (s *StructNamespace[T]) ChangeNotification(key string) {
	s.srv.ChangeNotification(ua.NewStringNodeID(s.id, key))
}

// NewStructNamespace creates a new namespace for the given struct.
// Be sure to pass the struct in as a pointer to continue to have access to it.
func NewStructNamespace[T any](srv *Server, name string, str T) *StructNamespace[T] {
	mrw := StructNamespace[T]{
		srv:                  srv,
		name:                 name,
		data:                 str,
		ExternalNotification: make(chan string),
	}
	srv.AddNamespace(&mrw)
	return &mrw
}

func (s *StructNamespace[T]) ID() uint16 {
	return s.id
}
func (ns *StructNamespace[T]) SetID(id uint16) {
	ns.id = id
}

func (ns *StructNamespace[T]) Browse(bd *ua.BrowseDescription) *ua.BrowseResult {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	if ns.srv.cfg.logger != nil {
		ns.srv.cfg.logger.Debug("BrowseRequest: id=%s mask=%08b\n", bd.NodeID, bd.ResultMask)
		ns.srv.cfg.logger.Debug("Browse req for %s", bd.NodeID.String())
	}
	if bd.NodeID.IntID() != id.RootFolder && bd.NodeID.IntID() != id.ObjectsFolder {
		refs := make([]*ua.ReferenceDescription, 0)
		return &ua.BrowseResult{
			StatusCode: ua.StatusGood,
			References: refs,
		}
		//return &ua.BrowseResult{StatusCode: ua.StatusBadNodeIDUnknown}
	}

	if bd.NodeID.IntID() == id.RootFolder {

		refs := make([]*ua.ReferenceDescription, 1)
		newid := ua.NewNumericNodeID(ns.id, id.ObjectsFolder)
		expnewid := ua.NewNumericExpandedNodeID(ns.id, id.ObjectsFolder)
		refs[0] = &ua.ReferenceDescription{
			ReferenceTypeID: newid,
			NodeID:          expnewid,
			BrowseName:      &ua.QualifiedName{NamespaceIndex: ns.id, Name: "Objects"},
			DisplayName:     &ua.LocalizedText{EncodingMask: ua.LocalizedTextText, Text: "Objects"},
			TypeDefinition:  expnewid,
		}

		return &ua.BrowseResult{
			StatusCode: ua.StatusGood,
			References: refs,
		}

	}

	strRef := reflect.TypeOf(ns.data)
	refs := make([]*ua.ReferenceDescription, strRef.NumField())

	keyid := 0
	for k := 0; k < strRef.NumField(); k++ {

		key := strRef.Field(k).Name
		refid := ua.NewNumericNodeID(0, id.HasComponent)
		expnewid := ua.NewStringExpandedNodeID(ns.id, key)

		refs[keyid] = &ua.ReferenceDescription{
			ReferenceTypeID: refid,
			IsForward:       true,
			NodeID:          expnewid,
			BrowseName:      &ua.QualifiedName{NamespaceIndex: ns.ID(), Name: key},
			DisplayName:     &ua.LocalizedText{EncodingMask: ua.LocalizedTextText, Text: key},
			NodeClass:       ua.NodeClassVariable, // when support is added for nested maps, this will be NodeClassObject
			TypeDefinition:  expnewid,
		}
		keyid++
	}

	return &ua.BrowseResult{
		StatusCode: ua.StatusGood,
		References: refs,
	}

}

func (ns *StructNamespace[T]) Attribute(n *ua.NodeID, a ua.AttributeID) *ua.DataValue {
	if ns.srv.cfg.logger != nil {
		ns.srv.cfg.logger.Debug("read: node=%s attr=%s", n.String(), a)
	}

	if n.IntID() != 0 {
		// this is not one of our normal tags.
		if n.IntID() != id.ObjectsFolder {
			return &ua.DataValue{
				EncodingMask:    ua.DataValueServerTimestamp | ua.DataValueStatusCode,
				ServerTimestamp: time.Now(),
				Status:          ua.StatusBadNodeIDInvalid,
			}
		}

		attrval, err := ns.Objects().Attribute(a)
		if err != nil {
			return &ua.DataValue{
				EncodingMask:    ua.DataValueServerTimestamp | ua.DataValueStatusCode,
				ServerTimestamp: time.Now(),
				Status:          ua.StatusBadAttributeIDInvalid,
			}
		}

		return attrval.Value

	}

	dv := &ua.DataValue{
		EncodingMask:    ua.DataValueServerTimestamp | ua.DataValueStatusCode,
		ServerTimestamp: time.Now(),
		Status:          ua.StatusBad,
	}

	key := n.StringID()

	var err error
	if ns.srv.cfg.logger != nil {
		ns.srv.cfg.logger.Debug("Read req for %s", key)
		ns.srv.cfg.logger.Debug("'%s' Data at read: %v", ns.name, ns.data)
	}

	// because our data is native go types we don't have any of the ua "attributes" attached to it.
	// so depending on what attribute the client wants, we'll inspect the data and return the appropriate
	// thing
	switch a {

	case ua.AttributeIDNodeID:
		dv.Status = ua.StatusOK
		dv.EncodingMask |= ua.DataValueValue
		dv.Value = ua.MustVariant(n)

		// we are going to use the node id directly to look it up from our data map.
	case ua.AttributeIDValue:

		dv.Status = ua.StatusOK
		dv.EncodingMask |= ua.DataValueValue
		strRef := reflect.ValueOf(ns.data)
		vRef := strRef.FieldByName(key)
		if vRef.IsZero() {
			return &ua.DataValue{
				EncodingMask:    ua.DataValueServerTimestamp | ua.DataValueStatusCode,
				ServerTimestamp: time.Now(),
				Status:          ua.StatusBadNodeIDUnknown,
			}
		}
		v := vRef.Interface()
		switch tv := v.(type) {
		case string:
			dv.Value = ua.MustVariant(tv)
		case int:
			// we can't use an int because it is of unspecified length.  I'm going to use int64 so that we don't
			// have to worry about cutting data off. probably.
			dv.Value = ua.MustVariant(int64(tv))
		case int32:
			dv.Value = ua.MustVariant(tv)
		case float32:
			dv.Value = ua.MustVariant(tv)
		case float64:
			dv.Value = ua.MustVariant(tv)
		case bool:
			dv.Value = ua.MustVariant(tv)
		default:
			dv.Value = ua.MustVariant(tv)
		}
		// nothing in this namespace has an ID Description
	case ua.AttributeIDDescription:
		dv.Status = ua.StatusOK
		dv.EncodingMask |= ua.DataValueValue
		dv.Value = ua.MustVariant(&ua.LocalizedText{EncodingMask: ua.LocalizedTextText, Text: ""})

	case ua.AttributeIDBrowseName:
		dv.Status = ua.StatusOK
		dv.EncodingMask |= ua.DataValueValue
		dv.Value = ua.MustVariant(attrs.BrowseName(key))
	case ua.AttributeIDDisplayName:
		dv.Status = ua.StatusOK
		dv.EncodingMask |= ua.DataValueValue
		dv.Value = ua.MustVariant(attrs.DisplayName(key, key))
	case ua.AttributeIDAccessLevel:
		dv.Status = ua.StatusOK
		dv.EncodingMask |= ua.DataValueValue
		level := byte(ua.AccessLevelExTypeCurrentWrite | ua.AccessLevelExTypeCurrentRead)
		dv.Value = ua.MustVariant(level)

	case ua.AttributeIDNodeClass:
		dv.Status = ua.StatusOK
		dv.EncodingMask |= ua.DataValueValue
		dv.Value = ua.MustVariant(int32(ua.NodeClassVariable))
		// nothing in this namespace has event notifiers
	case ua.AttributeIDEventNotifier:
		dv.Status = ua.StatusOK
		dv.EncodingMask |= ua.DataValueValue
		dv.Value = ua.MustVariant(int16(0))

	// values are in section 5.1.2 of the standard.
	// https://reference.opcfoundation.org/Core/Part6/v104/docs/5.1.2
	case ua.AttributeIDDataType:
		dv.Status = ua.StatusOK
		dv.EncodingMask |= ua.DataValueValue

		strRef := reflect.ValueOf(ns.data)
		vRef := strRef.FieldByName(key)
		if vRef.IsZero() {
			return &ua.DataValue{
				EncodingMask:    ua.DataValueServerTimestamp | ua.DataValueStatusCode,
				ServerTimestamp: time.Now(),
				Status:          ua.StatusBadNodeIDUnknown,
			}
		}
		v := vRef.Interface()
		switch v.(type) {
		case string:
			dv.Value, err = ua.NewVariant(ua.NewNumericNodeID(0, 12))
			if err != nil {
				if ns.srv.cfg.logger != nil {
					ns.srv.cfg.logger.Warn("problem creating variant: %v", err)
				}
			}
		case int:
			// we can't use an int because it is of unspecified length.  I'm going to use int64 so that we don't
			// have to worry about cutting data off.
			dv.Value, err = ua.NewVariant(ua.NewNumericNodeID(0, 6))
			if err != nil {
				if ns.srv.cfg.logger != nil {
					ns.srv.cfg.logger.Warn("problem creating variant: %v", err)
				}
			}
		case int32:
			dv.Value, err = ua.NewVariant(ua.NewNumericNodeID(0, 6))
			if err != nil {
				if ns.srv.cfg.logger != nil {
					ns.srv.cfg.logger.Warn("problem creating variant: %v", err)
				}
			}
		case float32:
			dv.Value, err = ua.NewVariant(ua.NewNumericNodeID(0, 10))
			if err != nil {
				if ns.srv.cfg.logger != nil {
					ns.srv.cfg.logger.Warn("problem creating variant: %v", err)
				}
			}
		case float64:
			dv.Value, err = ua.NewVariant(ua.NewNumericNodeID(0, 11))
			if err != nil {
				if ns.srv.cfg.logger != nil {
					ns.srv.cfg.logger.Warn("problem creating variant: %v", err)
				}
			}
		case bool:
			dv.Value, err = ua.NewVariant(ua.NewNumericNodeID(0, 1))
			if err != nil {
				if ns.srv.cfg.logger != nil {
					ns.srv.cfg.logger.Warn("problem creating variant: %v", err)
				}
			}
		default:
			dv.Value, err = ua.NewVariant(ua.NewNumericNodeID(0, 24))
			if err != nil {
				if ns.srv.cfg.logger != nil {
					ns.srv.cfg.logger.Warn("problem creating variant: %v", err)
				}
			}
		}

		// when we support arrays this will have to change.
	case ua.AttributeIDValueRank:
		dv.Status = ua.StatusOK
		dv.EncodingMask |= ua.DataValueValue
		dv.Value = ua.MustVariant(int32(-1))

	// when we support arrays this will have to change.
	case ua.AttributeIDArrayDimensions:
		dv.Status = ua.StatusOK
		dv.EncodingMask |= ua.DataValueValue
		dv.Value = ua.MustVariant([]uint32{})
	default:
		return dv
	}

	if dv.Value == nil {
		if ns.srv.cfg.logger != nil {
			ns.srv.cfg.logger.Warn("bad dv value")
		}
	} else {
		if ns.srv.cfg.logger != nil {
			ns.srv.cfg.logger.Debug("Read '%s' = '%v' (%v)", key, dv.Value, dv.Value.Value())
		}
	}

	return dv
}

func (s *StructNamespace[T]) SetAttribute(node *ua.NodeID, attr ua.AttributeID, val *ua.DataValue) ua.StatusCode {

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("'%s' Data pre-write: %v", s.name, s.data)
	}

	key := node.StringID()

	// we would normally look up the node in our actual address space, but since that's dumb, we're just
	// going to use the node id directly to look it up from our data map.
	if attr == ua.AttributeIDValue {
		v := val.Value.Value()
		strRef := reflect.ValueOf(s.data)
		vRef := strRef.FieldByName(key)
		if vRef.IsZero() {
			return ua.StatusBadNodeIDUnknown
		}
		if !vRef.CanSet() {
			return ua.StatusBadUserAccessDenied
		}
		if vRef.Type() != reflect.TypeOf(v) {
			return ua.StatusBadTypeMismatch
		}
		vRef.Set(reflect.ValueOf(v))

	}

	// notify the opc ua server the value has changed.
	s.srv.ChangeNotification(node)
	// notify the non-opc application the value has changed.
	select {
	case s.ExternalNotification <- key:
	default:
	}

	return ua.StatusOK
}

func (ns *StructNamespace[T]) Name() string {
	return ns.name
}
func (ns *StructNamespace[T]) AddNode(n *Node) *Node {
	return n
}
func (ns *StructNamespace[T]) Node(id *ua.NodeID) *Node {
	return nil

}
func (ns *StructNamespace[T]) Objects() *Node {
	oid := ua.NewNumericNodeID(ns.ID(), id.ObjectsFolder)
	//eoid := ua.NewNumericExpandedNodeID(ns.ID(), id.ObjectsFolder)
	typedef := ua.NewNumericExpandedNodeID(0, id.ObjectsFolder)
	//reftype := ua.NewTwoByteNodeID(uint8(id.HasComponent)) // folder
	n := NewNode(
		oid,
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDNodeClass:     DataValueFromValue(int32(ua.NodeClassObject)),
			ua.AttributeIDBrowseName:    DataValueFromValue(attrs.BrowseName(ns.name)),
			ua.AttributeIDDisplayName:   DataValueFromValue(attrs.DisplayName(ns.name, ns.name)),
			ua.AttributeIDDescription:   DataValueFromValue(uint32(ua.NodeClassObject)),
			ua.AttributeIDDataType:      DataValueFromValue(typedef),
			ua.AttributeIDEventNotifier: DataValueFromValue(int16(0)),
		},
		[]*ua.ReferenceDescription{},
		nil,
	)
	return n

}
func (ns *StructNamespace[T]) Root() *Node {
	n := NewNode(
		ua.NewNumericNodeID(ns.ID(), id.RootFolder),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDNodeClass:   DataValueFromValue(int32(ua.NodeClassObject)),
			ua.AttributeIDBrowseName:  DataValueFromValue(attrs.BrowseName("Root")),
			ua.AttributeIDDisplayName: DataValueFromValue(attrs.DisplayName("Root", "")),
		},
		[]*ua.ReferenceDescription{},
		nil,
	)
	return n

}
