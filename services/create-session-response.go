// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/errors"
)

// CreateSessionResponse represents a CreateSessionResponse.
//
// Specification: Part4, 5.6.2
type CreateSessionResponse struct {
	TypeID *datatypes.ExpandedNodeID
	*ResponseHeader
	SessionID                  datatypes.NodeID
	AuthenticationToken        datatypes.NodeID
	RevisedSessionTimeout      uint64
	ServerNonce                *datatypes.ByteString
	ServerCertificate          *datatypes.ByteString
	ServerEndpoints            *datatypes.EndpointDescriptionArray
	ServerSoftwareCertificates *datatypes.SignedSoftwareCertificateArray
	ServerSignature            *datatypes.SignatureData
	MaxRequestMessageSize      uint32
}

// NewCreateSessionResponse creates a new NewCreateSessionResponse with the given parameters.
func NewCreateSessionResponse(resHeader *ResponseHeader, sessionID, authToken datatypes.NodeID, timeout uint64, nonce, cert []byte, svrSignature *datatypes.SignatureData, maxRespSize uint32, endpoints ...*datatypes.EndpointDescription) *CreateSessionResponse {
	return &CreateSessionResponse{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(0, ServiceTypeCreateSessionResponse),
			"", 0,
		),
		ResponseHeader:             resHeader,
		SessionID:                  sessionID,
		AuthenticationToken:        authToken,
		RevisedSessionTimeout:      timeout,
		ServerNonce:                datatypes.NewByteString(nonce),
		ServerCertificate:          datatypes.NewByteString(cert),
		ServerEndpoints:            datatypes.NewEndpointDescriptionArray(endpoints),
		ServerSoftwareCertificates: &datatypes.SignedSoftwareCertificateArray{ArraySize: 0},
		ServerSignature:            svrSignature,
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
		return errors.NewErrTooShortToDecode(c, "should be longer than 120 bytes.")
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

	sessionID, err := datatypes.DecodeNodeID(b[offset:])
	if err != nil {
		return err
	}
	c.SessionID = sessionID
	offset += c.SessionID.Len()

	authenticationToken, err := datatypes.DecodeNodeID(b[offset:])
	if err != nil {
		return err
	}
	c.AuthenticationToken = authenticationToken
	offset += c.AuthenticationToken.Len()

	c.RevisedSessionTimeout = binary.LittleEndian.Uint64(b[offset : offset+8])
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

	c.ServerEndpoints = &datatypes.EndpointDescriptionArray{}
	if err := c.ServerEndpoints.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.ServerEndpoints.Len()

	c.ServerSoftwareCertificates = &datatypes.SignedSoftwareCertificateArray{}
	if err := c.ServerSoftwareCertificates.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.ServerSoftwareCertificates.Len()

	c.ServerSignature = &datatypes.SignatureData{}
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

	binary.LittleEndian.PutUint64(b[offset:offset+8], c.RevisedSessionTimeout)
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
		c.RevisedSessionTimeout,
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
