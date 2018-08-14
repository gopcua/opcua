// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import "github.com/wmnsk/gopcua/datatypes"

// AdditionalHeader represents the AdditionalHeader.
// TODO: add body handling.
type AdditionalHeader struct {
	TypeID       *datatypes.ExpandedNodeID
	EncodingMask uint8
}

// NewAdditionalHeader creates a new AdditionalHeader.
func NewAdditionalHeader(typeID *datatypes.ExpandedNodeID, mask uint8) *AdditionalHeader {
	return &AdditionalHeader{
		TypeID:       typeID,
		EncodingMask: mask,
	}
}

// DecodeAdditionalHeader decodes given bytes into AdditionalHeader.
func DecodeAdditionalHeader(b []byte) (*AdditionalHeader, error) {
	a := &AdditionalHeader{}
	if err := a.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return a, nil
}

// DecodeFromBytes decodes given bytes into AdditionalHeader.
func (a *AdditionalHeader) DecodeFromBytes(b []byte) error {
	if a.TypeID == nil {
		a.TypeID = &datatypes.ExpandedNodeID{}
	}
	if err := a.TypeID.DecodeFromBytes(b); err != nil {
		return err
	}
	a.EncodingMask = b[a.TypeID.Len()]

	return nil
}

// Serialize serializes AdditionalHeader into bytes.
func (a *AdditionalHeader) Serialize() ([]byte, error) {
	b := make([]byte, a.Len())
	if err := a.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes AdditionalHeader into bytes.
func (a *AdditionalHeader) SerializeTo(b []byte) error {
	var offset = 0
	if err := a.TypeID.SerializeTo(b); err != nil {
		return err
	}
	offset += a.TypeID.Len()

	b[offset] = a.EncodingMask

	return nil
}

// Len returns the actual length of AdditionalHeader in int.
func (a *AdditionalHeader) Len() int {
	return 1 + a.TypeID.Len()
}

// HasBinaryBody checks if an AdditionalHeader has binary body above.
func (a *AdditionalHeader) HasBinaryBody() bool {
	return a.EncodingMask&0x1 == 1
}

// HasXMLBody checks if an AdditionalHeader has binary body above.
func (a *AdditionalHeader) HasXMLBody() bool {
	return (a.EncodingMask>>1)&0x1 == 1
}
