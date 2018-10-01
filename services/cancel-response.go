// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/datatypes"
)

// CancelResponse is used to cancel outstanding Service requests. Successfully cancelled service
// requests shall respond with Bad_RequestCancelledByClient.
//
// Specification: Part4, 5.6.5
type CancelResponse struct {
	TypeID *datatypes.ExpandedNodeID
	*ResponseHeader
	CancelCount uint32
}

// NewCancelResponse creates a new CancelResponse.
func NewCancelResponse(resHeader *ResponseHeader, count uint32) *CancelResponse {
	return &CancelResponse{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(0, ServiceTypeCancelResponse),
			"", 0,
		),
		ResponseHeader: resHeader,
		CancelCount:    count,
	}
}

// DecodeCancelResponse decodes given bytes into CancelResponse.
func DecodeCancelResponse(b []byte) (*CancelResponse, error) {
	c := &CancelResponse{}
	if err := c.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return c, nil
}

// DecodeFromBytes decodes given bytes into CancelResponse.
func (c *CancelResponse) DecodeFromBytes(b []byte) error {
	var offset = 0
	c.TypeID = &datatypes.ExpandedNodeID{}
	if err := c.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.TypeID.Len()

	c.ResponseHeader = &ResponseHeader{}
	if err := c.ResponseHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.ResponseHeader.Len() - len(c.ResponseHeader.Payload)

	c.CancelCount = binary.LittleEndian.Uint32(b[offset : offset+4])
	return nil
}

// Serialize serializes CancelResponse into bytes.
func (c *CancelResponse) Serialize() ([]byte, error) {
	b := make([]byte, c.Len())
	if err := c.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes CancelResponse into bytes.
func (c *CancelResponse) SerializeTo(b []byte) error {
	var offset = 0
	if c.TypeID != nil {
		if err := c.TypeID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.TypeID.Len()
	}

	if c.ResponseHeader != nil {
		if err := c.ResponseHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.ResponseHeader.Len()
	}

	binary.LittleEndian.PutUint32(b[offset:offset+4], c.CancelCount)
	return nil
}

// Len returns the actual length of CancelResponse in int.
func (c *CancelResponse) Len() int {
	var l = 4
	if c.TypeID != nil {
		l += c.TypeID.Len()
	}
	if c.ResponseHeader != nil {
		l += c.ResponseHeader.Len()
	}

	return l
}

// String returns CancelResponse in string.
func (c *CancelResponse) String() string {
	return fmt.Sprintf("%v, %v, %d",
		c.TypeID,
		c.ResponseHeader,
		c.CancelCount,
	)
}

// ServiceType returns type of Service in uint16.
func (c *CancelResponse) ServiceType() uint16 {
	return ServiceTypeCancelResponse
}
