package server

import (
	"log"
	"maps"
	"slices"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server/attrs"
	"github.com/gopcua/opcua/server/refs"
	"github.com/gopcua/opcua/ua"
)

type Attributes map[ua.AttributeID]*ua.DataValue

type References []*ua.ReferenceDescription

type ValueFunc func() *ua.DataValue

type AttrValue struct {
	Value           *ua.DataValue
	SourceTimestamp time.Time
}

func NewAttrValue(v *ua.DataValue) *AttrValue {
	return &AttrValue{Value: v}

}

func DataValueFromValue(val any) *ua.DataValue {
	// if we already have a data value, just return it.
	switch v := val.(type) {
	case *ua.DataValue:
		return v
	case ua.DataValue:
		return &v
	case ua.Variant:
		return &ua.DataValue{
			EncodingMask:    ua.DataValueValue,
			Value:           &v,
			SourceTimestamp: time.Now(),
		}
	case *ua.Variant:
		return &ua.DataValue{
			EncodingMask:    ua.DataValueValue,
			Value:           v,
			SourceTimestamp: time.Now(),
		}
	case int:
		return &ua.DataValue{
			EncodingMask:    ua.DataValueValue,
			Value:           ua.MustVariant(int32(v)),
			SourceTimestamp: time.Now(),
		}

	}

	v := ua.MustVariant(val)
	return &ua.DataValue{
		EncodingMask:    ua.DataValueValue,
		Value:           v,
		SourceTimestamp: time.Now(),
	}
}

type Node struct {
	id   *ua.NodeID
	attr Attributes
	refs References
	val  ValueFunc

	ns NameSpace
}

func NewNode(id *ua.NodeID, attr Attributes, refs References, val ValueFunc) *Node {
	n := &Node{id, attr, refs, val, nil}
	n.sanitize()
	return n
}

func NewFolderNode(nodeID *ua.NodeID, name string) *Node {
	reftype := ua.NewTwoByteNodeID(uint8(id.HasComponent)) // folder
	eoid := ua.NewNumericExpandedNodeID(nodeID.Namespace(), id.ObjectsFolder)
	typedef := ua.NewNumericExpandedNodeID(0, id.ObjectsFolder)
	n := NewNode(
		nodeID,
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDNodeClass:     DataValueFromValue(uint32(ua.NodeClassObject)),
			ua.AttributeIDBrowseName:    DataValueFromValue(attrs.BrowseName(name)),
			ua.AttributeIDDisplayName:   DataValueFromValue(attrs.DisplayName(name, name)),
			ua.AttributeIDDescription:   DataValueFromValue(uint32(ua.NodeClassObject)),
			ua.AttributeIDEventNotifier: DataValueFromValue(int16(0)),
		},
		[]*ua.ReferenceDescription{{
			ReferenceTypeID: reftype,
			IsForward:       true,
			NodeID:          eoid,
			BrowseName:      &ua.QualifiedName{NamespaceIndex: nodeID.Namespace(), Name: name},
			DisplayName:     &ua.LocalizedText{EncodingMask: ua.LocalizedTextText, Text: name},
			NodeClass:       ua.NodeClassObject,
			TypeDefinition:  typedef,
		}},
		nil,
	)
	return n
}

func NewVariableNode(nodeID *ua.NodeID, name string, value any) *Node {
	//eoid := ua.NewNumericExpandedNodeID(nodeID.Namespace(), nodeID.IntID())
	vf, ok := value.(func() *ua.DataValue)
	if !ok {
		typedef := ua.NewNumericExpandedNodeID(0, id.VariableNode)
		n := NewNode(
			nodeID,
			map[ua.AttributeID]*ua.DataValue{
				ua.AttributeIDNodeClass:     DataValueFromValue(uint32(ua.NodeClassVariable)),
				ua.AttributeIDBrowseName:    DataValueFromValue(attrs.BrowseName(name)),
				ua.AttributeIDDisplayName:   DataValueFromValue(attrs.DisplayName(name, name)),
				ua.AttributeIDDescription:   DataValueFromValue(uint32(ua.NodeClassVariable)),
				ua.AttributeIDDataType:      DataValueFromValue(typedef),
				ua.AttributeIDEventNotifier: DataValueFromValue(int16(0)),
			},
			[]*ua.ReferenceDescription{},
			func() *ua.DataValue {
				return DataValueFromValue(value)
			},
		)
		dvFunc, ok := value.(ValueFunc)
		if ok {
			n.val = dvFunc
		}
		return n
	}
	typedef := ua.NewNumericExpandedNodeID(0, id.VariableNode)
	n := NewNode(
		nodeID,
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDNodeClass:     DataValueFromValue(uint32(ua.NodeClassVariable)),
			ua.AttributeIDBrowseName:    DataValueFromValue(attrs.BrowseName(name)),
			ua.AttributeIDDisplayName:   DataValueFromValue(attrs.DisplayName(name, name)),
			ua.AttributeIDDescription:   DataValueFromValue(uint32(ua.NodeClassVariable)),
			ua.AttributeIDDataType:      DataValueFromValue(typedef),
			ua.AttributeIDEventNotifier: DataValueFromValue(int16(0)),
		},
		[]*ua.ReferenceDescription{},
		vf,
	)
	return n
}

