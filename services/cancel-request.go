// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/datatypes"
)

// CancelRequest is used to cancel outstanding Service requests. Successfully cancelled service
// requests shall respond with Bad_RequestCancelledByClient.
//
// Specification: Part4, 5.6.5
type CancelRequest struct {
	TypeID *datatypes.ExpandedNodeID
	*RequestHeader
	RequestHandle uint32
}

// NewCancelRequest creates a new CancelRequest.
func NewCancelRequest(reqHeader *RequestHeader, reqHandle uint32) *CancelRequest {
	return &CancelRequest{
		TypeID:        datatypes.NewFourByteExpandedNodeID(0, ServiceTypeCancelRequest),
		RequestHeader: reqHeader,
		RequestHandle: reqHandle,
	}
}

// DecodeCancelRequest decodes given bytes into CancelRequest.
func DecodeCancelRequest(b []byte) (*CancelRequest, error) {
	c := &CancelRequest{}
	if err := c.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return c, nil
}

// DecodeFromBytes decodes given bytes into CancelRequest.
func (c *CancelRequest) DecodeFromBytes(b []byte) error {
	var offset = 0
	c.TypeID = &datatypes.ExpandedNodeID{}
	if err := c.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.TypeID.Len()

	c.RequestHeader = &RequestHeader{}
	if err := c.RequestHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.RequestHeader.Len() - len(c.RequestHeader.Payload)

	c.RequestHandle = binary.LittleEndian.Uint32(b[offset : offset+4])
	return nil
}

// Serialize serializes CancelRequest into bytes.
func (c *CancelRequest) Serialize() ([]byte, error) {
	b := make([]byte, c.Len())
	if err := c.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes CancelRequest into bytes.
func (c *CancelRequest) SerializeTo(b []byte) error {
	var offset = 0
	if c.TypeID != nil {
		if err := c.TypeID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.TypeID.Len()
	}

	if c.RequestHeader != nil {
		if err := c.RequestHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.RequestHeader.Len()
	}

	binary.LittleEndian.PutUint32(b[offset:offset+4], c.RequestHandle)
	return nil
}

// Len returns the actual length of CancelRequest in int.
func (c *CancelRequest) Len() int {
	var l = 4
	if c.TypeID != nil {
		l += c.TypeID.Len()
	}
	if c.RequestHeader != nil {
		l += c.RequestHeader.Len()
	}

	return l
}

// String returns CancelRequest in string.
func (c *CancelRequest) String() string {
	return fmt.Sprintf("%v, %v, %d",
		c.TypeID,
		c.RequestHeader,
		c.RequestHandle,
	)
}

// ServiceType returns type of Service in uint16.
func (c *CancelRequest) ServiceType() uint16 {
	return ServiceTypeCancelRequest
}
