// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"fmt"

	"github.com/wmnsk/gopcua/id"
	"github.com/wmnsk/gopcua/ua"
)

// ExtensionObject is encoded as sequence of bytes prefixed by the NodeId of its DataTypeEncoding
// and the number of bytes encoded.
//
// Specification: Part 6, 5.2.2.15
type ExtensionObject struct {
	TypeID       *ExpandedNodeID
	EncodingMask byte
	Value        ExtensionObjectValue
}

// NewExtensionObject creates a new ExtensionObject from the ExtensionObjectValue given.
func NewExtensionObject(mask uint8, extParam ExtensionObjectValue) *ExtensionObject {
	return &ExtensionObject{
		TypeID: NewExpandedNodeID(
			false, false, NewFourByteNodeID(0, uint16(extParam.Type())), "", 0,
		),
		EncodingMask: mask,
		Value:        extParam,
	}
}

func (e *ExtensionObject) Decode(b []byte) (int, error) {
	buf := ua.NewBuffer(b)
	e.TypeID = new(ExpandedNodeID)
	buf.ReadStruct(e.TypeID)
	e.EncodingMask = buf.ReadByte()
	length := buf.ReadUint32()
	if length == 0xffffffff || length == 0 {
		return 0, buf.Error()
	}

	e.Value = NewExtensionObjectValue(e.TypeID.NodeID)
	if e.Value == nil {
		return 0, fmt.Errorf("invalid ExtensionObjectValue")
	}
	buf.ReadStruct(e.Value)
	return buf.Pos(), buf.Error()
}

func (e *ExtensionObject) Encode() ([]byte, error) {
	buf := ua.NewBuffer(nil)
	buf.WriteStruct(e.TypeID)
	buf.WriteByte(e.EncodingMask)
	if e.Value == nil {
		buf.WriteUint32(0xffffffff)
		return nil, buf.Error()
	}
	d, err := e.Value.Encode()
	if err != nil {
		return nil, err
	}
	buf.WriteByteString(d)
	return buf.Bytes(), buf.Error()
}

// ExtensionObjectValue represents the value in ExtensionObject.
type ExtensionObjectValue interface {
	ua.BinaryDecoder
	ua.BinaryEncoder
	Type() int
}

func NewExtensionObjectValue(n *NodeID) ExtensionObjectValue {
	switch n.IntID() {
	case id.AnonymousIdentityToken_Encoding_DefaultBinary:
		return new(AnonymousIdentityToken)
	case id.UserNameIdentityToken_Encoding_DefaultBinary:
		return new(UserNameIdentityToken)
	case id.X509IdentityToken_Encoding_DefaultBinary:
		return new(X509IdentityToken)
	case id.IssuedIdentityToken_Encoding_DefaultBinary:
		return new(IssuedIdentityToken)
	default:
		return nil
	}
}
