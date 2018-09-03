// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"time"

	"github.com/wmnsk/gopcua/datatypes"
)

// CloseSessionRequest represents an CloseSessionRequest.
// This Service is used to terminate a Session.
//
// Specification: Part 4, 5.6.4.2
type CloseSessionRequest struct {
	TypeID *datatypes.ExpandedNodeID
	*RequestHeader
	DeleteSubscriptions *datatypes.Boolean
}

// NewCloseSessionRequest creates a CloseSessionRequest.
func NewCloseSessionRequest(ts time.Time, authToken uint8, handle, diag, timeout uint32, auditID string, deleteSubs bool) *CloseSessionRequest {
	o := &CloseSessionRequest{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(
				0, ServiceTypeCloseSessionRequest,
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
		DeleteSubscriptions: datatypes.NewBoolean(deleteSubs),
	}

	return o
}

// DecodeCloseSessionRequest decodes given bytes into CloseSessionRequest.
func DecodeCloseSessionRequest(b []byte) (*CloseSessionRequest, error) {
	o := &CloseSessionRequest{}
	if err := o.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return o, nil
}

// DecodeFromBytes decodes given bytes into CloseSessionRequest.
func (o *CloseSessionRequest) DecodeFromBytes(b []byte) error {
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

	o.DeleteSubscriptions = &datatypes.Boolean{}
	return o.DeleteSubscriptions.DecodeFromBytes(b[offset:])
}

// Serialize serializes CloseSessionRequest into bytes.
func (o *CloseSessionRequest) Serialize() ([]byte, error) {
	b := make([]byte, o.Len())
	if err := o.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes CloseSessionRequest into bytes.
func (o *CloseSessionRequest) SerializeTo(b []byte) error {
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

	if o.DeleteSubscriptions != nil {
		if err := o.DeleteSubscriptions.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += o.DeleteSubscriptions.Len()
	}

	return nil
}

// Len returns the actual length of CloseSessionRequest.
func (o *CloseSessionRequest) Len() int {
	var l = 0
	if o.TypeID != nil {
		l += o.TypeID.Len()
	}
	if o.RequestHeader != nil {
		l += (o.RequestHeader.Len() - len(o.Payload))
	}
	if o.DeleteSubscriptions != nil {
		l += o.DeleteSubscriptions.Len()
	}

	return l
}

// ServiceType returns type of Service in uint16.
func (o *CloseSessionRequest) ServiceType() uint16 {
	return ServiceTypeCloseSessionRequest
}
