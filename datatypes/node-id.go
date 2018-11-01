// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// NodeID type definitions.
//
// Specification: Part 6, 5.2.2.9
const (
	TypeTwoByte = iota
	TypeFourByte
	TypeNumeric
	TypeString
	TypeGUID
	TypeOpaque
)

// NodeID is an identifier for a node in the address space of an OPC UA Server.
// The NodeID object encodes all different node id types.
type NodeID struct {
	mask uint8
	ns   uint16
	nid  uint32
	bid  []byte
	gid  *GUID
}

// NewTwoByteNodeID returns a new two byte node id.
func NewTwoByteNodeID(id uint8) *NodeID {
	return &NodeID{
		mask: TypeTwoByte,
		nid:  uint32(id),
	}
}

// NewFourByteNodeID returns a new four byte node id.
func NewFourByteNodeID(ns uint8, id uint16) *NodeID {
	return &NodeID{
		mask: TypeFourByte,
		ns:   uint16(ns),
		nid:  uint32(id),
	}
}

// NewNumericNodeID returns a new numeric node id.
func NewNumericNodeID(ns uint16, id uint32) *NodeID {
	return &NodeID{
		mask: TypeNumeric,
		ns:   ns,
		nid:  id,
	}
}

// NewStringNodeID returns a new string node id.
func NewStringNodeID(ns uint16, id string) *NodeID {
	return &NodeID{
		mask: TypeString,
		ns:   ns,
		bid:  []byte(id),
	}
}

// NewGUIDNodeID returns a new GUID node id.
func NewGUIDNodeID(ns uint16, id string) *NodeID {
	return &NodeID{
		mask: TypeGUID,
		ns:   ns,
		gid:  NewGUID(id),
	}
}

// NewOpaqueNodeID returns a new opaque node id.
func NewOpaqueNodeID(ns uint16, id []byte) *NodeID {
	return &NodeID{
		mask: TypeOpaque,
		ns:   ns,
		bid:  id,
	}
}

// NewNodeID returns a node id from a string definition of the format
// 'ns=<namespace>;{s,i,b,g}=<identifier>'. Namespace URLs 'nsu=' are not
// supported since they require a lookup. For string node ids the 's='
// prefix can be omitted.
func NewNodeID(s string) (*NodeID, error) {
	if s == "" {
		return NewTwoByteNodeID(0), nil
	}

	p := strings.SplitN(s, ";", 2)
	if len(p) < 2 {
		return nil, fmt.Errorf("invalid node id: %s", s)
	}
	nsval, idval := p[0], p[1]

	// parse namespace
	var ns uint16
	switch {
	case strings.HasPrefix(nsval, "nsu="):
		return nil, fmt.Errorf("namespace urls are not supported: %s", s)

	case strings.HasPrefix(nsval, "ns="):
		n, err := strconv.Atoi(nsval[3:])
		if err != nil {
			return nil, fmt.Errorf("invalid namespace id: %s", s)
		}
		if n < 0 || n > math.MaxUint16 {
			return nil, fmt.Errorf("namespace id out of range (0..65535): %s", s)
		}
		ns = uint16(n)

	default:
		return nil, fmt.Errorf("invalid node id: %s", s)
	}

	// parse identifier
	switch {
	case strings.HasPrefix(idval, "i="):
		id, err := strconv.Atoi(idval[2:])
		if err != nil {
			return nil, fmt.Errorf("invalid numeric id: %s", s)
		}
		switch {
		case ns == 0 && id < 256:
			return NewTwoByteNodeID(byte(id)), nil
		case ns < 256 && id >= 0 && id < math.MaxUint16:
			return NewFourByteNodeID(byte(ns), uint16(id)), nil
		case id >= 0 && id < math.MaxUint32:
			return NewNumericNodeID(ns, uint32(id)), nil
		default:
			return nil, fmt.Errorf("numeric id out of range (0..2^32-1): %s", s)
		}

	case strings.HasPrefix(idval, "s="):
		return NewStringNodeID(ns, idval[2:]), nil

	case strings.HasPrefix(idval, "g="):
		n := NewGUIDNodeID(ns, idval[2:])
		if n == nil || n.StringID() == "" {
			return nil, fmt.Errorf("invalid guid node id: %s", s)
		}
		return n, nil

	case strings.HasPrefix(idval, "b="):
		b, err := base64.StdEncoding.DecodeString(idval[2:])
		if err != nil {
			return nil, fmt.Errorf("invalid opaque node id: %s", s)
		}
		return NewOpaqueNodeID(ns, b), nil

	default:
		return NewStringNodeID(ns, idval), nil
	}
}

