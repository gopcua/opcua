// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
)

// OpenSecureChannelResponse represents an OpenSecureChannelResponse.
type OpenSecureChannelResponse struct {
	TypeID                *datatypes.ExpandedNodeID
	ResponseHeader        *ResponseHeader
	ServerProtocolVersion uint32
	SecurityToken         *ChannelSecurityToken
	ServerNonce           []byte
}

// NewOpenSecureChannelResponse creates an OpenSecureChannelResponse.
func NewOpenSecureChannelResponse(resHeader *ResponseHeader, ver uint32, secToken *ChannelSecurityToken, nonce []byte) *OpenSecureChannelResponse {
	return &OpenSecureChannelResponse{
		TypeID:                datatypes.NewFourByteExpandedNodeID(0, ServiceTypeOpenSecureChannelResponse),
		ResponseHeader:        resHeader,
		ServerProtocolVersion: ver,
		SecurityToken:         secToken,
		ServerNonce:           nonce,
	}
}

// ServiceType returns type of Service in uint16.
func (o *OpenSecureChannelResponse) ServiceType() uint16 {
	return ServiceTypeOpenSecureChannelResponse
}
