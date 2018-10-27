// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/errors"
)

// SymmetricSecurityHeader represents a Symmetric Algorithm Security Header in OPC UA Secure Conversation.
type SymmetricSecurityHeader struct {
	TokenID uint32
	Payload []byte
}

// NewSymmetricSecurityHeader creates a new OPC UA Secure Conversation Symmetric Algorithm Security Header.
func NewSymmetricSecurityHeader(token uint32, payload []byte) *SymmetricSecurityHeader {
	return &SymmetricSecurityHeader{
		TokenID: token,
		Payload: payload,
	}
}

// DecodeSymmetricSecurityHeader decodes given bytes into OPC UA Secure Conversation Symmetric Algorithm Security Header.
func DecodeSymmetricSecurityHeader(b []byte) (*SymmetricSecurityHeader, error) {
	s := &SymmetricSecurityHeader{}
	if err := s.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return s, nil
}

// DecodeFromBytes decodes given bytes into OPC UA Secure Conversation Symmetric Algorithm Security Header.
// XXX - May be crashed when the length value and actual size is inconsistent.
func (s *SymmetricSecurityHeader) DecodeFromBytes(b []byte) error {
	l := len(b)
	if l < 4 {
		return errors.NewErrTooShortToDecode(s, "should be longer than 4 bytes")
	}
	s.TokenID = binary.LittleEndian.Uint32(b[:4])
	if len(b[4:]) > 0 {
		s.Payload = b[4:]
	}

	return nil
}

// Serialize serializes OPC UA Secure Conversation Symmetric Algorithm Security Header into bytes.
func (s *SymmetricSecurityHeader) Serialize() ([]byte, error) {
	b := make([]byte, int(s.Len()))
	if err := s.SerializeTo(b); err != nil {
		return nil, err
	}
	return b, nil
}

// SerializeTo serializes OPC UA Secure Conversation SymmetricSecurityHeader into given bytes.
// TODO: add error handling.
func (s *SymmetricSecurityHeader) SerializeTo(b []byte) error {
	binary.LittleEndian.PutUint32(b[:4], s.TokenID)
	copy(b[4:], s.Payload)

	return nil
}

// Len returns the actual length of SymmetricSecurityHeader in int.
func (s *SymmetricSecurityHeader) Len() int {
	return 4 + len(s.Payload)
}

// String returns Header in string.
func (s *SymmetricSecurityHeader) String() string {
	return fmt.Sprintf(
		"TokenID: %d, Payload: %x",
		s.TokenID,
		s.Payload,
	)
}
