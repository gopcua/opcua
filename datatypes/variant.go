// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"fmt"
	"time"

	"github.com/wmnsk/gopcua"
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

func (m *Variant) TypeID() byte {
	return m.EncodingMask & 0x3f
}

func (m *Variant) HasArrayDimensions() bool {
	return m.EncodingMask&0x40 == 0x40
}

func (m *Variant) HasArrayValues() bool {
	return m.EncodingMask&0x80 == 0x80
}

func (m *Variant) Decode(b []byte) (int, error) {
	buf := gopcua.NewBuffer(b)

	m.EncodingMask = buf.ReadByte()

	elems := 1
	if m.HasArrayValues() {
		m.ArrayLength = buf.ReadInt32()
		elems = int(m.ArrayLength)
	}

	values := make([]interface{}, elems)
	for i := 0; i < elems; i++ {
		mask := m.EncodingMask & 0x3f
		switch mask {
		case TypeBoolean:
			values[i] = buf.ReadBool()
		case TypeSByte:
			values[i] = buf.ReadInt8()
		case TypeByte:
			values[i] = buf.ReadByte()
		case TypeInt16:
			values[i] = buf.ReadInt16()
		case TypeUint16:
			values[i] = buf.ReadUint16()
		case TypeInt32:
			values[i] = buf.ReadInt32()
		case TypeUint32:
			values[i] = buf.ReadUint32()
		case TypeInt64:
			values[i] = buf.ReadInt64()
		case TypeUint64:
			values[i] = buf.ReadUint64()
		case TypeFloat:
			values[i] = buf.ReadFloat32()
		case TypeDouble:
			values[i] = buf.ReadFloat64()
		case TypeString:
			values[i] = buf.ReadString()
		case TypeDateTime:
			values[i] = buf.ReadTime()
		case TypeGuid:
			v := new(GUID)
			buf.ReadStruct(v)
			values[i] = v
		case TypeByteString:
			values[i] = buf.ReadBytes()
		case TypeXmlElement:
			values[i] = XmlElement(buf.ReadString())
		case TypeNodeId:
			v := new(NodeID)
			buf.ReadStruct(v)
			values[i] = v
		case TypeExpandedNodeId:
			v := new(ExpandedNodeID)
			buf.ReadStruct(v)
			values[i] = v
		case TypeStatusCode:
			values[i] = StatusCode(buf.ReadUint32())
		case TypeQualifiedName:
			v := new(QualifiedName)
			buf.ReadStruct(v)
			values[i] = v
		case TypeLocalizedText:
			v := new(LocalizedText)
			buf.ReadStruct(v)
			values[i] = v
		case TypeExtensionObject:
			v := new(ExtensionObject)
			buf.ReadStruct(v)
			values[i] = v
		case TypeDataValue:
			v := new(DataValue)
			buf.ReadStruct(v)
			values[i] = v
		case TypeVariant:
			// todo(fs): limit recursion depth to 100
			v := new(Variant)
			buf.ReadStruct(v)
			values[i] = v
		case TypeDiagnosticInfo:
			// todo(fs): limit recursion depth to 100
			v := new(DiagnosticInfo)
			buf.ReadStruct(v)
			values[i] = v
		}
	}

	if m.HasArrayDimensions() {
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
	buf := gopcua.NewBuffer(nil)

	buf.WriteByte(m.EncodingMask)

	if m.HasArrayValues() {
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

	if m.HasArrayDimensions() {
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
		m.EncodingMask = TypeBoolean
	case int8:
		m.EncodingMask = TypeSByte
	case byte:
		m.EncodingMask = TypeByte
	case int16:
		m.EncodingMask = TypeInt16
	case uint16:
		m.EncodingMask = TypeUint16
	case int32:
		m.EncodingMask = TypeInt32
	case uint32:
		m.EncodingMask = TypeUint32
	case int64:
		m.EncodingMask = TypeInt64
	case uint64:
		m.EncodingMask = TypeUint64
	case float32:
		m.EncodingMask = TypeFloat
	case float64:
		m.EncodingMask = TypeDouble
	case string:
		m.EncodingMask = TypeString
	case time.Time:
		m.EncodingMask = TypeDateTime
	case *GUID:
		m.EncodingMask = TypeGuid
	case []byte:
		m.EncodingMask = TypeByteString
	case XmlElement:
		m.EncodingMask = TypeXmlElement
	case *NodeID:
		m.EncodingMask = TypeNodeId
	case *ExpandedNodeID:
		m.EncodingMask = TypeExpandedNodeId
	case StatusCode:
		m.EncodingMask = TypeStatusCode
	case *QualifiedName:
		m.EncodingMask = TypeQualifiedName
	case *LocalizedText:
		m.EncodingMask = TypeLocalizedText
	case *ExtensionObject:
		m.EncodingMask = TypeExtensionObject
	case *DataValue:
		m.EncodingMask = TypeDataValue
	case *Variant:
		m.EncodingMask = TypeVariant
	case *DiagnosticInfo:
		m.EncodingMask = TypeDiagnosticInfo
	default:
		return fmt.Errorf("opcua: cannot set variant to %T", v)
	}
	m.Value = v
	return nil
}

func (m *Variant) String() string {
	return fmt.Sprintf("%v", m.Value)
}
