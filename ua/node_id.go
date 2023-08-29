// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"

	"github.com/gopcua/opcua/errors"
)

// todo(fs): fix mask

// NodeID is an identifier for a node in the address space of an OPC UA Server.
// The NodeID object encodes all different node id types.
type NodeID struct {
	mask NodeIDType
	ns   uint16
	nid  uint32
	bid  []byte
	gid  *GUID
}

// NewTwoByteNodeID returns a new two byte node id.
func NewTwoByteNodeID(id uint8) *NodeID {
	return &NodeID{
		mask: NodeIDTypeTwoByte,
		nid:  uint32(id),
	}
}

// NewFourByteNodeID returns a new four byte node id.
func NewFourByteNodeID(ns uint8, id uint16) *NodeID {
	return &NodeID{
		mask: NodeIDTypeFourByte,
		ns:   uint16(ns),
		nid:  uint32(id),
	}
}

// NewNumericNodeID returns a new numeric node id.
func NewNumericNodeID(ns uint16, id uint32) *NodeID {
	return &NodeID{
		mask: NodeIDTypeNumeric,
		ns:   ns,
		nid:  id,
	}
}

// NewStringNodeID returns a new string node id.
func NewStringNodeID(ns uint16, id string) *NodeID {
	return &NodeID{
		mask: NodeIDTypeString,
		ns:   ns,
		bid:  []byte(id),
	}
}

// NewGUIDNodeID returns a new GUID node id.
func NewGUIDNodeID(ns uint16, id string) *NodeID {
	return &NodeID{
		mask: NodeIDTypeGUID,
		ns:   ns,
		gid:  NewGUID(id),
	}
}

// NewByteStringNodeID returns a new byte string node id.
func NewByteStringNodeID(ns uint16, id []byte) *NodeID {
	return &NodeID{
		mask: NodeIDTypeByteString,
		ns:   ns,
		bid:  id,
	}
}

// NewNodeIDFromExpandedNodeID returns a new NodeID derived from ExpandedNodeID
func NewNodeIDFromExpandedNodeID(id *ExpandedNodeID) *NodeID {
	bid := make([]byte, len(id.NodeID.bid))
	copy(bid, id.NodeID.bid)
	if len(bid) == 0 {
		bid = nil
	}
	var gid *GUID
	if egid := id.NodeID.gid; egid != nil {
		var ngid GUID
		ngid.Data1 = egid.Data1
		ngid.Data2 = egid.Data2
		ngid.Data3 = egid.Data3
		ngid.Data4 = make([]byte, len(egid.Data4))
		copy(ngid.Data4, egid.Data4)
		if len(ngid.Data4) == 0 {
			ngid.Data4 = nil
		}
		gid = &ngid
	}
	return &NodeID{
		mask: id.NodeID.mask & 0b00111111, // expanded flag NamespaceURI and ServerIndex need to be unset
		ns:   id.NodeID.ns,
		nid:  id.NodeID.nid,
		bid:  bid,
		gid:  gid,
	}
}

// MustParseNodeID returns a node id from a string definition
// if it is parseable by ParseNodeID. Otherwise, the function
// panics.
func MustParseNodeID(s string) *NodeID {
	id, err := ParseNodeID(s)
	if err != nil {
		panic(err.Error())
	}
	return id
}

// ParseNodeID returns a node id from a string definition of the format
// 'ns=<namespace>;{s,i,b,g}=<identifier>'.
//
// The 's=' prefix can be omitted for string node ids in namespace 0.
//
// For numeric ids the smallest possible type which can store the namespace
// and id value is returned.
//
// Namespace URIs are not supported since NodeID cannot store them. If you need
// to support namespace URIs use ParseExpandedNodeID.
func ParseNodeID(s string) (*NodeID, error) {
	id, err := ParseExpandedNodeID(s, nil)
	if err != nil {
		return nil, err
	}
	if id.HasNamespaceURI() {
		return nil, errors.Errorf("namespace uris are not supported. use `ua.ParseExpandedNodeID`")
	}
	if id.HasServerIndex() {
		return nil, errors.Errorf("server index is not supported. use `ua.ParseExpandedNodeID`")
	}
	return id.NodeID, nil
}

