// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"fmt"
	"reflect"

	"github.com/gopcua/opcua/debug"
)

// eotypes contains all known extension objects.
var eotypes = NewFuncRegistry()

// RegisterExtensionObject registers a new extension object type.
// It panics if the type or the id is already registered.
func RegisterExtensionObject(typeID *NodeID, v interface{}) {
	RegisterExtensionObjectFunc(typeID, DefaultEncodeExtensionObject, DefaultDecodeExtensionObject(v))
}

// RegisterExtensionObjectFunc registers a new extension object type using encode and decode functions
// It panics if the type or the id is already registered.
func RegisterExtensionObjectFunc(typeID *NodeID, ef encodefunc, df decodefunc) {
	if err := eotypes.Register(typeID, ef, df); err != nil {
		panic("Extension object " + err.Error())
	}
}

func Deregister(typeID *NodeID) {
	eotypes.Deregister(typeID)
}

// These flags define the value type of an ExtensionObject.
// They cannot be combined.
const (
	ExtensionObjectEmpty  = 0
	ExtensionObjectBinary = 1
	ExtensionObjectXML    = 2
)

type encodefunc func(v interface{}) ([]byte, error)
type decodefunc func(b []byte, v interface{}) error

// ExtensionObject is encoded as sequence of bytes prefixed by the NodeId of its DataTypeEncoding
// and the number of bytes encoded.
//
// Specification: Part 6, 5.2.2.15
type ExtensionObject struct {
	EncodingMask uint8
	TypeID       *ExpandedNodeID
	Value        interface{}
}

func NewExtensionObject(value interface{}, typeID *ExpandedNodeID) *ExtensionObject {
	e := &ExtensionObject{
		TypeID: typeID,
		Value:  value,
	}
	e.UpdateMask()
	return e
}

// Decode fails if there is no decode func registered for e
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
	decode := eotypes.DecodeFunc(typeID)
	if decode == nil {
		debug.Printf("ua: unknown extension object %s", typeID)
		return buf.Pos(), buf.Error()
	}
	err := decode(body.Bytes(), &e.Value)
	if err != nil {
		// TODO: we are losing Pos by creating new buf in decode?
		return buf.Pos(), err
	}
	// TODO: we are losing Pos by creating new buf in decode?
	return buf.Pos(), body.Error()
}

// Encode falls back to defaultencode if there is no encode func registered for e
func (e *ExtensionObject) Encode() ([]byte, error) {
	if e == nil {
		e = &ExtensionObject{TypeID: NewTwoByteExpandedNodeID(0), EncodingMask: ExtensionObjectEmpty}
	}

	typeID := e.TypeID.NodeID
	encode := eotypes.EncodeFunc(typeID)
	if encode == nil {
		debug.Printf("ua: unknown extension object %s", typeID)
		return DefaultEncodeExtensionObject(e)
	}
	return encode(e)
}

// DefaultDecode creates a new instance of v and decodes into it
func DefaultDecodeExtensionObject(v interface{}) func([]byte, interface{}) error {
	return func(b []byte, vv interface{}) error {
		rv := reflect.ValueOf(vv)
		if rv.Kind() != reflect.Pointer || rv.IsNil() {
			return fmt.Errorf("incorrect type to decode into")
		}
		r := reflect.New(reflect.TypeOf(v).Elem()).Interface()
		buf := NewBuffer(b)
		buf.ReadStruct(r)
		reflect.Indirect(rv).Set(reflect.ValueOf(r))
		return nil
	}
}

// DefaultEncode encodes into bytes based on the go struct
func DefaultEncodeExtensionObject(v interface{}) ([]byte, error) {
	e, ok := v.(*ExtensionObject)
	if !ok {
		return nil, fmt.Errorf("expected ExtensionObject")
	}
	buf := NewBuffer(nil)
	buf.WriteStruct(e.TypeID)
	buf.WriteByte(e.EncodingMask)
	if e.EncodingMask == ExtensionObjectEmpty {
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

func (e *ExtensionObject) UpdateMask() {
	if e.Value == nil {
		e.EncodingMask = ExtensionObjectEmpty
	} else if _, ok := e.Value.(*XMLElement); ok {
		e.EncodingMask = ExtensionObjectXML
	} else {
		e.EncodingMask = ExtensionObjectBinary
	}
}