// EncodingMask returns the encoding mask field including the
// type information and additional flags.
func (n *NodeID) EncodingMask() uint8 {
	return n.mask
}

// Type returns the node id type in EncodingMask.
func (n *NodeID) Type() uint8 {
	return n.mask & 0xf
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

// Serialize serializes NodeID to bytes.
func (n *NodeID) Serialize() ([]byte, error) {
	b := make([]byte, n.Len())
	if err := n.SerializeTo(b); err != nil {
		return nil, err
	}
	return b, nil
}

// SerializeTo serializes NodeID into bytes.
func (n *NodeID) SerializeTo(b []byte) error {
	switch n.Type() {
	case TypeTwoByte:
		b[0] = n.mask
		b[1] = uint8(n.nid)
		return nil

	case TypeFourByte:
		b[0] = n.mask
		b[1] = uint8(n.ns)
		binary.LittleEndian.PutUint16(b[2:], uint16(n.nid))
		return nil

	case TypeNumeric:
		b[0] = n.mask
		binary.LittleEndian.PutUint16(b[1:3], n.ns)
		binary.LittleEndian.PutUint32(b[3:7], n.nid)
		return nil

	case TypeGUID:
		b[0] = n.mask
		binary.LittleEndian.PutUint16(b[1:3], n.ns)
		return n.gid.SerializeTo(b[3:])

	case TypeString, TypeOpaque:
		b[0] = n.mask
		binary.LittleEndian.PutUint16(b[1:3], n.ns)
		binary.LittleEndian.PutUint32(b[3:7], uint32(len(n.bid)))
		copy(b[7:7+len(n.bid)], n.bid)
		return nil

	default:
		return fmt.Errorf("invalid node id type: %d", n.Type())
	}

	return nil
}

// DecodeFromBytes decodes a NodeID from bytes.
func (n *NodeID) DecodeFromBytes(b []byte) error {
	if len(b) == 0 {
		return io.ErrUnexpectedEOF
	}
	n.mask = b[0]

	switch n.Type() {
	case TypeTwoByte:
		if len(b) < 2 {
			return io.ErrUnexpectedEOF
		}
		n.nid = uint32(b[1])
		return nil

	case TypeFourByte:
		if len(b) < 4 {
			return io.ErrUnexpectedEOF
		}
		n.ns = uint16(b[1])
		n.nid = uint32(binary.LittleEndian.Uint16(b[2:4]))
		return nil

	case TypeNumeric:
		if len(b) < 7 {
			return io.ErrUnexpectedEOF
		}
		n.ns = binary.LittleEndian.Uint16(b[1:3])
		n.nid = binary.LittleEndian.Uint32(b[3:7])
		return nil

	case TypeGUID:
		if len(b) < 19 {
			return io.ErrUnexpectedEOF
		}
		n.ns = binary.LittleEndian.Uint16(b[1:3])
		n.gid = &GUID{}
		return n.gid.DecodeFromBytes(b[3:19])

	case TypeString, TypeOpaque:
		if len(b) < 7 {
			return io.ErrUnexpectedEOF
		}
		n.ns = binary.LittleEndian.Uint16(b[1:3])
		l := binary.LittleEndian.Uint32(b[3:7])
		if len(b) < int(l) {
			return io.ErrUnexpectedEOF
		}
		n.bid = make([]byte, l)
		copy(n.bid, b[7:7+l])
		return nil

	default:
		panic(fmt.Sprintf("invalid node id type: %d", n.Type()))
	}
}

// Len returns the length of a serialized NodeID in bytes.
func (n *NodeID) Len() int {
	switch n.Type() {
	case TypeTwoByte:
		return 2

	case TypeFourByte:
		return 4

	case TypeNumeric:
		return 7

	case TypeGUID:
		return 19

	case TypeString, TypeOpaque:
		return 7 + len(n.bid)

	default:
		panic(fmt.Sprintf("invalid node id type: %d", n.Type()))
	}
}

// Namespace returns the namespace id. For two byte node ids
// this will always be zero.
func (n *NodeID) Namespace() int {
	return int(n.ns)
}

// SetNamespace sets the namespace id. It returns an error
// if the id is not within the range of the node id type.
func (n *NodeID) SetNamespace(v int) error {
	switch n.Type() {
	case TypeTwoByte:
		if v != 0 {
			return fmt.Errorf("out of range [0..0]: %d", v)
		}
		return nil

	case TypeFourByte:
		if max := math.MaxUint8; v < 0 || v > max {
			return fmt.Errorf("out of range [0..%d]: %d", max, v)
		}
		n.ns = uint16(v)
		return nil

	default:
		if max := math.MaxUint16; v < 0 || v > max {
			return fmt.Errorf("out of range [0..%d]: %d", max, v)
		}
		n.ns = uint16(v)
		return nil
	}
}

// IntID returns the identifier value if the type is
// TwoByte, FourByte or Numeric. For all other types IntID
// returns 0.
func (n *NodeID) IntID() int {
	return int(n.nid)
}

// SetIntID sets the identifier value for two byte, four byte and
// numeric node ids. It returns an error for other types.
func (n *NodeID) SetIntID(v int) error {
	switch n.Type() {
	case TypeTwoByte:
		if max := math.MaxUint8; v < 0 || v > max {
			return fmt.Errorf("out of range [0..%d]: %d", max, v)
		}
		n.nid = uint32(v)
		return nil

	case TypeFourByte:
		if max := math.MaxUint16; v < 0 || v > max {
			return fmt.Errorf("out of range [0..%d]: %d", max, v)
		}
		n.nid = uint32(v)
		return nil

	case TypeNumeric:
		if max := math.MaxUint32; v < 0 || v > max {
			return fmt.Errorf("out of range [0..%d]: %d", max, v)
		}
		n.nid = uint32(v)
		return nil

	default:
		return fmt.Errorf("incompatible node id type")
	}
}

// StringID returns the string value of the identifier
// for String and GUID NodeIDs, and the base64 encoded
// value for Opaque types. For all other types StringID
// returns an empty string.
func (n *NodeID) StringID() string {
	switch n.Type() {
	case TypeGUID:
		if n.gid == nil {
			return ""
		}
		return n.gid.String()
	case TypeString:
		return string(n.bid)
	case TypeOpaque:
		return base64.StdEncoding.EncodeToString(n.bid)
	default:
		return ""
	}
}

// SetStringID sets the identifier value for string, guid and opaque
// node ids. It returns an error for other types.
func (n *NodeID) SetStringID(v string) error {
	switch n.Type() {
	case TypeGUID:
		n.gid = NewGUID(v)
		return nil

	case TypeString:
		n.bid = []byte(v)
		return nil

	case TypeOpaque:
		b, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return err
		}
		n.bid = b
		return nil

	default:
		return fmt.Errorf("incompatible node id type")
	}
}