// EncodingMask returns the encoding mask field including the
// type information and additional flags.
func (n *NodeID) EncodingMask() NodeIDType {
	return n.mask
}

// Type returns the node id type in EncodingMask.
func (n *NodeID) Type() NodeIDType {
	return n.mask & NodeIDType(0xf)
}

// URIFlag returns whether the URI flag is set in EncodingMask.
func (n *NodeID) URIFlag() bool {
	return n.mask&0x80 == 0x80
}

// SetURIFlag sets NamespaceURI flag in EncodingMask.
func (n *NodeID) SetURIFlag() {
	n.mask |= 0x80
}

// IndexFlag returns whether the Index flag is set in EncodingMask.
func (n *NodeID) IndexFlag() bool {
	return n.mask&0x40 == 0x40
}

// SetIndexFlag sets NamespaceURI flag in EncodingMask.
func (n *NodeID) SetIndexFlag() {
	n.mask |= 0x40
}

// Namespace returns the namespace id. For two byte node ids
// this will always be zero.
func (n *NodeID) Namespace() uint16 {
	return n.ns
}

// SetNamespace sets the namespace id. It returns an error
// if the id is not within the range of the node id type.
func (n *NodeID) SetNamespace(v uint16) error {
	switch n.Type() {
	case NodeIDTypeTwoByte:
		if v != 0 {
			return errors.Errorf("out of range [0..0]: %d", v)
		}
		return nil

	case NodeIDTypeFourByte:
		if max := uint16(math.MaxUint8); v > max {
			return errors.Errorf("out of range [0..%d]: %d", max, v)
		}
		n.ns = uint16(v)
		return nil

	default:
		if max := uint16(math.MaxUint16); v > max {
			return errors.Errorf("out of range [0..%d]: %d", max, v)
		}
		n.ns = uint16(v)
		return nil
	}
}

// IntID returns the identifier value if the type is
// TwoByte, FourByte or Numeric. For all other types IntID
// returns 0.
func (n *NodeID) IntID() uint32 {
	return n.nid
}

// SetIntID sets the identifier value for two byte, four byte and
// numeric node ids. It returns an error for other types.
func (n *NodeID) SetIntID(v uint32) error {
	switch n.Type() {
	case NodeIDTypeTwoByte:
		if max := uint32(math.MaxUint8); v > max {
			return errors.Errorf("out of range [0..%d]: %d", max, v)
		}
		n.nid = uint32(v)
		return nil

	case NodeIDTypeFourByte:
		if max := uint32(math.MaxUint16); v > max {
			return errors.Errorf("out of range [0..%d]: %d", max, v)
		}
		n.nid = uint32(v)
		return nil

	case NodeIDTypeNumeric:
		if max := uint32(math.MaxUint32); v > max {
			return errors.Errorf("out of range [0..%d]: %d", max, v)
		}
		n.nid = uint32(v)
		return nil

	default:
		return errors.Errorf("incompatible node id type")
	}
}

// StringID returns the string value of the identifier
// for String and GUID NodeIDs, and the base64 encoded
// value for Opaque types. For all other types StringID
// returns an empty string.
func (n *NodeID) StringID() string {
	switch n.Type() {
	case NodeIDTypeGUID:
		if n.gid == nil {
			return ""
		}
		return n.gid.String()
	case NodeIDTypeString:
		return string(n.bid)
	case NodeIDTypeByteString:
		return base64.StdEncoding.EncodeToString(n.bid)
	default:
		return ""
	}
}

// SetStringID sets the identifier value for string, guid and opaque
// node ids. It returns an error for other types.
func (n *NodeID) SetStringID(v string) error {
	switch n.Type() {
	case NodeIDTypeGUID:
		n.gid = NewGUID(v)
		return nil

	case NodeIDTypeString:
		n.bid = []byte(v)
		return nil

	case NodeIDTypeByteString:
		b, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return err
		}
		n.bid = b
		return nil

	default:
		return errors.Errorf("incompatible node id type")
	}
}

