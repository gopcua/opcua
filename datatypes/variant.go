package datatypes

import (
	"github.com/wmnsk/gopcua/errors"
	"github.com/wmnsk/gopcua/id"
)

// Service is an interface to handle any kind of OPC UA Services.
type Service interface {
	DecodeFromBytes([]byte) error
	Serialize() ([]byte, error)
	SerializeTo([]byte) error
	Len() int
	// String() string
	ServiceType() uint16
}

// Variant is a union of the built-in types.
//
// Specification: Part 6, 5.2.2.16
type Variant struct {
	EncodingMask          uint8
	ArrayLength           *int32
	Value                 Service
	ArrayDimensionsLength *int32
	ArrayDimensions       []*int32
}

// NewVariant creates a new Variant.
func NewVariant(service Service) *Variant {
	v := &Variant{
		EncodingMask: uint8(service.ServiceType()),
		Value:        service,
	}
	return v
}

// DecodeVariant decodes given bytes into Variant.
func DecodeVariant(b []byte) (*Variant, error) {
	v := &Variant{}
	if err := v.DecodeFromBytes(b); err != nil {
		return nil, err
	}
	return v, nil
}

// DecodeFromBytes decodes given bytes into Variant.
func (v *Variant) DecodeFromBytes(b []byte) error {
	v.EncodingMask = b[0]

	switch v.EncodingMask {
	case id.Boolean:
		v.Value = &Boolean{}
	case id.LocalizedText:
		v.Value = &LocalizedText{}
	case id.Float:
		v.Value = &Float{}
	default:
		return errors.NewErrInvalidType(v.EncodingMask, "decode", "got undefined type")
	}

	if err := v.Value.DecodeFromBytes(b[1:]); err != nil {
		return err
	}
	return nil
}

// Serialize serializes Variant into bytes.
func (v *Variant) Serialize() ([]byte, error) {
	b := make([]byte, v.Len())
	if err := v.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes Variant into bytes.
func (v *Variant) SerializeTo(b []byte) error {
	b[0] = v.EncodingMask

	offset := 1
	if v.Value != nil {
		if err := v.Value.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += v.Value.Len()
	}

	return nil
}

// Len returns the actual length of Variant in int.
func (v *Variant) Len() int {
	length := 1

	if v.Value != nil {
		length += v.Value.Len()
	}

	return length
}
