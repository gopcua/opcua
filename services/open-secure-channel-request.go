// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

// SecurityTokenRequestType definitions.
//
// Specification: Part 4, 5.5.2.2
const (
	ReqTypeIssue uint32 = iota
	ReqTypeRenew
)

// MessageSecurityMode definitions.
//
// Specification: Part 4, 7.15
const (
	SecModeInvalid uint32 = iota
	SecModeNone
	SecModeSign
	SecModeSignAndEncrypt
)

// OpenSecureChannelRequest represents an OpenSecureChannelRequest.
// This Service is used to open or renew a SecureChannel that can be used to ensure Confidentiality
// and Integrity for Message exchange during a Session.
//
// Specification: Part 4, 5.5.2.2
type OpenSecureChannelRequest struct {
	RequestHeader            *RequestHeader
	ClientProtocolVersion    uint32
	SecurityTokenRequestType uint32
	MessageSecurityMode      uint32
	ClientNonce              []byte
	RequestedLifetime        uint32
}

// NewOpenSecureChannelRequest creates an OpenSecureChannelRequest.
func NewOpenSecureChannelRequest(reqHeader *RequestHeader, ver, tokenType, securityMode, lifetime uint32, nonce []byte) *OpenSecureChannelRequest {
	return &OpenSecureChannelRequest{
		RequestHeader:            reqHeader,
		ClientProtocolVersion:    ver,
		SecurityTokenRequestType: tokenType,
		MessageSecurityMode:      securityMode,
		ClientNonce:              nonce,
		RequestedLifetime:        lifetime,
	}
}
