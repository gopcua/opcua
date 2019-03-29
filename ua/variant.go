// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"fmt"
	"time"
)

// These flags define the size and dimension of a Variant value.
const (
	VariantArrayDimensions = 0x40
	VariantArrayValues     = 0x80
)

// Variant is a union of the built-in types.
//
// Specification: Part 6, 5.2.2.16
type Variant struct {
	// EncodingMask contains the type and the array flags
	// bits 0:5: built-in type id 1-25
	// bit 6: array dimensions
	// bit 7: array values
	EncodingMask byte

	// ArrayLength is the number of elements in the array.
	// This field is only present if the 'array values'
	// flag is set.
	//
	// Multi-dimensional arrays are encoded as a one-dimensional array and this
	// field specifies the total number of elements. The original array can be
	// reconstructed from the dimensions that are encoded after the value
	// field.
	ArrayLength int32

	// ArrayDimensionsLength is the numer of dimensions.
	// This field is only present if the 'array dimensions' flag
	// is set.
	ArrayDimensionsLength int32

	// ArrayDimensions is the size for each dimension.
	// This field is only present if the 'array dimensions' flag
	// is set.
	ArrayDimensions []int32

	Value interface{}
}

func NewVariant(v interface{}) (*Variant, error) {
	va := &Variant{}
	if err := va.Set(v); err != nil {
		return nil, err
	}
	return va, nil
}

func MustVariant(v interface{}) *Variant {
	va, err := NewVariant(v)
	if err != nil {
		panic(err)
	}
	return va
}

// Type returns the type id of the value.
func (m *Variant) Type() TypeID {
	return TypeID(m.EncodingMask & 0x3f)
}

func (m *Variant) setType(t TypeID) {
	m.EncodingMask = byte(t & 0x3f)
}

func (m *Variant) Has(mask byte) bool {
	return m.EncodingMask&mask == mask
}

func (m *Variant) Decode(b []byte) (int, error) {
	buf := NewBuffer(b)

	m.EncodingMask = buf.ReadByte()

	elems := 1
	if m.Has(VariantArrayValues) {
		m.ArrayLength = buf.ReadInt32()
		elems = int(m.ArrayLength)
	}

	values := make([]interface{}, elems)
	for i := 0; i < elems; i++ {
		switch m.Type() {
		case TypeIDBoolean:
			values[i] = buf.ReadBool()
		case TypeIDSByte:
			values[i] = buf.ReadInt8()
		case TypeIDByte:
			values[i] = buf.ReadByte()
		case TypeIDInt16:
			values[i] = buf.ReadInt16()
		case TypeIDUint16:
			values[i] = buf.ReadUint16()
		case TypeIDInt32:
			values[i] = buf.ReadInt32()
		case TypeIDUint32:
			values[i] = buf.ReadUint32()
		case TypeIDInt64:
			values[i] = buf.ReadInt64()
		case TypeIDUint64:
			values[i] = buf.ReadUint64()
		case TypeIDFloat:
			values[i] = buf.ReadFloat32()
		case TypeIDDouble:
			values[i] = buf.ReadFloat64()
		case TypeIDString:
			values[i] = buf.ReadString()
		case TypeIDDateTime:
			values[i] = buf.ReadTime()
		case TypeIDGUID:
			v := new(GUID)
			buf.ReadStruct(v)
			values[i] = v
		case TypeIDByteString:
			values[i] = buf.ReadBytes()
		case TypeIDXMLElement:
			values[i] = XmlElement(buf.ReadString())
		case TypeIDNodeID:
			v := new(NodeID)
			buf.ReadStruct(v)
			values[i] = v
		case TypeIDExpandedNodeID:
			v := new(ExpandedNodeID)
			buf.ReadStruct(v)
			values[i] = v
		case TypeIDStatusCode:
			values[i] = StatusCode(buf.ReadUint32())
		case TypeIDQualifiedName:
			v := new(QualifiedName)
			buf.ReadStruct(v)
			values[i] = v
		case TypeIDLocalizedText:
			v := new(LocalizedText)
			buf.ReadStruct(v)
			values[i] = v
		case TypeIDExtensionObject:
			v := new(ExtensionObject)
			buf.ReadStruct(v)
			values[i] = v
		case TypeIDDataValue:
			v := new(DataValue)
			buf.ReadStruct(v)
			values[i] = v
		case TypeIDVariant:
			// todo(fs): limit recursion depth to 100
			v := new(Variant)
			buf.ReadStruct(v)
			values[i] = v
		case TypeIDDiagnosticInfo:
			// todo(fs): limit recursion depth to 100
			v := new(DiagnosticInfo)
			buf.ReadStruct(v)
			values[i] = v
		}
	}

	if m.Has(VariantArrayDimensions) {
		m.ArrayDimensionsLength = buf.ReadInt32()
		m.ArrayDimensions = make([]int32, m.ArrayDimensionsLength)
		for i := 0; i < int(m.ArrayDimensionsLength); i++ {
			m.ArrayDimensions[i] = buf.ReadInt32()
		}
	}

	m.Value = values
	if elems == 1 {
		m.Value = values[0]
	}

	return buf.Pos(), buf.Error()
}

