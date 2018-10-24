// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"fmt"

	"github.com/wmnsk/gopcua/datatypes"
)

// CreateSessionResponse represents a CreateSessionResponse.
//
// Specification: Part4, 5.6.2
type CreateSessionResponse struct {
	TypeID *datatypes.ExpandedNodeID
	*ResponseHeader
	SessionID                  *datatypes.NodeID
	AuthenticationToken        *datatypes.NodeID
	RevisedSessionTimeout      uint64
	ServerNonce                []byte
	ServerCertificate          []byte
	ServerEndpoints            []*EndpointDescription
	ServerSoftwareCertificates []*SignedSoftwareCertificate
	ServerSignature            *SignatureData
	MaxRequestMessageSize      uint32
}

// NewCreateSessionResponse creates a new NewCreateSessionResponse with the given parameters.
func NewCreateSessionResponse(resHeader *ResponseHeader, sessionID, authToken *datatypes.NodeID, timeout uint64, nonce, cert []byte, svrSignature *SignatureData, maxRespSize uint32, endpoints ...*EndpointDescription) *CreateSessionResponse {
	return &CreateSessionResponse{
		TypeID:                     datatypes.NewFourByteExpandedNodeID(0, ServiceTypeCreateSessionResponse),
		ResponseHeader:             resHeader,
		SessionID:                  sessionID,
		AuthenticationToken:        authToken,
		RevisedSessionTimeout:      timeout,
		ServerNonce:                nonce,
		ServerCertificate:          cert,
		ServerEndpoints:            endpoints,
		ServerSoftwareCertificates: nil,
		ServerSignature:            svrSignature,
		MaxRequestMessageSize:      maxRespSize,
	}
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
