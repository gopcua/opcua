// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/binary"
)

// WriteValue is a set of Node and Attribute to write.
//
// Specification: Part4, 5.10.4.2
type WriteValue struct {
	NodeID
	AttributeID IntegerID
	IndexRange  *String
	Value       *DataValue
}

// NewWriteValue creates a new NewWriteValue.
func NewWriteValue(node NodeID, attr IntegerID, idxRange string, value *DataValue) *WriteValue {
	return &WriteValue{
		NodeID:      node,
		AttributeID: attr,
		IndexRange:  NewString(idxRange),
		Value:       value,
	}
}

// DecodeWriteValue decodes given bytes into WriteValue.
func DecodeWriteValue(b []byte) (*WriteValue, error) {
	w := &WriteValue{}
	if err := w.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return w, nil
}

// DecodeFromBytes decodes given bytes into OPC UA WriteValue.
func (w *WriteValue) DecodeFromBytes(b []byte) error {
	nodeID, err := DecodeNodeID(b)
	if err != nil {
		return err
	}
	w.NodeID = nodeID
	offset := w.NodeID.Len()

	w.AttributeID = IntegerID(binary.LittleEndian.Uint32(b[offset:]))
	offset += 4

	w.IndexRange = &String{}
	if err := w.IndexRange.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += w.IndexRange.Len()

	w.Value = &DataValue{}
	if err := w.Value.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	return nil
}

// Serialize serializes WriteValue into bytes.
func (w *WriteValue) Serialize() ([]byte, error) {
	b := make([]byte, w.Len())
	if err := w.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes WriteValue into bytes.
func (w *WriteValue) SerializeTo(b []byte) error {
	offset := 0

	if w.NodeID != nil {
		if err := w.NodeID.SerializeTo(b); err != nil {
			return err
		}
		offset += w.NodeID.Len()
	}

	binary.LittleEndian.PutUint32(b[offset:offset+4], uint32(w.AttributeID))
	offset += 4

	if w.IndexRange != nil {
		if err := w.IndexRange.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += w.IndexRange.Len()
	}

	if w.Value != nil {
		return w.Value.SerializeTo(b[offset:])
	}

	return nil
}

// Len returns the actual length of WriteValue in int.
func (w *WriteValue) Len() int {
	l := 4

	if w.NodeID != nil {
		l += w.NodeID.Len()
	}

	if w.IndexRange != nil {
		l += w.IndexRange.Len()
	}

	if w.Value != nil {
		l += w.Value.Len()
	}

	return l
}

// WriteValueArray represents an array of WriteValues.
// It does not correspond to a certain type from the specification
// but makes encoding and decoding easier.
type WriteValueArray struct {
	ArraySize   int32
	WriteValues []*WriteValue
}

// NewWriteValueArray creates a new WriteValueArray from multiple WriteValues.
func NewWriteValueArray(ids []*WriteValue) *WriteValueArray {
	if ids == nil {
		r := &WriteValueArray{
			ArraySize: 0,
		}
		return r
	}

	r := &WriteValueArray{
		ArraySize:   int32(len(ids)),
		WriteValues: ids,
	}

	return r
}

// DecodeWriteValueArray decodes given bytes into WriteValueArray.
func DecodeWriteValueArray(b []byte) (*WriteValueArray, error) {
	w := &WriteValueArray{}
	if err := w.DecodeFromBytes(b); err != nil {
		return nil, err
	}
	return w, nil
}

// DecodeFromBytes decodes given bytes into WriteValueArray.
func (w *WriteValueArray) DecodeFromBytes(b []byte) error {
	w.ArraySize = int32(binary.LittleEndian.Uint32(b[:4]))
	if w.ArraySize <= 0 {
		return nil
	}

	offset := 4
	for i := 1; i <= int(w.ArraySize); i++ {
		id, err := DecodeWriteValue(b[offset:])
		if err != nil {
			return err
		}
		w.WriteValues = append(w.WriteValues, id)
		offset += id.Len()
	}

	return nil
}

// Serialize serializes WriteValueArray into bytes.
func (w *WriteValueArray) Serialize() ([]byte, error) {
	b := make([]byte, w.Len())
	if err := w.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes WriteValueArray into bytes.
func (w *WriteValueArray) SerializeTo(b []byte) error {
	var offset = 4
	binary.LittleEndian.PutUint32(b[:4], uint32(w.ArraySize))

	for _, id := range w.WriteValues {
		if err := id.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += id.Len()
	}

	return nil
}

// Len returns the actual length in int.
func (w *WriteValueArray) Len() int {
	l := 4
	for _, id := range w.WriteValues {
		l += id.Len()
	}
	return l
}
