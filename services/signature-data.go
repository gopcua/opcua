// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"fmt"
)

// SignatureData represents a SignatureData.
//
// Specification: Part 4, 7.32
type SignatureData struct {
	Algorithm string
	Signature []byte
}

// NewSignatureData creates a new SignatureData.
func NewSignatureData(algorithm string, signature []byte) *SignatureData {
	return &SignatureData{
		Algorithm: algorithm,
		Signature: signature,
	}
}

// NewSignatureDataFrom generates SignatureData from certificate and nonce given.
//
// Specification: Part4, Table 15 and Table 17 (serverSignature and clientSignature).
func NewSignatureDataFrom(cert, nonce []byte) *SignatureData {
	// TODO: add calculation here.
	if cert == nil && nonce == nil {
		return NewSignatureData("", nil)
	}
	return NewSignatureData("", nil)
}

// datatypes.String returns SignatureData in string.
func (s *SignatureData) String() string {
	return fmt.Sprintf("%s, %x", s.Algorithm, s.Signature)
}
