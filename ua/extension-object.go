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
	TypeID *ExpandedNodeID
	Value  interface{}
}

func NewExtensionObject(value interface{}) *ExtensionObject {
	return &ExtensionObject{
		TypeID: ExtensionObjectTypeID(value),
		Value:  value,
	}
}

func (e *ExtensionObject) Decode(b []byte) (int, error) {
	buf := NewBuffer(b)
	e.TypeID = new(ExpandedNodeID)
	buf.ReadStruct(e.TypeID)

	mask := buf.ReadByte()
	if mask == ExtensionObjectNoBody {
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

	switch mask {
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
	if e.TypeID == nil {
		buf.WriteStruct(NewTwoByteExpandedNodeID(0))
	} else {
		buf.WriteStruct(e.TypeID)
	}

	switch e.Value.(type) {
	case *XmlElement:
		buf.WriteByte(ExtensionObjectXMLElementBody)
	default:
		if e.Value != nil {
			buf.WriteByte(ExtensionObjectByteStringBody)
		} else {
			buf.WriteByte(ExtensionObjectNoBody)
			return buf.Bytes(), buf.Error()
		}
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

func ExtensionObjectTypeID(v interface{}) *ExpandedNodeID {
	switch v.(type) {
	case *AnonymousIdentityToken:
		return NewFourByteExpandedNodeID(0, id.AnonymousIdentityToken_Encoding_DefaultBinary)
	case *UserNameIdentityToken:
		return NewFourByteExpandedNodeID(0, id.UserNameIdentityToken_Encoding_DefaultBinary)
	case *X509IdentityToken:
		return NewFourByteExpandedNodeID(0, id.X509IdentityToken_Encoding_DefaultBinary)
	case *IssuedIdentityToken:
		return NewFourByteExpandedNodeID(0, id.IssuedIdentityToken_Encoding_DefaultBinary)
	default:
		return NewTwoByteExpandedNodeID(0)
	}
}
