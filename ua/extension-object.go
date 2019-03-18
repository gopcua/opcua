// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"github.com/gopcua/opcua/id"
)

const (
	ExtensionObjectNoBody         = 0
	ExtensionObjectByteStringBody = 1
	ExtensionObjectXMLElementBody = 2
)

// ExtensionObject is encoded as sequence of bytes prefixed by the NodeId of its DataTypeEncoding
// and the number of bytes encoded.
//
// Specification: Part 6, 5.2.2.15
type ExtensionObject struct {
	TypeID       *ExpandedNodeID
	EncodingMask byte
	Value        interface{}
}

// NewExtensionObject creates a new ExtensionObject from the ExtensionObjectValue given.
func NewExtensionObject(mask uint8, value interface{}) *ExtensionObject {
	return &ExtensionObject{
		TypeID:       NewFourByteExpandedNodeID(0, ExtObjID(value)),
		EncodingMask: mask,
		Value:        value,
	}
}

func NewNullExtensionObject() *ExtensionObject {
	return &ExtensionObject{
		TypeID: NewTwoByteExpandedNodeID(0),
	}
}

func (e *ExtensionObject) Decode(b []byte) (int, error) {
	buf := NewBuffer(b)
	e.TypeID = new(ExpandedNodeID)
	buf.ReadStruct(e.TypeID)
	e.EncodingMask = buf.ReadByte()

	if e.EncodingMask == ExtensionObjectNoBody {
		return buf.Pos(), buf.Error()
	}

	length := buf.ReadUint32()
	if length == 0 || length == 0xffffffff || buf.Error() != nil {
		return buf.Pos(), buf.Error()
	}

	body := NewBuffer(buf.ReadN(int(length)))
	if buf.Error() != nil {
		return buf.Pos(), buf.Error()
	}

	switch e.EncodingMask {
	case ExtensionObjectXMLElementBody:
		e.Value = new(XmlElement)
		body.ReadStruct(e.Value)

	case ExtensionObjectByteStringBody:
		switch e.TypeID.NodeID.IntID() {
		case id.AnonymousIdentityToken_Encoding_DefaultBinary:
			e.Value = new(AnonymousIdentityToken)
			body.ReadStruct(e.Value)

		case id.UserNameIdentityToken_Encoding_DefaultBinary:
			e.Value = new(UserNameIdentityToken)
			body.ReadStruct(e.Value)

		case id.X509IdentityToken_Encoding_DefaultBinary:
			e.Value = new(X509IdentityToken)
			body.ReadStruct(e.Value)

		case id.IssuedIdentityToken_Encoding_DefaultBinary:
			e.Value = new(IssuedIdentityToken)
			body.ReadStruct(e.Value)

		default:
			e.Value = body.ReadBytes()
		}

	default:
		e.Value = buf.ReadBytes()
	}

	return buf.Pos(), body.Error()
}

func (e *ExtensionObject) Encode() ([]byte, error) {
	buf := NewBuffer(nil)
	buf.WriteStruct(e.TypeID)
	buf.WriteByte(e.EncodingMask)

	if e.EncodingMask == ExtensionObjectNoBody {
		return buf.Bytes(), buf.Error()
	}

	if e.Value == nil {
		buf.WriteUint32(0xffffffff)
		return buf.Bytes(), buf.Error()
	}

	body := NewBuffer(nil)
	body.WriteStruct(e.Value)
	if body.Error() != nil {
		return nil, body.Error()
	}

	buf.WriteUint32(uint32(body.Len()))
	buf.Write(body.Bytes())
	return buf.Bytes(), buf.Error()
}

func ExtObjID(v interface{}) uint16 {
	switch v.(type) {
	case *AnonymousIdentityToken:
		return id.AnonymousIdentityToken_Encoding_DefaultBinary
	case *UserNameIdentityToken:
		return id.UserNameIdentityToken_Encoding_DefaultBinary
	case *X509IdentityToken:
		return id.X509IdentityToken_Encoding_DefaultBinary
	case *IssuedIdentityToken:
		return id.IssuedIdentityToken_Encoding_DefaultBinary
	default:
		return 0
	}
}
