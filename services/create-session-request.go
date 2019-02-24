// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"fmt"

	"github.com/wmnsk/gopcua/datatypes"
)

// CreateSessionRequest represents a CreateSessionRequest.
//
// Specification: Part4, 5.6.2
type CreateSessionRequest struct {
	TypeID *datatypes.ExpandedNodeID
	*RequestHeader
	ClientDescription       *ApplicationDescription
	ServerURI               string
	EndpointURL             string
	SessionName             string
	ClientNonce             []byte
	ClientCertificate       []byte
	RequestedSessionTimeout uint64
	MaxResponseMessageSize  uint32
}

var CreateSessionRequestID = datatypes.NewFourByteExpandedNodeID(0, ServiceTypeCreateSessionRequest)

// NewCreateSessionRequest creates a new NewCreateSessionRequest with the given parameters.
func NewCreateSessionRequest(reqHeader *RequestHeader, appDescr *ApplicationDescription, serverURI, endpoint, sessionName string, nonce, cert []byte, timeout uint64, maxRespSize uint32) *CreateSessionRequest {
	return &CreateSessionRequest{
		TypeID:                  CreateSessionRequestID,
		RequestHeader:           reqHeader,
		ClientDescription:       appDescr,
		ServerURI:               serverURI,
		EndpointURL:             endpoint,
		SessionName:             sessionName,
		ClientNonce:             nonce,
		ClientCertificate:       cert,
		RequestedSessionTimeout: timeout,
		MaxResponseMessageSize:  maxRespSize,
	}
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
