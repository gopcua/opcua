// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/errors"
)

// SequenceHeader represents a Sequence Header in OPC UA Secure Conversation.
type SequenceHeader struct {
	SequenceNumber uint32
	RequestID      uint32
	Payload        []byte
}

// NewSequenceHeader creates a new OPC UA Secure Conversation Sequence Header.
func NewSequenceHeader(seq, req uint32, payload []byte) *SequenceHeader {
	return &SequenceHeader{
		SequenceNumber: seq,
		RequestID:      req,
		Payload:        payload,
	}
}

// DecodeSequenceHeader decodes given bytes into OPC UA Secure Conversation Sequence Header.
func DecodeSequenceHeader(b []byte) (*SequenceHeader, error) {
	s := &SequenceHeader{}
	if err := s.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return s, nil
}

// DecodeFromBytes decodes given bytes into OPC UA Secure Conversation Sequence Header.
// XXX - May be crashed when the length value and actual size is inconsistent.
func (s *SequenceHeader) DecodeFromBytes(b []byte) error {
	l := len(b)
	if l < 8 {
		return errors.NewErrTooShortToDecode(s, "should be longer than 8 bytes")
	}
	s.SequenceNumber = binary.LittleEndian.Uint32(b[:4])
	s.RequestID = binary.LittleEndian.Uint32(b[4:8])
	if len(b[8:]) > 0 {
		s.Payload = b[8:]
	}

	return nil
}

// Serialize serializes OPC UA Secure Conversation Sequence Header into bytes.
func (s *SequenceHeader) Serialize() ([]byte, error) {
	b := make([]byte, int(s.Len()))
	if err := s.SerializeTo(b); err != nil {
		return nil, err
	}
	return b, nil
}

// SerializeTo serializes OPC UA Secure Conversation SequenceHeader into given bytes.
// TODO: add error handling.
func (s *SequenceHeader) SerializeTo(b []byte) error {
	binary.LittleEndian.PutUint32(b[:4], s.SequenceNumber)
	binary.LittleEndian.PutUint32(b[4:8], s.RequestID)
	copy(b[8:], s.Payload)

	return nil
}

// Len returns the actual length of SequenceHeader in int.
func (s *SequenceHeader) Len() int {
	return 8 + len(s.Payload)
}

// String returns Header in string.
func (s *SequenceHeader) String() string {
	return fmt.Sprintf(
		"SequenceNumber: %d, RequestID: %d, Payload: %x",
		s.SequenceNumber,
		s.RequestID,
		s.Payload,
	)
}
