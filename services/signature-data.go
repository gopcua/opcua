// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"fmt"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/errors"
)

// SignatureData represents a SignatureData.
//
// Specification: Part 4, 7.32
type SignatureData struct {
	Algorithm *datatypes.String
	Signature *datatypes.ByteString
}

// NewSignatureData creates a new SignatureData.
func NewSignatureData(algorithm string, signature []byte) *SignatureData {
	return &SignatureData{
		Algorithm: datatypes.NewString(algorithm),
		Signature: datatypes.NewByteString(signature),
	}
}

// NewSignatureDataFrom generates SignatureData from certificate and nonce given.
//
// Specification: Part4, Table 15 and Table 17 (serverSignature and clientSignature).
func NewSignatureDataFrom(cert, nonce []byte) *SignatureData {
	// TODO: add calculation here.
	return &SignatureData{}
}

// DecodeSignatureData decodes given bytes into SignatureData.
func DecodeSignatureData(b []byte) (*SignatureData, error) {
	s := &SignatureData{}
	if err := s.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return s, nil
}

// DecodeFromBytes decodes given bytes into SignatureData.
func (s *SignatureData) DecodeFromBytes(b []byte) error {
	if len(b) < 8 {
		return errors.NewErrTooShortToDecode(s, "should be longer than 8 bytes.")
	}
	var offset = 0
	s.Algorithm = &datatypes.String{}
	if err := s.Algorithm.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += s.Algorithm.Len()

	s.Signature = &datatypes.ByteString{}
	return s.Signature.DecodeFromBytes(b[offset:])
}

// Serialize serializes SignatureData into bytes.
func (s *SignatureData) Serialize() ([]byte, error) {
	b := make([]byte, s.Len())
	if err := s.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes SignatureData into bytes.
func (s *SignatureData) SerializeTo(b []byte) error {
	var offset = 0
	if s.Algorithm != nil {
		if err := s.Algorithm.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += s.Algorithm.Len()
	}

	if s.Signature != nil {
		if err := s.Signature.SerializeTo(b[offset:]); err != nil {
			return err
		}
	}

	return nil
}

// Len returns the actual length of SignatureData in int.
func (s *SignatureData) Len() int {
	var l = 0
	if s.Algorithm != nil {
		l += s.Algorithm.Len()
	}
	if s.Signature != nil {
		l += s.Signature.Len()
	}

	return l
}

// datatypes.String returns SignatureData in string.
func (s *SignatureData) String() string {
	return fmt.Sprintf("%s, %x", s.Algorithm, s.Signature)
}
