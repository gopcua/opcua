// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/errors"
)

// String represents the String type in OPC UA Specifications. This consists of the four-byte length field and variable length of contents.
type String struct {
	Length int32
	Value  []byte
}

// NewString creates a new String.
func NewString(str string) *String {
	if len(str) == 0 {
		s := &String{}
		s.Length = -1

		return s
	}

	s := &String{
		Value: []byte(str),
	}
	s.Length = int32(len(s.Value))

	return s
}

// DecodeString decodes given bytes into String.
func DecodeString(b []byte) (*String, error) {
	s := &String{}
	if err := s.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return s, nil
}

// DecodeFromBytes decodes given bytes into OPC UA String.
func (s *String) DecodeFromBytes(b []byte) error {
	if len(b) < 4 {
		return &errors.ErrTooShortToDecode{s, "should be longer than 4 bytes"}
	}

	s.Length = int32(binary.LittleEndian.Uint32(b[:4]))
	if s.Length <= 0 {
		return nil
	}
	s.Value = b[4 : 4+int(s.Length)]
	return nil
}

// Serialize serializes String into bytes.
func (s *String) Serialize() ([]byte, error) {
	b := make([]byte, s.Len())
	if err := s.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes String into bytes.
func (s *String) SerializeTo(b []byte) error {
	if len(b) < s.Len() {
		return &errors.ErrInvalidLength{s, "bytes should be longer"}
	}

	binary.LittleEndian.PutUint32(b[:4], uint32(s.Length))
	copy(b[4:s.Len()], s.Value)

	return nil
}

// Len returns the actual length of String in int.
func (s *String) Len() int {
	return 4 + len(s.Value)
}

// Get returns the value in Golang's built-in type string.
func (s *String) Get() string {
	return string(s.Value)
}

// Set sets the string value in String and calcurate length.
func (s *String) Set(str string) {
	s.Value = []byte(str)
	s.Length = int32(len(s.Value))
}

// String returns String in string.
func (s *String) String() string {
	return fmt.Sprintf("%d, %s", s.Length, s.Get())
}

// StringArray represents the StringArray.
type StringArray struct {
	ArraySize int32
	Strings   []*String
}

// NewStringArray creates a new StringArray from multiple strings.
func NewStringArray(strs []string) *StringArray {
	if strs == nil {
		s := &StringArray{
			ArraySize: 0,
		}
		return s
	}

	s := &StringArray{
		ArraySize: int32(len(strs)),
	}
	for _, ss := range strs {
		s.Strings = append(s.Strings, NewString(ss))
	}

	return s
}

// DecodeStringArray decodes given bytes into StringArray.
func DecodeStringArray(b []byte) (*StringArray, error) {
	s := &StringArray{}
	if err := s.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return s, nil
}

// DecodeFromBytes decodes given bytes into StringArray.
// TODO: add validation to avoid crash.
func (s *StringArray) DecodeFromBytes(b []byte) error {
	s.ArraySize = int32(binary.LittleEndian.Uint32(b[:4]))
	if s.ArraySize <= 0 {
		return nil
	}

	var offset = 4
	for i := 1; i <= int(s.ArraySize); i++ {
		str, err := DecodeString(b[offset:])
		if err != nil {
			return err
		}
		s.Strings = append(s.Strings, str)
		offset += str.Len()
	}

	return nil
}

// Serialize serializes StringArray into bytes.
func (s *StringArray) Serialize() ([]byte, error) {
	b := make([]byte, s.Len())
	if err := s.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes StringArray into bytes.
func (s *StringArray) SerializeTo(b []byte) error {
	var offset = 4
	binary.LittleEndian.PutUint32(b[:4], uint32(s.ArraySize))

	for _, ss := range s.Strings {
		if err := ss.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += ss.Len()
	}

	return nil
}

// Len returns the actual length in int.
func (s *StringArray) Len() int {
	l := 4
	for _, ss := range s.Strings {
		l += ss.Len()
	}

	return l
}