func (m *Variant) Encode() ([]byte, error) {
	buf := NewBuffer(nil)

	buf.WriteByte(m.EncodingMask)

	if m.Has(VariantArrayValues) {
		buf.WriteInt32(m.ArrayLength)
	}

	switch v := m.Value.(type) {
	case bool:
		buf.WriteBool(v)
	case int8:
		buf.WriteInt8(v)
	case byte:
		buf.WriteByte(v)
	case int16:
		buf.WriteInt16(v)
	case uint16:
		buf.WriteUint16(v)
	case int32:
		buf.WriteInt32(v)
	case uint32:
		buf.WriteUint32(v)
	case int64:
		buf.WriteInt64(v)
	case uint64:
		buf.WriteUint64(v)
	case float32:
		buf.WriteFloat32(v)
	case float64:
		buf.WriteFloat64(v)
	case string:
		buf.WriteString(v)
	case time.Time:
		buf.WriteTime(v)
	case *GUID:
		buf.WriteStruct(v)
	case []byte:
		buf.WriteByteString(v)
	case XmlElement:
		buf.WriteString(string(v))
	case *NodeID:
		buf.WriteStruct(v)
	case *ExpandedNodeID:
		buf.WriteStruct(v)
	case StatusCode:
		buf.WriteUint32(uint32(v))
	case *QualifiedName:
		buf.WriteStruct(v)
	case *LocalizedText:
		buf.WriteStruct(v)
	case *ExtensionObject:
		buf.WriteStruct(v)
	case *DataValue:
		buf.WriteStruct(v)
	case *Variant:
		buf.WriteStruct(v)
	case *DiagnosticInfo:
		buf.WriteStruct(v)
	}

	if m.Has(VariantArrayDimensions) {
		buf.WriteInt32(m.ArrayDimensionsLength)
		for i := 0; i < int(m.ArrayDimensionsLength); i++ {
			buf.WriteInt32(m.ArrayDimensions[i])
		}
	}

	return buf.Bytes(), buf.Error()
}

func (m *Variant) Set(v interface{}) error {
	switch v.(type) {
	case bool:
		m.setType(TypeIDBoolean)
	case int8:
		m.setType(TypeIDSByte)
	case byte:
		m.setType(TypeIDByte)
	case int16:
		m.setType(TypeIDInt16)
	case uint16:
		m.setType(TypeIDUint16)
	case int32:
		m.setType(TypeIDInt32)
	case uint32:
		m.setType(TypeIDUint32)
	case int64:
		m.setType(TypeIDInt64)
	case uint64:
		m.setType(TypeIDUint64)
	case float32:
		m.setType(TypeIDFloat)
	case float64:
		m.setType(TypeIDDouble)
	case string:
		m.setType(TypeIDString)
	case time.Time:
		m.setType(TypeIDDateTime)
	case *GUID:
		m.setType(TypeIDGUID)
	case []byte:
		m.setType(TypeIDByteString)
	case XmlElement:
		m.setType(TypeIDXMLElement)
	case *NodeID:
		m.setType(TypeIDNodeID)
	case *ExpandedNodeID:
		m.setType(TypeIDExpandedNodeID)
	case StatusCode:
		m.setType(TypeIDStatusCode)
	case *QualifiedName:
		m.setType(TypeIDQualifiedName)
	case *LocalizedText:
		m.setType(TypeIDLocalizedText)
	case *ExtensionObject:
		m.setType(TypeIDExtensionObject)
	case *DataValue:
		m.setType(TypeIDDataValue)
	case *Variant:
		m.setType(TypeIDVariant)
	case *DiagnosticInfo:
		m.setType(TypeIDDiagnosticInfo)
	default:
		return fmt.Errorf("opcua: cannot set variant to %T", v)
	}
	m.Value = v
	return nil
}

// todo(fs): this should probably be StringValue or we need to handle all types
// todo(fs): and recursion
func (m *Variant) String() string {
	switch m.Type() {
	case TypeIDString:
		return m.Value.(string)
	case TypeIDLocalizedText:
		return m.Value.(*LocalizedText).Text
	case TypeIDQualifiedName:
		return m.Value.(*QualifiedName).Name
	default:
		return ""
		//return fmt.Sprintf("%v", m.Value)
	}
}

func (m *Variant) Bool() bool {
	switch m.Type() {
	case TypeIDBoolean:
		return m.Value.(bool)
	default:
		return false
	}
}

func (m *Variant) Float() float64 {
	switch m.Type() {
	case TypeIDFloat:
		return float64(m.Value.(float32))
	case TypeIDDouble:
		return m.Value.(float64)
	default:
		return 0
	}
}

func (m *Variant) Int() int64 {
	switch m.Type() {
	case TypeIDSByte:
		return int64(m.Value.(int8))
	case TypeIDInt16:
		return int64(m.Value.(int16))
	case TypeIDInt32:
		return int64(m.Value.(int32))
	case TypeIDInt64:
		return m.Value.(int64)
	default:
		return 0
	}
}

func (m *Variant) Uint() uint64 {
	switch m.Type() {
	case TypeIDByte:
		return uint64(m.Value.(byte))
	case TypeIDUint16:
		return uint64(m.Value.(uint16))
	case TypeIDUint32:
		return uint64(m.Value.(uint32))
	case TypeIDUint64:
		return m.Value.(uint64)
	default:
		return 0
	}
}

func (m *Variant) Time() time.Time {
	switch m.Type() {
	case TypeIDDateTime:
		return m.Value.(time.Time)
	default:
		return time.Time{}
	}
}
