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

// NodeId type definitions.
//
// Specification: Part 6, 5.2.2.9
const (
	TypeTwoByte uint8 = iota
	TypeFourByte
	TypeNumeric
	TypeString
	TypeGUID
	TypeOpaque
)

type NodeID struct {
	mask uint8
	ns   uint16
	nid  uint32
	bid  []byte
	gid  *GUID
}

func NewTwoByteNodeID(id uint8) *NodeID {
	return &NodeID{
		mask: TypeTwoByte,
		nid:  uint32(id),
	}
}

func NewFourByteNodeID(ns uint8, id uint16) *NodeID {
	return &NodeID{
		mask: TypeFourByte,
		ns:   uint16(ns),
		nid:  uint32(id),
	}
}

func NewNumericNodeID(ns uint16, id uint32) *NodeID {
	return &NodeID{
		mask: TypeNumeric,
		ns:   ns,
		nid:  id,
	}
}

func NewStringNodeID(ns uint16, id string) *NodeID {
	return &NodeID{
		mask: TypeString,
		ns:   ns,
		bid:  []byte(id),
	}
}

func NewGUIDNodeID(ns uint16, id string) *NodeID {
	return &NodeID{
		mask: TypeGUID,
		ns:   ns,
		gid:  NewGUID(id),
	}
}

func NewOpaqueNodeID(ns uint16, id []byte) *NodeID {
	return &NodeID{
		mask: TypeOpaque,
		ns:   ns,
		bid:  id,
	}
}

func (n *NodeID) EncodingMask() uint8 {
	return n.mask
}

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

// GetIdentifier returns value in Identifier field in bytes.
func (n *NodeID) GetIdentifier() []byte {
	switch n.Type() {
	case TypeTwoByte:
		return []byte{uint8(n.nid)}

	case TypeFourByte:
		b := make([]byte, 2)
		binary.LittleEndian.PutUint16(b, uint16(n.nid))
		return b

	case TypeNumeric:
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, n.nid)
		return b

	case TypeGUID:
		b, _ := n.gid.Serialize()
		return b

	case TypeString, TypeOpaque:
		return n.bid

	default:
		panic(fmt.Sprintf("invalid node id type: %d", n.Type()))
	}
}

// Serialize serializes NodeID into bytes.
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

// GetIdentifier returns value in Identifier field in bytes.
func (n *NodeID) Len() int {
	switch n.Type() {
	case TypeTwoByte:
		return 2

	case TypeFourByte:
		return 4

	case TypeNumeric:
		return 7

	case TypeString:
		return 7 + len(n.bid)

	case TypeGUID:
		return 19

	case TypeOpaque:
		return 7 + len(n.bid)

	default:
		panic(fmt.Sprintf("invalid node id type: %d", n.Type()))
	}
}

func (n *NodeID) IntID() int {
	return int(n.nid)
}

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

// ParseNodeID returns a node id from a string definition of the format
// 'ns=<namespace>;{s,i,b,g}=<identifier>'. Namespace URLs 'nsu=' are not
// supported since they require a lookup. The 's=' prefix can be omitted
// for string node ids.
func ParseNodeID(s string) (*NodeID, error) {
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

func DecodeNodeID(b []byte) (*NodeID, error) {
	n := &NodeID{}
	if err := n.DecodeFromBytes(b); err != nil {
		return nil, err
	}
	return n, nil
}
