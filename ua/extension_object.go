// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"bytes"

	"github.com/gopcua/opcua/codec"
	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/id"
)

// eotypes contains all known extension objects.
var eotypes = NewTypeRegistry()

// RegisterExtensionObject registers a new extension object type.
// It panics if the type or the id is already registered.
func RegisterExtensionObject(typeID *NodeID, v interface{}) {
	if err := eotypes.Register(typeID, v); err != nil {
		panic("Extension object " + err.Error())
	}
}

// These flags define the value type of an ExtensionObject.
// They cannot be combined.
const (
	ExtensionObjectEmpty  = 0
	ExtensionObjectBinary = 1
	ExtensionObjectXML    = 2
)

// ExtensionObject is encoded as sequence of bytes prefixed by the NodeId of its DataTypeEncoding
// and the number of bytes encoded.
//
// Specification: Part 6, 5.2.2.15
type ExtensionObject struct {
	EncodingMask uint8
	TypeID       *ExpandedNodeID
	Value        interface{}
}

func NewExtensionObject(value interface{}) *ExtensionObject {
	e := &ExtensionObject{
		TypeID: ExtensionObjectTypeID(value),
		Value:  value,
	}
	e.UpdateMask()
	return e
}

func (e *ExtensionObject) Decode(b []byte) (int, error) {
	buf := NewBuffer(b)
	e.TypeID = new(ExpandedNodeID)
	buf.ReadStruct(e.TypeID)

	e.EncodingMask = buf.ReadByte()
	if e.EncodingMask == ExtensionObjectEmpty {
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

	if e.EncodingMask == ExtensionObjectXML {
		e.Value = new(XMLElement)
		body.ReadStruct(e.Value)
		return buf.Pos(), body.Error()
	}

	typeID := e.TypeID.NodeID
	e.Value = eotypes.New(typeID)
	if e.Value == nil {
		debug.Printf("ua: unknown extension object %s", typeID)
		return buf.Pos(), buf.Error()
	}

	body.ReadStruct(e.Value)
	return buf.Pos(), body.Error()
}

func (e *ExtensionObject) MarshalOPCUA() ([]byte, error) {
	var buf bytes.Buffer
	b, err := codec.Marshal(e.TypeID)
	buf.Write(b)
	buf.WriteByte(e.EncodingMask)
	if e.EncodingMask == ExtensionObjectEmpty {
		return buf.Bytes(), err
	}

	body, err := codec.Marshal(e.Value)
	if err != nil {
		return buf.Bytes(), err
	}
	n := uint32(len(body))
	buf.Write([]byte{byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24)})
	buf.Write(body)
	return buf.Bytes(), err
}

func (e *ExtensionObject) UpdateMask() {
	if e.Value == nil {
		e.EncodingMask = ExtensionObjectEmpty
	} else if _, ok := e.Value.(*XMLElement); ok {
		e.EncodingMask = ExtensionObjectXML
	} else {
		e.EncodingMask = ExtensionObjectBinary
	}
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
	case *ServerStatusDataType:
		return NewFourByteExpandedNodeID(0, id.ServerStatusDataType_Encoding_DefaultBinary)
	default:
		if id := eotypes.Lookup(v); id != nil {
			return &ExpandedNodeID{NodeID: id}
		}
		return NewTwoByteExpandedNodeID(0)
	}
}
