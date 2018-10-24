// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
)

// ActivateSessionResponse is used by the Server to answer to the ActivateSessionRequest.
// Once used, a serverNonce cannot be used again. For that reason, the Server returns a new
// serverNonce each time the ActivateSession Service is called.
//
// When the ActivateSession Service is called for the first time then the Server shall reject the
// request if the SecureChannel is not same as the one associated with the CreateSession request.
// Subsequent calls to ActivateSession may be associated with different SecureChannels. If this is
// the case then the Server shall verify that the Certificate the Client used to create the new
// SecureChannel is the same as the Certificate used to create the original SecureChannel. In
// addition, the Server shall verify that the Client supplied a UserIdentityToken that is identical to the
// token currently associated with the Session. Once the Server accepts the new SecureChannel it
// shall reject requests sent via the old SecureChannel.
//
// Specification: Part 4, 5.6.3.2
type ActivateSessionResponse struct {
	TypeID *datatypes.ExpandedNodeID
	*ResponseHeader
	ServerNonce     *datatypes.ByteString
	Results         *datatypes.Uint32Array
	DiagnosticInfos *DiagnosticInfoArray
}

// NewActivateSessionResponse creates a new NewActivateSessionResponse.
func NewActivateSessionResponse(resHeader *ResponseHeader, nonce []byte, results []uint32, diags []*DiagnosticInfo) *ActivateSessionResponse {
	return &ActivateSessionResponse{
		TypeID:          datatypes.NewFourByteExpandedNodeID(0, ServiceTypeActivateSessionResponse),
		ResponseHeader:  resHeader,
		ServerNonce:     datatypes.NewByteString(nonce),
		Results:         datatypes.NewUint32Array(results),
		DiagnosticInfos: NewDiagnosticInfoArray(diags),
	}
}

// DecodeActivateSessionResponse decodes given bytes into ActivateSessionResponse.
func DecodeActivateSessionResponse(b []byte) (*ActivateSessionResponse, error) {
	a := &ActivateSessionResponse{}
	if err := a.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return a, nil
}

// DecodeFromBytes decodes given bytes into ActivateSessionResponse.
func (a *ActivateSessionResponse) DecodeFromBytes(b []byte) error {
	var offset = 0

	a.TypeID = &datatypes.ExpandedNodeID{}
	if err := a.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.TypeID.Len()

	a.ResponseHeader = &ResponseHeader{}
	if err := a.ResponseHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.ResponseHeader.Len() - len(a.ResponseHeader.Payload)

	a.ServerNonce = &datatypes.ByteString{}
	if err := a.ServerNonce.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.ServerNonce.Len()

	a.Results = &datatypes.Uint32Array{}
	if err := a.Results.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.Results.Len()

	a.DiagnosticInfos = &DiagnosticInfoArray{}
	return a.DiagnosticInfos.DecodeFromBytes(b[offset:])
}

// Serialize serializes ActivateSessionResponse into bytes.
func (a *ActivateSessionResponse) Serialize() ([]byte, error) {
	b := make([]byte, a.Len())
	if err := a.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes ActivateSessionResponse into bytes.
func (a *ActivateSessionResponse) SerializeTo(b []byte) error {
	var offset = 0
	if a.TypeID != nil {
		if err := a.TypeID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.TypeID.Len()
	}

	if a.ResponseHeader != nil {
		if err := a.ResponseHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.ResponseHeader.Len()
	}

	if a.ServerNonce != nil {
		if err := a.ServerNonce.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.ServerNonce.Len()
	}

	if a.Results != nil {
		if err := a.Results.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.Results.Len()
	}

	if a.DiagnosticInfos != nil {
		if err := a.DiagnosticInfos.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.DiagnosticInfos.Len()
	}

	return nil
}

// Len returns the actual length of ActivateSessionResponse.
func (a *ActivateSessionResponse) Len() int {
	length := 0

	if a.TypeID != nil {
		length += a.TypeID.Len()
	}

	if a.ResponseHeader != nil {
		length += a.ResponseHeader.Len()
	}

	if a.ServerNonce != nil {
		length += a.ServerNonce.Len()
	}

	if a.Results != nil {
		length += a.Results.Len()
	}

	if a.DiagnosticInfos != nil {
		length += a.DiagnosticInfos.Len()
	}

	return length
}

// ServiceType returns type of Service.
func (a *ActivateSessionResponse) ServiceType() uint16 {
	return ServiceTypeActivateSessionResponse
}
