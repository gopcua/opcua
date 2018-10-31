// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/binary"
	"math"

	"github.com/wmnsk/gopcua/id"
)

const f32qnan = 0xffc00000

// Float values shall be encoded with the appropriate IEEE-754 binary representation
// which has three basic components: the sign, the exponent, and the fraction.
//
// Specification: Part 6, 5.2.2.3
type Float struct {
	Value float32
}

// NewFloat creates a new Float.
func NewFloat(value float32) *Float {
	return &Float{
		Value: value,
	}
}

// DecodeFloat decodes given bytes into Float.
func DecodeFloat(b []byte) (*Float, error) {
	f := &Float{}
	if err := f.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return f, nil
}

// DecodeFromBytes decodes given bytes into OPC UA Float.
func (f *Float) DecodeFromBytes(b []byte) error {
	bits := binary.LittleEndian.Uint32(b)
	f.Value = math.Float32frombits(bits)
	return nil
}

// Serialize serializes Float into bytes.
func (f *Float) Serialize() ([]byte, error) {
	b := make([]byte, f.Len())
	if err := f.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes Float into bytes.
func (f *Float) SerializeTo(b []byte) error {
	// Part 6, 5.2.2.3 encode NaN as IEEE 754 silent NaN
	if math.IsNaN(float64(f.Value)) {
		binary.LittleEndian.PutUint32(b, f32qnan)
		return nil
	}

	bits := math.Float32bits(f.Value)
	binary.LittleEndian.PutUint32(b, bits)
	return nil
}

// Len returns the actual length of Float in int.
func (f *Float) Len() int {
	return 4
}

// DataType returns type of Data.
func (f *Float) DataType() uint16 {
	return id.Float
}
