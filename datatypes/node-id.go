// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/errors"
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

// NodeID is an interface to handle all types of NodeId.
type NodeID interface {
	Serialize() ([]byte, error)
	SerializeTo([]byte) error
	DecodeFromBytes([]byte) error
	Len() int
	String() string
	EncodingMaskValue() uint8
	SetURIFlag()
	SetIndexFlag()
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
// It is a numeric value that fits into the two-byte representation.
//
// Specification: Part 6, 5.2.2.9
type TwoByteNodeID struct {
	EncodingMask uint8
	Identifier   byte
}

// NewTwoByteNodeID creates a new TwoByteNodeID.
func NewTwoByteNodeID(val byte) *TwoByteNodeID {
	t := &TwoByteNodeID{
		EncodingMask: TypeTwoByte,
		Identifier:   val,
	}

	return t
}

// DecodeFromBytes decodes given bytes into TwoByteNodeID.
func (t *TwoByteNodeID) DecodeFromBytes(b []byte) error {
	if len(b) < 2 {
		return &errors.ErrTooShortToDecode{t, "should be 2 bytes"}
	}

	t.EncodingMask = b[0]
	t.Identifier = b[1]
	return nil
}

// Serialize serializes TwoByteNodeID into bytes.
func (t *TwoByteNodeID) Serialize() ([]byte, error) {
	b := make([]byte, t.Len())
	if err := t.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes TwoByteNodeID into bytes.
func (t *TwoByteNodeID) SerializeTo(b []byte) error {
	b[0] = t.EncodingMask
	b[1] = t.Identifier

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

// SetURIFlag sets NamespaceURI flag in EncodingMask.
func (t *TwoByteNodeID) SetURIFlag() {
	t.EncodingMask |= 0x80
}

// SetIndexFlag sets NamespaceURI flag in EncodingMask.
func (t *TwoByteNodeID) SetIndexFlag() {
	t.EncodingMask |= 0x40
}

// String returns the values in TwoByteNodeID in string.
func (t *TwoByteNodeID) String() string {
	return fmt.Sprintf("%x, %d", t.EncodingMask, t.Identifier)
}

// FourByteNodeID represents the FourByteNodeId.
// It is a numeric value that fits into the four-byte representation.
//
// Specification: Part 6, 5.2.2.9
type FourByteNodeID struct {
	EncodingMask uint8
	Namespace    uint8
	Identifier   uint16
}

// NewFourByteNodeID creates a new FourByteNodeID.
func NewFourByteNodeID(idx uint8, val uint16) *FourByteNodeID {
	f := &FourByteNodeID{
		EncodingMask: TypeFourByte,
		Namespace:    idx,
		Identifier:   val,
	}

	return f
}

// DecodeFromBytes decodes given bytes into FourByteNodeID.
func (f *FourByteNodeID) DecodeFromBytes(b []byte) error {
	if len(b) < 4 {
		return &errors.ErrTooShortToDecode{f, "should be 4 bytes"}
	}

	f.EncodingMask = b[0]
	f.Namespace = b[1]
	f.Identifier = binary.LittleEndian.Uint16(b[2:4])
	return nil
}

// Serialize serializes FourByteNodeID into bytes.
func (f *FourByteNodeID) Serialize() ([]byte, error) {
	b := make([]byte, f.Len())
	if err := f.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes FourByteNodeID into bytes.
func (f *FourByteNodeID) SerializeTo(b []byte) error {
	b[0] = f.EncodingMask
	b[1] = f.Namespace
	binary.LittleEndian.PutUint16(b[2:], f.Identifier)

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

// SetURIFlag sets NamespaceURI flag in EncodingMask.
func (f *FourByteNodeID) SetURIFlag() {
	f.EncodingMask |= 0x80
}

// SetIndexFlag sets NamespaceURI flag in EncodingMask.
func (f *FourByteNodeID) SetIndexFlag() {
	f.EncodingMask |= 0x40
}

// String returns the values in FourByteNodeID in string.
func (f *FourByteNodeID) String() string {
	return fmt.Sprintf("%x, %d, %d", f.EncodingMask, f.Namespace, f.Identifier)
}

// NumericNodeID represents the NumericNodeId.
// It is a numeric value that does not fit into the two or four byte representations.
//
// Specification: Part 6, 5.2.2.9
type NumericNodeID struct {
	EncodingMask uint8
	Namespace    uint16
	Identifier   uint32
}

// NewNumericNodeID creates a new NumericNodeID.
func NewNumericNodeID(idx uint16, val uint32) *NumericNodeID {
	n := &NumericNodeID{
		EncodingMask: TypeNumeric,
		Namespace:    idx,
		Identifier:   val,
	}

	return n
}

// DecodeFromBytes decodes given bytes into NumericNodeID.
func (n *NumericNodeID) DecodeFromBytes(b []byte) error {
	if len(b) < 7 {
		return &errors.ErrTooShortToDecode{n, "should be 7 bytes"}
	}

	n.EncodingMask = b[0]
	n.Namespace = binary.LittleEndian.Uint16(b[1:3])
	n.Identifier = binary.LittleEndian.Uint32(b[3:7])
	return nil
}

// Serialize serializes NumericNodeID into bytes.
func (n *NumericNodeID) Serialize() ([]byte, error) {
	b := make([]byte, n.Len())
	if err := n.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes NumericNodeID into bytes.
func (n *NumericNodeID) SerializeTo(b []byte) error {
	b[0] = n.EncodingMask
	binary.LittleEndian.PutUint16(b[1:3], n.Namespace)
	binary.LittleEndian.PutUint32(b[3:7], n.Identifier)

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

// SetURIFlag sets NamespaceURI flag in EncodingMask.
func (n *NumericNodeID) SetURIFlag() {
	n.EncodingMask |= 0x80
}

// SetIndexFlag sets NamespaceURI flag in EncodingMask.
func (n *NumericNodeID) SetIndexFlag() {
	n.EncodingMask |= 0x40
}

// String returns the values in NumericNodeID in string.
func (n *NumericNodeID) String() string {
	return fmt.Sprintf("%x, %d, %d", n.EncodingMask, n.Namespace, n.Identifier)
}

// StringNodeID represents the StringNodeId.
//
// Specification: Part 6, 5.2.2.9
type StringNodeID struct {
	EncodingMask uint8
	Namespace    uint16
	Length       uint32
	Identifier   []byte
}

// NewStringNodeID creates a new StringNodeID.
func NewStringNodeID(idx uint16, val string) *StringNodeID {
	n := &StringNodeID{
		EncodingMask: TypeString,
		Namespace:    idx,
		Identifier:   []byte(val),
	}
	n.Length = uint32(len(val))

	return n
}

// DecodeFromBytes decodes given bytes into StringNodeID.
func (s *StringNodeID) DecodeFromBytes(b []byte) error {
	if len(b) < 7 {
		return &errors.ErrTooShortToDecode{s, "should be longer than 7 bytes"}
	}

	s.EncodingMask = b[0]
	s.Namespace = binary.LittleEndian.Uint16(b[1:3])
	s.Length = binary.LittleEndian.Uint32(b[3:7])
	s.Identifier = b[7:]
	return nil
}

// Serialize serializes StringNodeID into bytes.
func (s *StringNodeID) Serialize() ([]byte, error) {
	b := make([]byte, s.Len())
	if err := s.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes StringNodeID into bytes.
func (s *StringNodeID) SerializeTo(b []byte) error {
	b[0] = s.EncodingMask
	binary.LittleEndian.PutUint16(b[1:3], s.Namespace)
	binary.LittleEndian.PutUint32(b[3:7], s.Length)
	copy(b[7:7+int(s.Length)], s.Identifier)

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

// SetURIFlag sets NamespaceURI flag in EncodingMask.
func (s *StringNodeID) SetURIFlag() {
	s.EncodingMask |= 0x80
}

// SetIndexFlag sets NamespaceURI flag in EncodingMask.
func (s *StringNodeID) SetIndexFlag() {
	s.EncodingMask |= 0x40
}

// Value returns Identifier in string.
func (s *StringNodeID) Value() string {
	return string(s.Identifier)
}

// String returns the values in StringNodeID in string.
func (s *StringNodeID) String() string {
	return fmt.Sprintf("%x, %d, %d, %d", s.EncodingMask, s.Namespace, s.Length, s.Identifier)
}

// GUIDNodeID represents the GUIDNodeId.
//
// Specification: Part 6, 5.2.2.9
type GUIDNodeID struct {
	EncodingMask uint8
	Namespace    uint16
	Identifier   *GUID
}

// NewGUIDNodeID creates a new GUIDNodeID.
func NewGUIDNodeID(idx uint16, val string) *GUIDNodeID {
	guid := NewGUID(val)
	n := &GUIDNodeID{
		EncodingMask: TypeGUID,
		Namespace:    idx,
		Identifier:   guid,
	}

	return n
}

// DecodeFromBytes decodes given bytes into GUIDNodeID.
func (g *GUIDNodeID) DecodeFromBytes(b []byte) error {
	if len(b) < 19 {
		return &errors.ErrTooShortToDecode{g, "should be 19 bytes"}
	}

	g.EncodingMask = b[0]
	g.Namespace = binary.LittleEndian.Uint16(b[1:3])
	g.Identifier = &GUID{}
	return g.Identifier.DecodeFromBytes(b[3:19])
}

// Serialize serializes GUIDNodeID into bytes.
func (g *GUIDNodeID) Serialize() ([]byte, error) {
	b := make([]byte, g.Len())
	if err := g.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes GUIDNodeID into bytes.
func (g *GUIDNodeID) SerializeTo(b []byte) error {
	b[0] = g.EncodingMask
	binary.LittleEndian.PutUint16(b[1:3], g.Namespace)
	return g.Identifier.SerializeTo(b[3:])
}

// Len returns the actual length of GUIDNodeID in int.
func (g *GUIDNodeID) Len() int {
	return 19
}

// EncodingMaskValue returns EncodingMask in uint8.
func (g *GUIDNodeID) EncodingMaskValue() uint8 {
	return g.EncodingMask
}

// SetURIFlag sets NamespaceURI flag in EncodingMask.
func (g *GUIDNodeID) SetURIFlag() {
	g.EncodingMask |= 0x80
}

// SetIndexFlag sets NamespaceURI flag in EncodingMask.
func (g *GUIDNodeID) SetIndexFlag() {
	g.EncodingMask |= 0x40
}

// Value returns Identifier in string.
func (g *GUIDNodeID) Value() string {
	return g.Identifier.String()
}

// String returns the values in GUIDNodeID in string.
func (g *GUIDNodeID) String() string {
	return fmt.Sprintf("%x, %d, %v", g.EncodingMask, g.Namespace, g.Identifier)
}

// OpaqueNodeID represents the OpaqueNodeId.
//
// Specification: Part 6, 5.2.2.9
type OpaqueNodeID struct {
	EncodingMask uint8
	Namespace    uint16
	Length       uint32
	Identifier   []byte
}

// NewOpaqueNodeID creates a new OpaqueNodeID.
func NewOpaqueNodeID(idx uint16, val []byte) *OpaqueNodeID {
	n := &OpaqueNodeID{
		EncodingMask: TypeOpaque,
		Namespace:    idx,
		Identifier:   val,
	}
	n.Length = uint32(len(val))

	return n
}

// DecodeFromBytes decodes given bytes into OpaqueNodeID.
func (o *OpaqueNodeID) DecodeFromBytes(b []byte) error {
	if len(b) < 7 {
		return &errors.ErrTooShortToDecode{o, "should be longer than 7 bytes"}
	}

	o.EncodingMask = b[0]
	o.Namespace = binary.LittleEndian.Uint16(b[1:3])
	o.Length = binary.LittleEndian.Uint32(b[3:7])
	o.Identifier = b[7 : 7+int(o.Length)]

	return nil
}

// Serialize serializes OpaqueNodeID into bytes.
func (o *OpaqueNodeID) Serialize() ([]byte, error) {
	b := make([]byte, o.Len())
	if err := o.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes OpaqueNodeID into bytes.
func (o *OpaqueNodeID) SerializeTo(b []byte) error {
	b[0] = o.EncodingMask
	binary.LittleEndian.PutUint16(b[1:3], o.Namespace)
	binary.LittleEndian.PutUint32(b[3:7], o.Length)
	copy(b[7:7+int(o.Length)], o.Identifier)

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

// SetURIFlag sets NamespaceURI flag in EncodingMask.
func (o *OpaqueNodeID) SetURIFlag() {
	o.EncodingMask |= 0x80
}

// SetIndexFlag sets NamespaceURI flag in EncodingMask.
func (o *OpaqueNodeID) SetIndexFlag() {
	o.EncodingMask |= 0x40
}

// String returns the values in OpaqueNodeID in string.
func (o *OpaqueNodeID) String() string {
	return fmt.Sprintf("%x, %d, %d, %d", o.EncodingMask, o.Namespace, o.Length, o.Identifier)
}
