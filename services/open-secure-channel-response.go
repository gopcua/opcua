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

// OpenSecureChannelResponse represents an OpenSecureChannelResponse.
type OpenSecureChannelResponse struct {
	TypeID *datatypes.ExpandedNodeID
	*ResponseHeader
	ServerProtocolVersion uint32
	SecurityToken         *ChannelSecurityToken
	ServerNonce           *datatypes.ByteString
}

// NewOpenSecureChannelResponse creates an OpenSecureChannelResponse.
func NewOpenSecureChannelResponse(timestamp time.Time, handle, code uint32, diag *DiagnosticInfo, strs []string, ver uint32, secToken *ChannelSecurityToken, nonce []byte) *OpenSecureChannelResponse {
	o := &OpenSecureChannelResponse{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(
				0, ServiceTypeOpenSecureChannelResponse,
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
		ServerProtocolVersion: ver,
		SecurityToken:         secToken,
		ServerNonce:           datatypes.NewByteString(nonce),
	}

	return o
}

// DecodeOpenSecureChannelResponse decodes given bytes into OpenSecureChannelResponse.
func DecodeOpenSecureChannelResponse(b []byte) (*OpenSecureChannelResponse, error) {
	o := &OpenSecureChannelResponse{}
	if err := o.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return o, nil
}

// DecodeFromBytes decodes given bytes into OpenSecureChannelResponse.
func (o *OpenSecureChannelResponse) DecodeFromBytes(b []byte) error {
	if len(b) < 8 {
		return &errors.ErrTooShortToDecode{o, "should be longer than 16 bytes"}
	}

	var offset = 0
	o.TypeID = &datatypes.ExpandedNodeID{}
	if err := o.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += o.TypeID.Len()

	o.ResponseHeader = &ResponseHeader{}
	if err := o.ResponseHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += o.ResponseHeader.Len() - len(o.ResponseHeader.Payload)

	o.ServerProtocolVersion = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	o.SecurityToken = &ChannelSecurityToken{}
	if err := o.SecurityToken.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += o.SecurityToken.Len()

	o.ServerNonce = &datatypes.ByteString{}
	if err := o.ServerNonce.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += o.ServerNonce.Len()

	return nil
}

// Serialize serializes OpenSecureChannelResponse into bytes.
func (o *OpenSecureChannelResponse) Serialize() ([]byte, error) {
	b := make([]byte, o.Len())
	if err := o.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes OpenSecureChannelResponse into bytes.
func (o *OpenSecureChannelResponse) SerializeTo(b []byte) error {
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

	binary.LittleEndian.PutUint32(b[offset:offset+4], o.ServerProtocolVersion)
	offset += 4

	if o.SecurityToken != nil {
		if err := o.SecurityToken.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += o.SecurityToken.Len()
	}

	if o.ServerNonce != nil {
		if err := o.ServerNonce.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += o.ServerNonce.Len()
	}

	return nil
}

// Len returns the actual length of OpenSecureChannelResponse.
func (o *OpenSecureChannelResponse) Len() int {
	var l = 4
	if o.TypeID != nil {
		l += o.TypeID.Len()
	}
	if o.ResponseHeader != nil {
		l += (o.ResponseHeader.Len() - len(o.Payload))
	}
	if o.SecurityToken != nil {
		l += o.SecurityToken.Len()
	}
	if o.ServerNonce != nil {
		l += o.ServerNonce.Len()
	}

	return l
}

// ServiceType returns type of Service in uint16.
func (o *OpenSecureChannelResponse) ServiceType() uint16 {
	return ServiceTypeOpenSecureChannelResponse
}
