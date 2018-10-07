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

// CreateSessionRequest represents a CreateSessionRequest.
//
// Specification: Part4, 5.6.2
type CreateSessionRequest struct {
	TypeID *datatypes.ExpandedNodeID
	*RequestHeader
	ClientDescription       *ApplicationDescription
	ServerURI               *datatypes.String
	EndpointURL             *datatypes.String
	SessionName             *datatypes.String
	ClientNonce             *datatypes.ByteString
	ClientCertificate       *datatypes.ByteString
	RequestedSessionTimeout uint64
	MaxResponseMessageSize  uint32
}

// NewCreateSessionRequest creates a new NewCreateSessionRequest with the given parameters.
func NewCreateSessionRequest(reqHeader *RequestHeader, appDescr *ApplicationDescription, serverURI, endpoint, sessionName string, nonce, cert []byte, timeout uint64, maxRespSize uint32) *CreateSessionRequest {
	return &CreateSessionRequest{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(0, ServiceTypeCreateSessionRequest),
			"", 0,
		),
		RequestHeader:           reqHeader,
		ClientDescription:       appDescr,
		ServerURI:               datatypes.NewString(serverURI),
		EndpointURL:             datatypes.NewString(endpoint),
		SessionName:             datatypes.NewString(sessionName),
		ClientNonce:             datatypes.NewByteString(nonce),
		ClientCertificate:       datatypes.NewByteString(cert),
		RequestedSessionTimeout: timeout,
		MaxResponseMessageSize:  maxRespSize,
	}
}

// DecodeCreateSessionRequest decodes given bytes into CreateSessionRequest.
func DecodeCreateSessionRequest(b []byte) (*CreateSessionRequest, error) {
	c := &CreateSessionRequest{}
	if err := c.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return c, nil
}

// DecodeFromBytes decodes given bytes into CreateSessionRequest.
func (c *CreateSessionRequest) DecodeFromBytes(b []byte) error {
	if len(b) < 120 {
		return errors.NewErrTooShortToDecode(c, "should be longer than 120 bytes.")
	}

	var offset = 0
	c.TypeID = &datatypes.ExpandedNodeID{}
	if err := c.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.TypeID.Len()

	c.RequestHeader = &RequestHeader{}
	if err := c.RequestHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.RequestHeader.Len() - len(c.RequestHeader.Payload)

	c.ClientDescription = &ApplicationDescription{}
	if err := c.ClientDescription.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.ClientDescription.Len()

	c.ServerURI = &datatypes.String{}
	if err := c.ServerURI.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.ServerURI.Len()

	c.EndpointURL = &datatypes.String{}
	if err := c.EndpointURL.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.EndpointURL.Len()

	c.SessionName = &datatypes.String{}
	if err := c.SessionName.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.SessionName.Len()

	c.ClientNonce = &datatypes.ByteString{}
	if err := c.ClientNonce.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.ClientNonce.Len()

	c.ClientCertificate = &datatypes.ByteString{}
	if err := c.ClientCertificate.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.ClientCertificate.Len()

	c.RequestedSessionTimeout = binary.LittleEndian.Uint64(b[offset : offset+8])
	offset += 8

	c.MaxResponseMessageSize = binary.LittleEndian.Uint32(b[offset : offset+4])

	return nil
}

// Serialize serializes CreateSessionRequest into bytes.
func (c *CreateSessionRequest) Serialize() ([]byte, error) {
	b := make([]byte, c.Len())
	if err := c.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes CreateSessionRequest into bytes.
func (c *CreateSessionRequest) SerializeTo(b []byte) error {
	var offset = 0
	if c.TypeID != nil {
		if err := c.TypeID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.TypeID.Len()
	}

	if c.RequestHeader != nil {
		if err := c.RequestHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.RequestHeader.Len()
	}

	if c.ClientDescription != nil {
		if err := c.ClientDescription.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.ClientDescription.Len()
	}

	if c.ServerURI != nil {
		if err := c.ServerURI.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.ServerURI.Len()
	}

	if c.EndpointURL != nil {
		if err := c.EndpointURL.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.EndpointURL.Len()
	}

	if c.SessionName != nil {
		if err := c.SessionName.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.SessionName.Len()
	}

	if c.ClientNonce != nil {
		if err := c.ClientNonce.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.ClientNonce.Len()
	}

	if c.ClientCertificate != nil {
		if err := c.ClientCertificate.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.ClientCertificate.Len()
	}

	binary.LittleEndian.PutUint64(b[offset:offset+8], c.RequestedSessionTimeout)
	offset += 8

	binary.LittleEndian.PutUint32(b[offset:offset+4], c.MaxResponseMessageSize)

	return nil
}

// Len returns the actual length of CreateSessionRequest in int.
func (c *CreateSessionRequest) Len() int {
	var l = 12
	if c.TypeID != nil {
		l += c.TypeID.Len()
	}
	if c.RequestHeader != nil {
		l += c.RequestHeader.Len()
	}
	if c.ClientDescription != nil {
		l += c.ClientDescription.Len()
	}
	if c.ServerURI != nil {
		l += c.ServerURI.Len()
	}
	if c.EndpointURL != nil {
		l += c.EndpointURL.Len()
	}
	if c.SessionName != nil {
		l += c.SessionName.Len()
	}
	if c.ClientNonce != nil {
		l += c.ClientNonce.Len()
	}
	if c.ClientCertificate != nil {
		l += c.ClientCertificate.Len()
	}

	return l
}

// String returns CreateSessionRequest in string.
func (c *CreateSessionRequest) String() string {
	return fmt.Sprintf("%v, %v, %v, %v, %v, %v, %v, %d, %d",
		c.TypeID,
		c.RequestHeader,
		c.ClientDescription,
		c.ServerURI,
		c.EndpointURL,
		c.ClientNonce,
		c.ClientCertificate,
		c.RequestedSessionTimeout,
		c.MaxResponseMessageSize,
	)
}

// ServiceType returns type of Service in uint16.
func (c *CreateSessionRequest) ServiceType() uint16 {
	return ServiceTypeCreateSessionRequest
}