func (n *Node) sanitize() {
	if n.attr == nil {
		n.attr = Attributes{}
	}
	if n.attr[ua.AttributeIDBrowseName] == nil {
		n.SetBrowseName("")
	}
	if n.attr[ua.AttributeIDDisplayName] == nil {
		n.SetDisplayName("", "")
	}
	if n.DisplayName().Text == "" {
		n.SetDisplayName(n.BrowseName().Name, "")
	}
	if n.attr[ua.AttributeIDDescription] == nil {
		n.SetDescription("", "")
	}
	//if n.attr[ua.AttributeIDDataType] == nil {
	//n.attr[ua.AttributeIDDataType] = ua.MustVariant(ua.NewTwoByteExpandedNodeID(0))
	//}
}

func (n *Node) ID() *ua.NodeID {
	return n.id
}

func (n *Node) Value() *ua.DataValue {
	if n.val == nil {
		return nil
	}
	return n.val()
}

func (n *Node) Attribute(id ua.AttributeID) (*AttrValue, error) {
	switch {
	case id == ua.AttributeIDValue:
		if n.val != nil {
			val := n.val()
			if val == nil {
				return nil, ua.StatusBadAttributeIDInvalid
			}
			return NewAttrValue(val), nil
		}
		return nil, ua.StatusBadAttributeIDInvalid
	case n.attr == nil:
		return nil, ua.StatusBadAttributeIDInvalid
	default:
		if v := n.attr[id]; v != nil {
			return NewAttrValue(v), nil
		}
		return nil, ua.StatusBadAttributeIDInvalid
	}
}
func (n *Node) SetAttribute(id ua.AttributeID, val *ua.DataValue) error {
	switch {
	case id == ua.AttributeIDValue:

		// TODO: probably need to do some type checking here.
		// And some permissions tests
		n.val = func() *ua.DataValue {
			return val
		}

		return nil
	default:
		n.attr[id] = val
	}
	return ua.StatusBadNodeAttributesInvalid
}

func (n *Node) BrowseName() *ua.QualifiedName {
	v := n.attr[ua.AttributeIDBrowseName]
	if v == nil || v.Value.Value() == nil {
		return &ua.QualifiedName{}
	}
	return v.Value.Value().(*ua.QualifiedName)
}

func (n *Node) SetBrowseName(s string) {
	n.attr[ua.AttributeIDBrowseName] = DataValueFromValue(&ua.QualifiedName{Name: s})
}

func (n *Node) DisplayName() *ua.LocalizedText {
	v := n.attr[ua.AttributeIDDisplayName]
	if v == nil || v.Value.Value() == nil {
		return &ua.LocalizedText{}
	}
	val := v.Value.Value().(*ua.LocalizedText)
	val.UpdateMask()
	return val
}

func (n *Node) SetDisplayName(text, locale string) {
	lt := &ua.LocalizedText{Text: text, Locale: locale}
	lt.UpdateMask()
	n.attr[ua.AttributeIDDisplayName] = DataValueFromValue(lt)
}

func (n *Node) Description() *ua.LocalizedText {
	v := n.attr[ua.AttributeIDDescription]
	if v == nil || v.Value.Value() == nil {
		return &ua.LocalizedText{}
	}
	return v.Value.Value().(*ua.LocalizedText)
}

func (n *Node) SetDescription(text, locale string) {
	n.attr[ua.AttributeIDDescription] = DataValueFromValue(&ua.LocalizedText{Text: text, Locale: locale})
}

