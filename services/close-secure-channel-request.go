// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/binary"
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/errors"
)

// CloseSecureChannelRequest represents an CloseSecureChannelRequest.
// This Service is used to terminate a SecureChannel.
//
// Specification: Part 4, 5.5.2.3
type CloseSecureChannelRequest struct {
	TypeID *datatypes.ExpandedNodeID
	*RequestHeader
	SecureChannelID uint32
}

// NewCloseSecureChannelRequest creates an CloseSecureChannelRequest.
func NewCloseSecureChannelRequest(ts time.Time, authToken uint8, handle, diag, timeout uint32, auditID string, chanID uint32) *CloseSecureChannelRequest {
	o := &CloseSecureChannelRequest{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(
				0, ServiceTypeCloseSecureChannelRequest,
			),
			"", 0,
		),
		RequestHeader: NewRequestHeader(
			datatypes.NewTwoByteNodeID(authToken),
			ts,
			handle,
			diag,
			timeout,
			auditID,
			NewAdditionalHeader(
				datatypes.NewExpandedNodeID(
					false, false,
					datatypes.NewTwoByteNodeID(0),
					"", 0,
				),
				0x00,
			),
			nil,
		),
		SecureChannelID: chanID,
	}

	return o
}

// DecodeCloseSecureChannelRequest decodes given bytes into CloseSecureChannelRequest.
func DecodeCloseSecureChannelRequest(b []byte) (*CloseSecureChannelRequest, error) {
	o := &CloseSecureChannelRequest{}
	if err := o.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return o, nil
}

// DecodeFromBytes decodes given bytes into CloseSecureChannelRequest.
func (o *CloseSecureChannelRequest) DecodeFromBytes(b []byte) error {
	if len(b) < 16 {
		return errors.NewErrTooShortToDecode(o, "should be longer than 16 bytes")
	}

	var offset = 0
	o.TypeID = &datatypes.ExpandedNodeID{}
	if err := o.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += o.TypeID.Len()

	o.RequestHeader = &RequestHeader{}
	if err := o.RequestHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += o.RequestHeader.Len() - len(o.RequestHeader.Payload)

	o.SecureChannelID = binary.LittleEndian.Uint32(b[offset : offset+4])

	return nil
}

// Serialize serializes CloseSecureChannelRequest into bytes.
func (o *CloseSecureChannelRequest) Serialize() ([]byte, error) {
	b := make([]byte, o.Len())
	if err := o.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes CloseSecureChannelRequest into bytes.
func (o *CloseSecureChannelRequest) SerializeTo(b []byte) error {
	var offset = 0
	if o.TypeID != nil {
		if err := o.TypeID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += o.TypeID.Len()
	}

	if o.RequestHeader != nil {
		if err := o.RequestHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += o.RequestHeader.Len() - len(o.Payload)
	}

	binary.LittleEndian.PutUint32(b[offset:offset+4], o.SecureChannelID)

	return nil
}

// Len returns the actual length of CloseSecureChannelRequest.
func (o *CloseSecureChannelRequest) Len() int {
	var l = 4
	if o.TypeID != nil {
		l += o.TypeID.Len()
	}
	if o.RequestHeader != nil {
		l += (o.RequestHeader.Len() - len(o.Payload))
	}

	return l
}

// ServiceType returns type of Service in uint16.
func (o *CloseSecureChannelRequest) ServiceType() uint16 {
	return ServiceTypeCloseSecureChannelRequest
}
