// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// CreateSessionRequest represents a CreateSessionRequest.
//
// Specification: Part4, 5.6.2
// type CreateSessionRequest struct {
// 	*RequestHeader
// 	ClientDescription       *ApplicationDescription
// 	ServerURI               string
// 	EndpointURL             string
// 	SessionName             string
// 	ClientNonce             []byte
// 	ClientCertificate       []byte
// 	RequestedSessionTimeout uint64
// 	MaxResponseMessageSize  uint32
// }

// NewCreateSessionRequest creates a new NewCreateSessionRequest with the given parameters.
func NewCreateSessionRequest(reqHeader *RequestHeader, appDescr *ApplicationDescription, serverURI, endpoint, sessionName string, nonce, cert []byte, timeout float64, maxRespSize uint32) *CreateSessionRequest {
	return &CreateSessionRequest{
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
