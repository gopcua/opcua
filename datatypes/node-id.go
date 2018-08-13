// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/binary"

	"github.com/wmnsk/gopcua/errors"
)

// NodeId type definitions.
const (
	TypeTwoByte uint8 = iota
	TypeFourByte
	TypeNumeric
	TypeString
	TypeGUID
	TypeOpaque
)

// NodeID is an interface to handle all types of NodeId.
type NodeID interface {
	// Serialize() ([]byte, error)
	// SerializeTo([]byte) error
	DecodeFromBytes([]byte) error
	Len() int
	// String() string
	EncodingMaskValue() uint8
}

// DecodeNodeID decodes given bytes into NodeID, depending on the Encoding Mask.
func DecodeNodeID(b []byte) (NodeID, error) {
	var n NodeID

	encodingMask := b[0] & 0xf
	switch encodingMask {
	case TypeTwoByte:
		n = &TwoByteNodeID{}
	case TypeFourByte:
		n = &FourByteNodeID{}
	case TypeNumeric:
		n = &NumericNodeID{}
	case TypeString:
		n = &StringNodeID{}
	case TypeGUID:
		n = &GUIDNodeID{}
	case TypeOpaque:
		n = &OpaqueNodeID{}
	default:
		return nil, &errors.ErrInvalidType{
			Type:   encodingMask,
			Action: "decode",
			Msg:    "got undefined type",
		}
	}

	if err := n.DecodeFromBytes(b); err != nil {
		return nil, err
	}
	return n, nil
}

// TwoByteNodeID represents the TwoByteNodeId.
type TwoByteNodeID struct {
	EncodingMask uint8
	Identifier   byte
}

// DecodeFromBytes decodes given bytes into TwoByteNodeID.
func (t *TwoByteNodeID) DecodeFromBytes(b []byte) error {
	if len(b) != 2 {
		return &errors.ErrTooShortToDecode{t, "should be 2 bytes"}
	}

	t.EncodingMask = b[0]
	t.Identifier = b[1]
	return nil
}

// Len returns the actual length of TwoByteNodeID in int.
func (t *TwoByteNodeID) Len() int {
	return 2
}

// EncodingMaskValue returns EncodingMask in uint8.
func (t *TwoByteNodeID) EncodingMaskValue() uint8 {
	return t.EncodingMask
}

// FourByteNodeID represents the FourByteNodeId.
type FourByteNodeID struct {
	EncodingMask uint8
	Namespace    uint8
	Identifier   uint16
}

// DecodeFromBytes decodes given bytes into FourByteNodeID.
func (f *FourByteNodeID) DecodeFromBytes(b []byte) error {
	if len(b) != 4 {
		return &errors.ErrTooShortToDecode{f, "should be 4 bytes"}
	}

	f.EncodingMask = b[0]
	f.Namespace = b[1]
	f.Identifier = binary.LittleEndian.Uint16(b[2:4])
	return nil
}

// Len returns the actual length of FourByteNodeID in int.
func (f *FourByteNodeID) Len() int {
	return 4
}

// EncodingMaskValue returns EncodingMask in uint8.
func (f *FourByteNodeID) EncodingMaskValue() uint8 {
	return f.EncodingMask
}

// NumericNodeID represents the NumericNodeId.
type NumericNodeID struct {
	EncodingMask uint8
	Namespace    uint16
	Identifier   uint32
}

// DecodeFromBytes decodes given bytes into NumericNodeID.
func (n *NumericNodeID) DecodeFromBytes(b []byte) error {
	if len(b) != 7 {
		return &errors.ErrTooShortToDecode{n, "should be 7 bytes"}
	}

	n.EncodingMask = b[0]
	n.Namespace = binary.LittleEndian.Uint16(b[1:3])
	n.Identifier = binary.LittleEndian.Uint32(b[3:7])
	return nil
}

// Len returns the actual length of NumericNodeID in int.
func (n *NumericNodeID) Len() int {
	return 7
}

// EncodingMaskValue returns EncodingMask in uint8.
func (n *NumericNodeID) EncodingMaskValue() uint8 {
	return n.EncodingMask
}

// StringNodeID represents the StringNodeId.
type StringNodeID struct {
	EncodingMask uint8
	Namespace    uint16
	Length       uint32
	Identifier   []byte
}

