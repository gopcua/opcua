// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/gopcua/opcua/datatypes"
)

// CreateSessionResponse represents a CreateSessionResponse.
//
// Specification: Part4, 5.6.2
type CreateSessionResponse struct {
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
