// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/errors"
)

// CreateSessionResponse represents a CreateSessionResponse.
//
// Specification: Part4, 5.6.2
type CreateSessionResponse struct {
	TypeID *datatypes.ExpandedNodeID
	*ResponseHeader
	SessionID                  *datatypes.NumericNodeID
	AuthenticationToken        *datatypes.FourByteNodeID
	ReviesedSessionTimeout     uint64
	ServerNonce                *datatypes.ByteString
	ServerCertificate          *datatypes.ByteString
	ServerEndpoints            *EndpointDescriptionArray
	ServerSoftwareCertificates *SignedSoftwareCertificateArray
	ServerSignature            *SignatureData
	MaxRequestMessageSize      uint32
}

// NewCreateSessionResponse creates a new NewCreateSessionResponse with the given parameters.
func NewCreateSessionResponse(time time.Time, result uint32, diag *DiagnosticInfo, sessionID uint32, authToken uint16, timeout uint64, nonce, cert []byte, endpoints []*EndpointDescription, alg string, sign []byte, maxRespSize uint32) *CreateSessionResponse {
	return &CreateSessionResponse{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(0, ServiceTypeCreateSessionResponse),
			"", 0,
		),
		ResponseHeader: NewResponseHeader(
			time, 1, result, diag, []string{},
			NewNullAdditionalHeader(),
			nil,
		),
		SessionID:                  datatypes.NewNumericNodeID(0, sessionID),
		AuthenticationToken:        datatypes.NewFourByteNodeID(0, authToken),
		ReviesedSessionTimeout:     timeout,
		ServerNonce:                datatypes.NewByteString(nonce),
		ServerCertificate:          datatypes.NewByteString(cert),
		ServerEndpoints:            NewEndpointDescriptionArray(endpoints),
		ServerSoftwareCertificates: NewSignedSoftwareCertificateArray(nil),
		ServerSignature:            NewSignatureData(alg, sign),
		MaxRequestMessageSize:      maxRespSize,
	}
}

// DecodeCreateSessionResponse decodes given bytes into CreateSessionResponse.
func DecodeCreateSessionResponse(b []byte) (*CreateSessionResponse, error) {
	c := &CreateSessionResponse{}
	if err := c.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return c, nil
}

// DecodeFromBytes decodes given bytes into CreateSessionResponse.
func (c *CreateSessionResponse) DecodeFromBytes(b []byte) error {
	if len(b) < 120 {
		return &errors.ErrTooShortToDecode{c, "should be longer than 120 bytes."}
	}

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

	c.SessionID = &datatypes.NumericNodeID{}
	if err := c.SessionID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.SessionID.Len()

	c.AuthenticationToken = &datatypes.FourByteNodeID{}
	if err := c.AuthenticationToken.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.AuthenticationToken.Len()

	c.ReviesedSessionTimeout = binary.LittleEndian.Uint64(b[offset : offset+8])
	offset += 8

	c.ServerNonce = &datatypes.ByteString{}
	if err := c.ServerNonce.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.ServerNonce.Len()

	c.ServerCertificate = &datatypes.ByteString{}
	if err := c.ServerCertificate.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.ServerCertificate.Len()

	c.ServerEndpoints = &EndpointDescriptionArray{}
	if err := c.ServerEndpoints.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.ServerEndpoints.Len()

	c.ServerSoftwareCertificates = &SignedSoftwareCertificateArray{}
	if err := c.ServerSoftwareCertificates.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.ServerSoftwareCertificates.Len()

	c.ServerSignature = &SignatureData{}
	if err := c.ServerSignature.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.ServerSignature.Len()

	c.MaxRequestMessageSize = binary.LittleEndian.Uint32(b[offset : offset+4])

	return nil
}

// Serialize serializes CreateSessionResponse into bytes.
func (c *CreateSessionResponse) Serialize() ([]byte, error) {
	b := make([]byte, c.Len())
	if err := c.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes CreateSessionResponse into bytes.
func (c *CreateSessionResponse) SerializeTo(b []byte) error {
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

	if c.SessionID != nil {
		if err := c.SessionID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.SessionID.Len()
	}

	if c.AuthenticationToken != nil {
		if err := c.AuthenticationToken.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.AuthenticationToken.Len()
	}

	binary.LittleEndian.PutUint64(b[offset:offset+8], c.ReviesedSessionTimeout)
	offset += 8

	if c.ServerNonce != nil {
		if err := c.ServerNonce.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.ServerNonce.Len()
	}

	if c.ServerCertificate != nil {
		if err := c.ServerCertificate.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.ServerCertificate.Len()
	}

	if c.ServerEndpoints != nil {
		if err := c.ServerEndpoints.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.ServerEndpoints.Len()
	}

	if c.ServerSoftwareCertificates != nil {
		if err := c.ServerSoftwareCertificates.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.ServerSoftwareCertificates.Len()
	}

	if c.ServerSignature != nil {
		if err := c.ServerSignature.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.ServerSignature.Len()
	}

	binary.LittleEndian.PutUint32(b[offset:offset+4], c.MaxRequestMessageSize)

	return nil
}

// Len returns the actual length of CreateSessionResponse in int.
func (c *CreateSessionResponse) Len() int {
	var l = 12
	if c.TypeID != nil {
		l += c.TypeID.Len()
	}
	if c.ResponseHeader != nil {
		l += c.ResponseHeader.Len()
	}
	if c.SessionID != nil {
		l += c.SessionID.Len()
	}
	if c.AuthenticationToken != nil {
		l += c.AuthenticationToken.Len()
	}
	if c.ServerNonce != nil {
		l += c.ServerNonce.Len()
	}
	if c.ServerCertificate != nil {
		l += c.ServerCertificate.Len()
	}
	if c.ServerEndpoints != nil {
		l += c.ServerEndpoints.Len()
	}
	if c.ServerSoftwareCertificates != nil {
		l += c.ServerSoftwareCertificates.Len()
	}
	if c.ServerSignature != nil {
		l += c.ServerSignature.Len()
	}

	return l
}

// String returns CreateSessionResponse in string.
func (c *CreateSessionResponse) String() string {
	return fmt.Sprintf("%v, %v, %v, %v, %d, %v, %v, %v, %v, %v, %d",
		c.TypeID,
		c.ResponseHeader,
		c.SessionID,
		c.AuthenticationToken,
		c.ReviesedSessionTimeout,
		c.ServerNonce,
		c.ServerCertificate,
		c.ServerEndpoints,
		c.ServerSoftwareCertificates,
		c.ServerSignature,
		c.MaxRequestMessageSize,
	)
}

// ServiceType returns type of Service in uint16.
func (c *CreateSessionResponse) ServiceType() uint16 {
	return ServiceTypeCreateSessionResponse
}
