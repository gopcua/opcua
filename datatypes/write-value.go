// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import "github.com/gopcua/opcua/ua"

// WriteValue is a set of Node and Attribute to write.
//
// Specification: Part4, 5.10.4.2
type WriteValue struct {
	NodeID      *NodeID
	AttributeID uint32
	IndexRange  string
	Value       *DataValue
}

// NewWriteValue creates a new NewWriteValue.
func NewWriteValue(node *NodeID, attr uint32, idxRange string, value *DataValue) *WriteValue {
	return &WriteValue{
		NodeID:      node,
		AttributeID: attr,
		IndexRange:  idxRange,
		Value:       value,
	}
}

func (v *WriteValue) Decode(b []byte) (int, error) {
	buf := ua.NewBuffer(b)
	v.NodeID = new(NodeID)
	buf.ReadStruct(v.NodeID)
	v.AttributeID = buf.ReadUint32()
	v.IndexRange = buf.ReadString()
	v.Value = new(DataValue)
	buf.ReadStruct(v.Value)
	return buf.Pos(), buf.Error()
}

func (v *WriteValue) Encode() ([]byte, error) {
	buf := ua.NewBuffer(nil)
	buf.WriteStruct(v.NodeID)
	buf.WriteUint32(v.AttributeID)
	buf.WriteString(v.IndexRange)
	buf.WriteStruct(v.Value)
	return buf.Bytes(), buf.Error()
}
