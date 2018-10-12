// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/errors"
)

// SignedSoftwareCertificate represents a SignedSoftwareCertificate.
//
// Specification: Part 4, 7.33
type SignedSoftwareCertificate struct {
	CertificateData *ByteString
	Signature       *ByteString
}

// NewSignedSoftwareCertificate creates a new SignedSoftwareCertificate.
func NewSignedSoftwareCertificate(cert, signature []byte) *SignedSoftwareCertificate {
	return &SignedSoftwareCertificate{
		CertificateData: NewByteString(cert),
		Signature:       NewByteString(signature),
	}
}

// DecodeSignedSoftwareCertificate decodes given bytes into SignedSoftwareCertificate.
func DecodeSignedSoftwareCertificate(b []byte) (*SignedSoftwareCertificate, error) {
	s := &SignedSoftwareCertificate{}
	if err := s.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return s, nil
}

// DecodeFromBytes decodes given bytes into SignedSoftwareCertificate.
func (s *SignedSoftwareCertificate) DecodeFromBytes(b []byte) error {
	if len(b) < 8 {
		return errors.NewErrTooShortToDecode(s, "should be longer than 8 bytes.")
	}
	var offset = 0
	s.CertificateData = &ByteString{}
	if err := s.CertificateData.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += s.CertificateData.Len()

	s.Signature = &ByteString{}
	return s.Signature.DecodeFromBytes(b[offset:])
}

// Serialize serializes SignedSoftwareCertificate into bytes.
func (s *SignedSoftwareCertificate) Serialize() ([]byte, error) {
	b := make([]byte, s.Len())
	if err := s.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes SignedSoftwareCertificate into bytes.
func (s *SignedSoftwareCertificate) SerializeTo(b []byte) error {
	var offset = 0
	if s.CertificateData != nil {
		if err := s.CertificateData.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += s.CertificateData.Len()
	}

	if s.Signature != nil {
		if err := s.Signature.SerializeTo(b[offset:]); err != nil {
			return err
		}
	}

	return nil
}

// Len returns the actual length of SignedSoftwareCertificate in int.
func (s *SignedSoftwareCertificate) Len() int {
	var l = 0
	if s.CertificateData != nil {
		l += s.CertificateData.Len()
	}
	if s.Signature != nil {
		l += s.Signature.Len()
	}

	return l
}

// ByteString returns SignedSoftwareCertificate in string.
func (s *SignedSoftwareCertificate) String() string {
	return fmt.Sprintf("%x, %x", s.CertificateData, s.Signature)
}

// SignedSoftwareCertificateArray represents the SignedSoftwareCertificateArray.
type SignedSoftwareCertificateArray struct {
	ArraySize    int32
	Certificates []*SignedSoftwareCertificate
}

// NewSignedSoftwareCertificateArray creates a new SignedSoftwareCertificateArray from multiple strings.
func NewSignedSoftwareCertificateArray(certs []*SignedSoftwareCertificate) *SignedSoftwareCertificateArray {
	if certs == nil {
		s := &SignedSoftwareCertificateArray{
			ArraySize: 0,
		}
		return s
	}

	s := &SignedSoftwareCertificateArray{
		ArraySize: int32(len(certs)),
	}
	s.Certificates = append(s.Certificates, certs...)

	return s
}

// DecodeSignedSoftwareCertificateArray decodes given bytes into SignedSoftwareCertificateArray.
func DecodeSignedSoftwareCertificateArray(b []byte) (*SignedSoftwareCertificateArray, error) {
	s := &SignedSoftwareCertificateArray{}
	if err := s.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return s, nil
}

// DecodeFromBytes decodes given bytes into SignedSoftwareCertificateArray.
// TODO: add validation to avoid crash.
func (s *SignedSoftwareCertificateArray) DecodeFromBytes(b []byte) error {
	s.ArraySize = int32(binary.LittleEndian.Uint32(b[:4]))
	if s.ArraySize <= 0 {
		return nil
	}

	var offset = 4
	for i := 1; i <= int(s.ArraySize); i++ {
		str, err := DecodeSignedSoftwareCertificate(b[offset:])
		if err != nil {
			return err
		}
		s.Certificates = append(s.Certificates, str)
		offset += str.Len()
	}

	return nil
}

// Serialize serializes SignedSoftwareCertificateArray into bytes.
func (s *SignedSoftwareCertificateArray) Serialize() ([]byte, error) {
	b := make([]byte, s.Len())
	if err := s.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes SignedSoftwareCertificateArray into bytes.
func (s *SignedSoftwareCertificateArray) SerializeTo(b []byte) error {
	var offset = 4
	binary.LittleEndian.PutUint32(b[:4], uint32(s.ArraySize))

	for _, cert := range s.Certificates {
		if err := cert.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += cert.Len()
	}

	return nil
}

// Len returns the actual length in int.
func (s *SignedSoftwareCertificateArray) Len() int {
	l := 4
	for _, ss := range s.Certificates {
		l += ss.Len()
	}

	return l
}
