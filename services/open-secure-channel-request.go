// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/binary"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/errors"
)

// OpenSecureChannelRequest represents an OpenSecureChannelRequest.
type OpenSecureChannelRequest struct {
	TypeID *datatypes.ExpandedNodeID
	*RequestHeader
	ClientProtocolVersion    uint32
	SecurityTokenRequestType uint32
	MessageSecurityMode      uint32
	ClientNonce              *datatypes.ByteString
	RequestedLifetime        uint32
}

// NewOpenSecureChannelRequest creates an OpenSecureChannelRequest.
func NewOpenSecureChannelRequest(hdr *RequestHeader, ver, tokenType, securityMode, lifetime uint32, nonce []byte) *OpenSecureChannelRequest {
	o := &OpenSecureChannelRequest{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(
				0, ServiceTypeOpenSecureChannelRequest,
			),
			"", 0,
		),
		RequestHeader:            hdr,
		ClientProtocolVersion:    ver,
		SecurityTokenRequestType: tokenType,
		MessageSecurityMode:      securityMode,
		ClientNonce:              datatypes.NewByteString(nonce),
		RequestedLifetime:        lifetime,
	}

	return o
}

// DecodeOpenSecureChannelRequest decodes given bytes into OpenSecureChannelRequest.
func DecodeOpenSecureChannelRequest(b []byte) (*OpenSecureChannelRequest, error) {
	o := &OpenSecureChannelRequest{}
	if err := o.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return o, nil
}

// DecodeFromBytes decodes given bytes into OpenSecureChannelRequest.
func (o *OpenSecureChannelRequest) DecodeFromBytes(b []byte) error {
	if len(b) < 16 {
		return &errors.ErrTooShortToDecode{o, "should be longer than 16 bytes"}
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

	o.ClientProtocolVersion = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4
	o.SecurityTokenRequestType = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4
	o.MessageSecurityMode = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	o.ClientNonce = &datatypes.ByteString{}
	if err := o.ClientNonce.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += o.ClientNonce.Len()

	o.RequestedLifetime = binary.LittleEndian.Uint32(b[offset : offset+4])

	return nil
}

// Serialize serializes OpenSecureChannelRequest into bytes.
func (o *OpenSecureChannelRequest) Serialize() ([]byte, error) {
	b := make([]byte, o.Len())
	if err := o.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes OpenSecureChannelRequest into bytes.
func (o *OpenSecureChannelRequest) SerializeTo(b []byte) error {
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

	binary.LittleEndian.PutUint32(b[offset:offset+4], o.ClientProtocolVersion)
	offset += 4
	binary.LittleEndian.PutUint32(b[offset:offset+4], o.SecurityTokenRequestType)
	offset += 4
	binary.LittleEndian.PutUint32(b[offset:offset+4], o.MessageSecurityMode)
	offset += 4

	if o.ClientNonce != nil {
		if err := o.ClientNonce.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += o.ClientNonce.Len()
	}

	binary.LittleEndian.PutUint32(b[offset:offset+4], o.RequestedLifetime)

	return nil
}

// Len returns the actual length of OpenSecureChannelRequest.
func (o *OpenSecureChannelRequest) Len() int {
	var l = 16
	if o.TypeID != nil {
		l += o.TypeID.Len()
	}
	if o.RequestHeader != nil {
		l += (o.RequestHeader.Len() - len(o.Payload))
	}
	if o.ClientNonce != nil {
		l += o.ClientNonce.Len()
	}

	return l
}

// ServiceType returns type of Service in uint16.
func (o *OpenSecureChannelRequest) ServiceType() uint16 {
	return ServiceTypeOpenSecureChannelRequest
}
