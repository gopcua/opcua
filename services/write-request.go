// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
)

// WriteRequest is used to write values to one or more Attributes of one or more Nodes. For
// constructed Attribute values whose elements are indexed, such as an array, this Service allows
// Clients to write the entire set of indexed values as a composite, to write individual elements or to
// write ranges of elements of the composite.
//
// Specification: Part 4, 5.10.4
type WriteRequest struct {
	TypeID *datatypes.ExpandedNodeID
	*RequestHeader
	NodesToWrite *datatypes.WriteValueArray
}

// NewWriteRequest creates a new WriteRequest.
func NewWriteRequest(reqHeader *RequestHeader, nodes ...*datatypes.WriteValue) *WriteRequest {
	return &WriteRequest{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(
				0, ServiceTypeWriteRequest,
			),
			"", 0,
		),
		RequestHeader: reqHeader,
		NodesToWrite:  datatypes.NewWriteValueArray(nodes),
	}
}

// DecodeWriteRequest decodes given bytes into WriteRequest.
func DecodeWriteRequest(b []byte) (*WriteRequest, error) {
	r := &WriteRequest{}
	if err := r.DecodeFromBytes(b); err != nil {
		return nil, err
	}
	return r, nil
}

// DecodeFromBytes decodes given bytes into WriteRequest.
func (r *WriteRequest) DecodeFromBytes(b []byte) error {
	offset := 0
	r.TypeID = &datatypes.ExpandedNodeID{}
	if err := r.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += r.TypeID.Len()

	r.RequestHeader = &RequestHeader{}
	if err := r.RequestHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += r.RequestHeader.Len() - len(r.RequestHeader.Payload)

	r.NodesToWrite = &datatypes.WriteValueArray{}
	if err := r.NodesToWrite.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}

	return nil
}

// Serialize serializes WriteRequest into bytes.
func (r *WriteRequest) Serialize() ([]byte, error) {
	b := make([]byte, r.Len())
	if err := r.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes WriteRequest into bytes.
func (r *WriteRequest) SerializeTo(b []byte) error {
	offset := 0
	if err := r.TypeID.SerializeTo(b[offset:]); err != nil {
		return err
	}
	offset += r.TypeID.Len()

	if err := r.RequestHeader.SerializeTo(b[offset:]); err != nil {
		return err
	}
	offset += r.RequestHeader.Len()

	// nodes to read
	return r.NodesToWrite.SerializeTo(b[offset:])
}

// Len returns the actual length of WriteRequest.
func (r *WriteRequest) Len() int {
	l := 0
	if r.TypeID != nil {
		l += r.TypeID.Len()
	}

	if r.RequestHeader != nil {
		l += r.RequestHeader.Len()
	}

	if r.NodesToWrite != nil {
		l += r.NodesToWrite.Len()
	}

	return l
}

// ServiceType returns type of Service in uint16.
func (r *WriteRequest) ServiceType() uint16 {
	return ServiceTypeWriteRequest
}
