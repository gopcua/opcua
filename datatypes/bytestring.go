// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/binary"

	"github.com/wmnsk/gopcua/errors"
)

// ByteString is encoded as sequence of bytes preceded by its length in bytes.
// The length is encoded as a 32-bit signed integer.
// If the length of the byte string is −1 then the byte string is ‘null’.
//
// Specification: Part 6, 5.2.2.7
type ByteString struct {
	Length int32
	Value  []byte
}

// NewByteString creates a new ByteString.
func NewByteString(b []byte) *ByteString {
	if len(b) == 0 {
		s := &ByteString{}
		s.Length = -1

		return s
	}

	s := &ByteString{
		Value: b,
	}
	s.Length = int32(len(s.Value))

	return s
}

// DecodeByteString decodes given bytes into ByteString.
func DecodeByteString(b []byte) (*ByteString, error) {
	s := &ByteString{}
	if err := s.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return s, nil
}

// DecodeFromBytes decodes given bytes into OPC UA ByteString.
func (s *ByteString) DecodeFromBytes(b []byte) error {
	if len(b) < 4 {
		return errors.NewErrTooShortToDecode(s, "should be longer than 4 bytes")
	}

	s.Length = int32(binary.LittleEndian.Uint32(b[:4]))
	if s.Length <= 0 {
		return nil
	}

	s.Value = b[4 : 4+s.Length]
	return nil
}

// Serialize serializes ByteString into bytes.
func (s *ByteString) Serialize() ([]byte, error) {
	b := make([]byte, s.Len())
	if err := s.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes ByteString into bytes.
func (s *ByteString) SerializeTo(b []byte) error {
	if len(b) < s.Len() {
		return errors.NewErrInvalidLength(s, "bytes should be longer")
	}

	binary.LittleEndian.PutUint32(b[:4], uint32(s.Length))
	copy(b[4:s.Len()], s.Value)

	return nil
}

// Len returns the actual length of ByteString in int.
func (s *ByteString) Len() int {
	return 4 + len(s.Value)
}

// Get returns the value in Golang's built-in type string.
func (s *ByteString) Get() []byte {
	return s.Value
}

// Set sets the string value in ByteString and calcurate length.
func (s *ByteString) Set(b []byte) {
	s.Value = b
	s.Length = int32(len(s.Value))
}