// DecodeFromBytes decodes given bytes into StringNodeID.
func (s *StringNodeID) DecodeFromBytes(b []byte) error {
	if len(b) < 7 {
		return &errors.ErrTooShortToDecode{s, "should be longer than 7 bytes"}
	}

	s.EncodingMask = b[0]
	s.Namespace = binary.LittleEndian.Uint16(b[1:3])
	s.Length = binary.LittleEndian.Uint32(b[3:7])
	copy(s.Identifier, b[7:])
	return nil
}

// Len returns the actual length of StringNodeID in int.
func (s *StringNodeID) Len() int {
	return 7 + len(s.Identifier)
}

// EncodingMaskValue returns EncodingMask in uint8.
func (s *StringNodeID) EncodingMaskValue() uint8 {
	return s.EncodingMask
}

// GUIDNodeID represents the GUIDNodeId.
type GUIDNodeID struct {
	EncodingMask uint8
	Namespace    uint16
	Identifier   *GUID
}

// DecodeFromBytes decodes given bytes into GUIDNodeID.
func (g *GUIDNodeID) DecodeFromBytes(b []byte) error {
	if len(b) != 20 {
		return &errors.ErrTooShortToDecode{g, "should be 20 bytes"}
	}

	g.EncodingMask = b[0]
	g.Namespace = binary.LittleEndian.Uint16(b[1:3])
	if err := g.Identifier.DecodeFromBytes(b[4:20]); err != nil {
		return err
	}

	return nil
}

// Len returns the actual length of GUIDNodeID in int.
func (g *GUIDNodeID) Len() int {
	return 20
}

// EncodingMaskValue returns EncodingMask in uint8.
func (g *GUIDNodeID) EncodingMaskValue() uint8 {
	return g.EncodingMask
}

// OpaqueNodeID represents the OpaqueNodeId.
type OpaqueNodeID struct {
	EncodingMask uint8
	Namespace    uint16
	Length       uint32
	Identifier   []byte
}

// DecodeFromBytes decodes given bytes into OpaqueNodeID.
func (o *OpaqueNodeID) DecodeFromBytes(b []byte) error {
	if len(b) < 7 {
		return &errors.ErrTooShortToDecode{o, "should be longer than 7 bytes"}
	}

	o.EncodingMask = b[0]
	o.Namespace = binary.LittleEndian.Uint16(b[1:3])
	o.Length = binary.LittleEndian.Uint32(b[3:7])
	copy(o.Identifier, b[7:])
	return nil
}

// Len returns the actual length of OpaqueNodeID in int.
func (o *OpaqueNodeID) Len() int {
	return 7 + len(o.Identifier)
}

// EncodingMaskValue returns EncodingMask in uint8.
func (o *OpaqueNodeID) EncodingMaskValue() uint8 {
	return o.EncodingMask
}

// ExpandedNodeID represents the ExpandedNodeID.
type ExpandedNodeID struct {
	NodeID
	NamespaceURI String
	ServerIndex  uint32
}

// DecodeFromBytes decodes given bytes into ExpandedNodeID.
func (e *ExpandedNodeID) DecodeFromBytes(b []byte) error {
	if err := e.NodeID.DecodeFromBytes(b); err != nil {
		return err
	}
	b = b[e.NodeID.Len():]
	if len(b) < 2 {
		return nil
	}

	if e.HasNamespaceURI() {
		if err := e.NamespaceURI.DecodeFromBytes(b); err != nil {
			return err
		}
		b = b[e.NamespaceURI.Len():]
	}

	if e.HasServerIndex() {
		e.ServerIndex = binary.LittleEndian.Uint32(b[:4])
	}

	return nil
}

// Len returns the actual length of ExpandedNodeID in int.
func (e *ExpandedNodeID) Len() int {
	if e.NodeID == nil {
		return 0
	}

	l := e.NodeID.Len()
	if e.HasNamespaceURI() {
		l += e.NamespaceURI.Len()
	}
	if e.HasServerIndex() {
		l += 4
	}

	return l
}

// HasNamespaceURI checks if an ExpandedNodeID has NamespaceURI Flag.
func (e *ExpandedNodeID) HasNamespaceURI() bool {
	return e.NodeID.EncodingMaskValue()>>7&0x1 == 1
}

// HasServerIndex checks if an ExpandedNodeID has ServerIndex Flag.
func (e *ExpandedNodeID) HasServerIndex() bool {
	return e.NodeID.EncodingMaskValue()>>6&0x1 == 1
}
