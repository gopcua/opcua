// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/binary"

	"github.com/wmnsk/gopcua/errors"
	"github.com/wmnsk/gopcua/id"
)

// ExtensionObject is encoded as sequence of bytes prefixed by the NodeId of its DataTypeEncoding
// and the number of bytes encoded.
//
// Specification: Part 6, 5.2.2.15
type ExtensionObject struct {
	TypeID       *ExpandedNodeID
	EncodingMask byte
	Length       int32
	Value        ExtensionObjectValue
}

// NewExtensionObject creates a new ExtensionObject from the ExtensionObjectValue given.
func NewExtensionObject(mask uint8, extParam ExtensionObjectValue) *ExtensionObject {
	e := &ExtensionObject{
		TypeID: NewExpandedNodeID(
			false, false, NewFourByteNodeID(0, uint16(extParam.Type())), "", 0,
		),
		EncodingMask: mask,
		Value:        extParam,
	}
	e.SetLength()

	return e
}

// DecodeExtensionObject decodes given bytes into ExtensionObject.
func DecodeExtensionObject(b []byte) (*ExtensionObject, error) {
	e := &ExtensionObject{}
	if err := e.DecodeFromBytes(b); err != nil {
		return nil, err
	}
	return e, nil
}

// DecodeFromBytes decodes given bytes into ExtensionObject.
func (e *ExtensionObject) DecodeFromBytes(b []byte) error {
	// type id
	nodeID, err := DecodeExpandedNodeID(b)
	if err != nil {
		return err
	}
	e.TypeID = nodeID
	offset := e.TypeID.Len()

	// encoding mask
	e.EncodingMask = b[offset]
	offset++

	e.Length = int32(binary.LittleEndian.Uint32(b[offset : offset+4]))
	offset += 4

	// extension object parameter
	var id int
	node := e.TypeID.NodeID
	switch node.Type() {
	case TypeTwoByte, TypeFourByte, TypeNumeric:
		id = node.IntID()
	default:
		return errors.NewErrInvalidType(e.TypeID.NodeID, "decode", "NodeID should be TwoByte, FourByte or Numeric")
	}
	val, err := DecodeExtensionObjectValue(b[offset:], int(id))
	if err != nil {
		return err
	}
	e.Value = val

	return nil
}

// Serialize serializes ExtensionObject into bytes.
func (e *ExtensionObject) Serialize() ([]byte, error) {
	b := make([]byte, e.Len())
	if err := e.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes ExtensionObject into bytes.
func (e *ExtensionObject) SerializeTo(b []byte) error {
	offset := 0

	// type id
	if e.TypeID != nil {
		if err := e.TypeID.SerializeTo(b); err != nil {
			return err
		}
		offset += e.TypeID.Len()
	}

	// encoding mask
	b[offset] = e.EncodingMask
	offset++

	binary.LittleEndian.PutUint32(b[offset:offset+4], uint32(e.Length))
	offset += 4

	// extension object parameter
	if e.Value != nil {
		if err := e.Value.SerializeTo(b[offset:]); err != nil {
			return err
		}
	}

	return nil
}

// Len returns the actual length of ExtensionObject in int.
func (e *ExtensionObject) Len() int {
	// encoding mask byte + length
	length := 1 + 4

	if e.TypeID != nil {
		length += e.TypeID.Len()
	}

	if e.Value != nil {
		length += e.Value.Len()
	}

	return length
}

// SetLength sets the length of Value in Length field.
func (e *ExtensionObject) SetLength() {
	e.Length = int32(e.Value.Len())
}

// ExtensionObjectValue represents the value in ExtensionObject.
type ExtensionObjectValue interface {
	DecodeFromBytes([]byte) error
	SerializeTo(b []byte) error
	Len() int
	Type() int
}

// DecodeExtensionObjectValue decodes given bytes as an ExtensionObjectValue depending on the specified type.
//
// The type should be one defined in the DiscoveryConfiguration, UserIdentityToken, NodeAttributes,
// HistoryReadDetails, HistoryData, HistoryUpdateDetails, MonitoringFilterResult, FilterOperand.
func DecodeExtensionObjectValue(b []byte, typ int) (ExtensionObjectValue, error) {
	var e ExtensionObjectValue
	switch typ {
	case id.ElementOperand_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.LiteralOperand_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.AttributeOperand_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.SimpleAttributeOperand_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.MdnsDiscoveryConfiguration_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.DataChangeFilter_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.EventFilter_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.AggregateFilter_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.ObjectAttributes_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.VariableAttributes_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.MethodAttributes_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.ObjectTypeAttributes_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.VariableTypeAttributes_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.ReferenceTypeAttributes_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.DataTypeAttributes_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.ViewAttributes_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.GenericAttributes_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.DataChangeNotification_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.EventNotificationList_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.StatusChangeNotification_Encoding_DefaultBinary:
		return nil, errors.NewErrUnsupported(typ, "not implemented")
	case id.AnonymousIdentityToken_Encoding_DefaultBinary:
		e = &AnonymousIdentityToken{}
	case id.UserNameIdentityToken_Encoding_DefaultBinary:
		e = &UserNameIdentityToken{}
	case id.X509IdentityToken_Encoding_DefaultBinary:
		e = &X509IdentityToken{}
	case id.IssuedIdentityToken_Encoding_DefaultBinary:
		e = &IssuedIdentityToken{}
	default:
		return nil, errors.NewErrInvalidType(typ, "decode", "should be a type of ExtensionObjectValue")
	}

	if err := e.DecodeFromBytes(b); err != nil {
		return nil, err
	}
	return e, nil
}