func (n *Node) DataType() *ua.ExpandedNodeID {
	if n == nil {
		log.Printf("n was nil!")
		return ua.NewTwoByteExpandedNodeID(0)
	}
	v := n.attr[ua.AttributeIDDataType]
	if v == nil || v.Value.Value() == nil {
		// if we have a type definition, return that?
		for i := range n.refs {
			r := n.refs[i]
			if r.ReferenceTypeID == nil {
				log.Printf("reftypeid was nil!")
			}
			if r.ReferenceTypeID.IntID() == id.HasTypeDefinition && r.IsForward {
				return r.NodeID
			}
		}
		return ua.NewTwoByteExpandedNodeID(0)
	}
	return v.Value.Value().(*ua.ExpandedNodeID)
}

func (n *Node) SetNodeClass(nc ua.NodeClass) {
	n.attr[ua.AttributeIDNodeClass] = DataValueFromValue(uint32(nc))
}

func (n *Node) NodeClass() ua.NodeClass {
	v := n.attr[ua.AttributeIDNodeClass]
	if v == nil || v.Value.Value() == nil {
		return ua.NodeClassObject
	}
	vi32, ok := v.Value.Value().(int32)
	if !ok {
		vui32, ok := v.Value.Value().(uint32)
		if !ok {
			return ua.NodeClassObject
		}
		return ua.NodeClass(int32(vui32))
	}
	return ua.NodeClass(vi32)

}

func (n *Node) AddObject(o *Node) *Node {
	nn := &Node{
		id:   o.id,
		attr: maps.Clone(o.attr),
		refs: slices.Clone(o.refs),
	}
	if n.attr == nil {
		n.attr = Attributes{}
	}
	nn.SetNodeClass(ua.NodeClassObject)
	n.refs = append(n.refs, refs.Organizes(nn.id, nn.BrowseName().Name, nn.DisplayName().Text, nn.DataType()))
	return n.ns.AddNode(nn)
}

func (n *Node) AddVariable(o *Node) *Node {
	nn := &Node{
		id:   o.id,
		attr: maps.Clone(o.attr),
		refs: slices.Clone(o.refs),
		val:  o.val,
	}
	if n.attr == nil {
		n.attr = Attributes{}
	}
	nn.SetNodeClass(ua.NodeClassVariable)
	n.refs = append(n.refs, refs.Organizes(nn.id, nn.BrowseName().Name, nn.DisplayName().Text, nn.DataType()))
	return nn
}

type RefType int

const (
	RefTypeIDHasComponent = id.HasComponent
	RefTypeIDOrganizes    = id.Organizes
)

func (n *Node) AddRef(o *Node, rt RefType, forward bool) {
	//eoid := ua.NewNumericExpandedNodeID(o.ns.ID(), o.)
	eoid := ua.NewExpandedNodeID(o.ID(), "", 0)

	ref := ua.ReferenceDescription{
		ReferenceTypeID: ua.NewNumericNodeID(0, uint32(rt)), //o.refs[0].ReferenceTypeID,
		IsForward:       forward,
		NodeID:          eoid,
		BrowseName:      o.BrowseName(),
		DisplayName:     o.DisplayName(),
		NodeClass:       o.NodeClass(),
		TypeDefinition:  o.DataType(),
	}
	n.refs = append(n.refs, &ref)
}

// Access returns true if the node has the access level requested.
// It checks both the UserAccessLevel and AccessLevel attributes.
// If neither are present, it assumes global access and returns true.
//
// I'm not sure what the best way to implement "user" specific access levels
// is presently.  Will need functioning user authentication first, and then a way to
// pass it into the nodes user access attribute so it can be checked properly.
func (n Node) Access(flag ua.AccessLevelType) bool {

	access, err := n.Attribute(ua.AttributeIDUserAccessLevel)
	if err == nil { // if we have a user access level, we need to check it.
		val0 := access.Value.Value.Value()
		val, ok := val0.(uint8)
		if !ok {
			return false
		}
		if val&uint8(flag) == 0 {
			return false
		}
	}
	access, err = n.Attribute(ua.AttributeIDAccessLevel)
	if err == nil { // if we have an access level, we need to check it.
		val0 := access.Value.Value.Value()
		val, ok := val0.(uint8)
		if !ok {
			return false
		}

		if val&uint8(flag) == 0 {
			return false
		}
	}
	return true

}