// String returns the string representation of the NodeID
// in the format described by NewNodeID.
func (n *NodeID) String() string {
	switch n.Type() {
	case TypeTwoByte:
		return fmt.Sprintf("i=%d", n.nid)

	case TypeFourByte:
		if n.ns == 0 {
			return fmt.Sprintf("i=%d", n.nid)
		}
		return fmt.Sprintf("ns=%d;i=%d", n.ns, n.nid)

	case TypeNumeric:
		if n.ns == 0 {
			return fmt.Sprintf("i=%d", n.nid)
		}
		return fmt.Sprintf("ns=%d;i=%d", n.ns, n.nid)

	case TypeString:
		if n.ns == 0 {
			return fmt.Sprintf("s=%s", n.StringID())
		}
		return fmt.Sprintf("ns=%d;s=%s", n.ns, n.StringID())

	case TypeGUID:
		if n.ns == 0 {
			return fmt.Sprintf("g=%s", n.StringID())
		}
		return fmt.Sprintf("ns=%d;g=%s", n.ns, n.StringID())

	case TypeOpaque:
		if n.ns == 0 {
			return fmt.Sprintf("o=%s", n.StringID())
		}
		return fmt.Sprintf("ns=%d;o=%s", n.ns, n.StringID())

	default:
		panic(fmt.Sprintf("invalid node id type: %d", n.Type()))
	}
}

// DecodeNodeID decodes a node id from bytes.
func DecodeNodeID(b []byte) (*NodeID, error) {
	n := &NodeID{}
	if err := n.DecodeFromBytes(b); err != nil {
		return nil, err
	}
	return n, nil
}
