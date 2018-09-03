// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/errors"
)

// CloseSecureChannelResponse represents an CloseSecureChannelResponse.
// This Service is used to terminate a SecureChannel.
//
// Specification: Part 4, 5.5.2.3
type CloseSecureChannelResponse struct {
	TypeID *datatypes.ExpandedNodeID
	*ResponseHeader
}

// NewCloseSecureChannelResponse creates an CloseSecureChannelResponse.
func NewCloseSecureChannelResponse(timestamp time.Time, handle, code uint32, diag *DiagnosticInfo, strs []string) *CloseSecureChannelResponse {
	o := &CloseSecureChannelResponse{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(
				0, ServiceTypeCloseSecureChannelResponse,
			),
			"", 0,
		),
		ResponseHeader: NewResponseHeader(
			timestamp, handle, code, diag, strs,
			NewAdditionalHeader(
				datatypes.NewExpandedNodeID(
					false, false,
					datatypes.NewTwoByteNodeID(0),
					"", 0,
				),
				0x00,
			), nil,
		),
	}

	return o
}

// DecodeCloseSecureChannelResponse decodes given bytes into CloseSecureChannelResponse.
func DecodeCloseSecureChannelResponse(b []byte) (*CloseSecureChannelResponse, error) {
	o := &CloseSecureChannelResponse{}
	if err := o.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return o, nil
}

// DecodeFromBytes decodes given bytes into CloseSecureChannelResponse.
func (o *CloseSecureChannelResponse) DecodeFromBytes(b []byte) error {
	if len(b) < 8 {
		return errors.NewErrTooShortToDecode(o, "should be longer than 16 bytes")
	}

	var offset = 0
	o.TypeID = &datatypes.ExpandedNodeID{}
	if err := o.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += o.TypeID.Len()

	o.ResponseHeader = &ResponseHeader{}
	return o.ResponseHeader.DecodeFromBytes(b[offset:])
}

// Serialize serializes CloseSecureChannelResponse into bytes.
func (o *CloseSecureChannelResponse) Serialize() ([]byte, error) {
	b := make([]byte, o.Len())
	if err := o.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes CloseSecureChannelResponse into bytes.
func (o *CloseSecureChannelResponse) SerializeTo(b []byte) error {
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

// Len returns the actual length of CloseSecureChannelResponse.
func (o *CloseSecureChannelResponse) Len() int {
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
func (o *CloseSecureChannelResponse) ServiceType() uint16 {
	return ServiceTypeCloseSecureChannelResponse
}