// String returns the string representation of the NodeID
// in the format described by ParseNodeID.
func (n *NodeID) String() string {
	switch n.Type() {
	case NodeIDTypeTwoByte:
		return fmt.Sprintf("i=%d", n.nid)

	case NodeIDTypeFourByte:
		if n.ns == 0 {
			return fmt.Sprintf("i=%d", n.nid)
		}
		return fmt.Sprintf("ns=%d;i=%d", n.ns, n.nid)

	case NodeIDTypeNumeric:
		if n.ns == 0 {
			return fmt.Sprintf("i=%d", n.nid)
		}
		return fmt.Sprintf("ns=%d;i=%d", n.ns, n.nid)

	case NodeIDTypeString:
		if n.ns == 0 {
			return fmt.Sprintf("s=%s", n.StringID())
		}
		return fmt.Sprintf("ns=%d;s=%s", n.ns, n.StringID())

	case NodeIDTypeGUID:
		if n.ns == 0 {
			return fmt.Sprintf("g=%s", n.StringID())
		}
		return fmt.Sprintf("ns=%d;g=%s", n.ns, n.StringID())

	case NodeIDTypeByteString:
		if n.ns == 0 {
			return fmt.Sprintf("b=%s", n.StringID())
		}
		return fmt.Sprintf("ns=%d;b=%s", n.ns, n.StringID())

	default:
		panic(fmt.Sprintf("invalid node id type: %d", n.Type()))
	}
}

func (n *NodeID) Decode(b []byte) (int, error) {
	buf := NewBuffer(b)

	n.mask = NodeIDType(buf.ReadByte())
	typ := n.mask & 0xf

	switch typ {
	case NodeIDTypeTwoByte:
		n.nid = uint32(buf.ReadByte())
		return buf.Pos(), buf.Error()

	case NodeIDTypeFourByte:
		n.ns = uint16(buf.ReadByte())
		n.nid = uint32(buf.ReadUint16())
		return buf.Pos(), buf.Error()

	case NodeIDTypeNumeric:
		n.ns = buf.ReadUint16()
		n.nid = buf.ReadUint32()
		return buf.Pos(), buf.Error()

	case NodeIDTypeGUID:
		n.ns = buf.ReadUint16()
		n.gid = &GUID{}
		buf.ReadStruct(n.gid)
		return buf.Pos(), buf.Error()

	case NodeIDTypeByteString, NodeIDTypeString:
		n.ns = buf.ReadUint16()
		n.bid = buf.ReadBytes()
		return buf.Pos(), buf.Error()

	default:
		return 0, errors.Errorf("invalid node id type %v", typ)
	}
}

func (n *NodeID) Encode() ([]byte, error) {
	buf := NewBuffer(nil)
	buf.WriteByte(byte(n.mask))

	switch n.Type() {
	case NodeIDTypeTwoByte:
		buf.WriteByte(byte(n.nid))
	case NodeIDTypeFourByte:
		buf.WriteByte(byte(n.ns))
		buf.WriteUint16(uint16(n.nid))
	case NodeIDTypeNumeric:
		buf.WriteUint16(n.ns)
		buf.WriteUint32(n.nid)
	case NodeIDTypeGUID:
		buf.WriteUint16(n.ns)
		buf.WriteStruct(n.gid)
	case NodeIDTypeByteString, NodeIDTypeString:
		buf.WriteUint16(n.ns)
		buf.WriteByteString(n.bid)
	default:
		return nil, errors.Errorf("invalid node id type %v", n.Type())
	}
	return buf.Bytes(), buf.Error()
}

func (n *NodeID) MarshalJSON() ([]byte, error) {
	if n == nil {
		return []byte(`null`), nil
	}
	return json.Marshal(n.String())
}

func (n *NodeID) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	nid, err := ParseNodeID(s)
	if err != nil {
		return err
	}
	*n = *nid
	return nil
}
