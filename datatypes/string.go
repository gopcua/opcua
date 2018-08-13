// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/binary"

	"github.com/wmnsk/gopcua/errors"
)

// String represents the String type in OPC UA Specifications. This consists of the four-byte length field and variable length of contents.
type String struct {
	Length uint32
	Value  []byte
}

// DecodeFromBytes decodes given bytes into OPC UA String.
func (s *String) DecodeFromBytes(b []byte) error {
	if len(b) < 4 {
		return &errors.ErrTooShortToDecode{s, "should be longer than 4 bytes"}
	}

	s.Length = binary.LittleEndian.Uint32(b[:4])
	copy(s.Value, b[4:int(s.Length)])
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

// Put puts the string value in String and calcurate length.
func (s *String) Put(str string) {
	s.Value = []byte(str)
	s.Length = uint32(len(s.Value))
}
