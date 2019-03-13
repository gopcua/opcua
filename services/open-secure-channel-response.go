// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

// OpenSecureChannelResponse represents an OpenSecureChannelResponse.
type OpenSecureChannelResponse struct {
	ResponseHeader        *ResponseHeader
	ServerProtocolVersion uint32
	SecurityToken         *ChannelSecurityToken
	ServerNonce           []byte
}

// NewOpenSecureChannelResponse creates an OpenSecureChannelResponse.
func NewOpenSecureChannelResponse(resHeader *ResponseHeader, ver uint32, secToken *ChannelSecurityToken, nonce []byte) *OpenSecureChannelResponse {
	return &OpenSecureChannelResponse{
		ResponseHeader:        resHeader,
		ServerProtocolVersion: ver,
		SecurityToken:         secToken,
		ServerNonce:           nonce,
	}
}
