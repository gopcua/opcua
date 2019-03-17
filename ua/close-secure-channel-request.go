// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// CloseSecureChannelRequest represents an CloseSecureChannelRequest.
// This Service is used to terminate a SecureChannel.
//
// Specification: Part 4, 5.5.3.2
type CloseSecureChannelRequest struct {
	RequestHeader   *RequestHeader
	SecureChannelID uint32
}

// NewCloseSecureChannelRequest creates an CloseSecureChannelRequest.
func NewCloseSecureChannelRequest(reqHeader *RequestHeader, chanID uint32) *CloseSecureChannelRequest {
	return &CloseSecureChannelRequest{
		RequestHeader:   reqHeader,
		SecureChannelID: chanID,
	}
}
