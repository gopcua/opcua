// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import "encoding/binary"

// Uint32Array represents the array of Uint32 type of data.
type Uint32Array struct {
	ArraySize int32
	Values    []uint32
}

// NewUint32Array creates a new NewUint32Array from multiple uint32 values.
func NewUint32Array(vals []uint32) *Uint32Array {
	if vals == nil {
		u := &Uint32Array{
			ArraySize: 0,
		}
		return u
	}

	u := &Uint32Array{
		ArraySize: int32(len(vals)),
	}
	for _, v := range vals {
		u.Values = append(u.Values, v)
	}

	return u
}

// DecodeUint32Array decodes given bytes into Uint32Array.
func DecodeUint32Array(b []byte) (*Uint32Array, error) {
	s := &Uint32Array{}
	if err := s.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return s, nil
}

// DecodeFromBytes decodes given bytes into Uint32Array.
// TODO: add validation to avoid crash.
func (u *Uint32Array) DecodeFromBytes(b []byte) error {
	u.ArraySize = int32(binary.LittleEndian.Uint32(b[:4]))
	if u.ArraySize <= 0 {
		return nil
	}

	var offset = 4
	for i := 1; i <= int(u.ArraySize); i++ {
		u.Values = append(u.Values, binary.LittleEndian.Uint32(b[offset:offset+4]))
		offset += 4
	}

	return nil
}

// Serialize serializes Uint32Array into bytes.
func (u *Uint32Array) Serialize() ([]byte, error) {
	b := make([]byte, u.Len())
	if err := u.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes Uint32Array into bytes.
func (u *Uint32Array) SerializeTo(b []byte) error {
	var offset = 4
	binary.LittleEndian.PutUint32(b[:4], uint32(u.ArraySize))

	for _, v := range u.Values {
		binary.LittleEndian.PutUint32(b[offset:offset+4], v)
		offset += 4
	}

	return nil
}

// Len returns the actual length in int.
func (u *Uint32Array) Len() int {
	return 4 + (len(u.Values) * 4)
}
