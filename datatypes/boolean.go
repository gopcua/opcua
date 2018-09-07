// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import "github.com/wmnsk/gopcua/id"

// Boolean represents the datatype Boolean.
//
// Specification: Part 3, 8.8
type Boolean struct {
	Value uint8
}

// NewBoolean creates a new Boolean datatype.
// If given true, this returns the Boolean(=uint8) value 0x01. Otherwise the value is 0x00,
func NewBoolean(b bool) *Boolean {
	if b {
		return &Boolean{Value: 0x01}
	}
	return &Boolean{Value: 0x00}
}

// DecodeBoolean decodes given bytes into Boolean.
func DecodeBoolean(b []byte) (*Boolean, error) {
	bo := &Boolean{}
	if err := bo.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return bo, nil
}

// DecodeFromBytes decodes given bytes into Boolean.
func (bo *Boolean) DecodeFromBytes(b []byte) error {
	// TODO: Add validation.
	bo.Value = b[0]
	return nil
}

// Serialize serializes Boolean into bytes.
func (bo *Boolean) Serialize() ([]byte, error) {
	b := make([]byte, bo.Len())
	if err := bo.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes Boolean into bytes.
func (bo *Boolean) SerializeTo(b []byte) error {
	b[0] = bo.Value
	return nil
}

// Len returns the actual length of Boolean.
func (bo *Boolean) Len() int {
	return 1
}

// String returns the value of Boolean in string.
//
// The returned value would be "FALSE" if the Boolean is 0x00. Otherwise "TRUE".
func (bo *Boolean) String() string {
	if bo.Value == 0x00 {
		return "FALSE"
	}
	return "TRUE"
}

// ServiceType returns type of Service.
func (bo *Boolean) ServiceType() uint16 {
	return id.Boolean
}
