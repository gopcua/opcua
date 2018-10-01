// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
)

// CloseSessionResponse represents an CloseSessionResponse.
// This Service is used to terminate a Session.
//
// Specification: Part 4, 5.6.4.2
type CloseSessionResponse struct {
	TypeID *datatypes.ExpandedNodeID
	*ResponseHeader
}

// NewCloseSessionResponse creates an CloseSessionResponse.
func NewCloseSessionResponse(resHeader *ResponseHeader) *CloseSessionResponse {
	return &CloseSessionResponse{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(
				0, ServiceTypeCloseSessionResponse,
			),
			"", 0,
		),
		ResponseHeader: resHeader,
	}
}

// DecodeCloseSessionResponse decodes given bytes into CloseSessionResponse.
func DecodeCloseSessionResponse(b []byte) (*CloseSessionResponse, error) {
	o := &CloseSessionResponse{}
	if err := o.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return o, nil
}

// DecodeFromBytes decodes given bytes into CloseSessionResponse.
func (o *CloseSessionResponse) DecodeFromBytes(b []byte) error {
	var offset = 0
	o.TypeID = &datatypes.ExpandedNodeID{}
	if err := o.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += o.TypeID.Len()

	o.ResponseHeader = &ResponseHeader{}
	return o.ResponseHeader.DecodeFromBytes(b[offset:])
}

// Serialize serializes CloseSessionResponse into bytes.
func (o *CloseSessionResponse) Serialize() ([]byte, error) {
	b := make([]byte, o.Len())
	if err := o.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes CloseSessionResponse into bytes.
func (o *CloseSessionResponse) SerializeTo(b []byte) error {
	var offset = 0
	if o.TypeID != nil {
		if err := o.TypeID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += o.TypeID.Len()
	}

	if o.ResponseHeader != nil {
		if err := o.ResponseHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += o.ResponseHeader.Len() - len(o.Payload)
	}

	return nil
}

// Len returns the actual length of CloseSessionResponse.
func (o *CloseSessionResponse) Len() int {
	var l = 0
	if o.TypeID != nil {
		l += o.TypeID.Len()
	}
	if o.ResponseHeader != nil {
		l += (o.ResponseHeader.Len() - len(o.Payload))
	}

	return l
}

// ServiceType returns type of Service in uint16.
func (o *CloseSessionResponse) ServiceType() uint16 {
	return ServiceTypeCloseSessionResponse
}
