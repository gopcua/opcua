// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
)

// CloseSecureChannelRequest represents an CloseSecureChannelRequest.
// This Service is used to terminate a SecureChannel.
//
// Specification: Part 4, 5.5.3.2
type CloseSecureChannelRequest struct {
	TypeID          *datatypes.ExpandedNodeID
	RequestHeader   *RequestHeader
	SecureChannelID uint32
}

var CloseSecureChannelRequestID = datatypes.NewFourByteExpandedNodeID(0, ServiceTypeCloseSecureChannelRequest)

// NewCloseSecureChannelRequest creates an CloseSecureChannelRequest.
func NewCloseSecureChannelRequest(reqHeader *RequestHeader, chanID uint32) *CloseSecureChannelRequest {
	return &CloseSecureChannelRequest{
		TypeID:          CloseSecureChannelRequestID,
		RequestHeader:   reqHeader,
		SecureChannelID: chanID,
	}
}

// ServiceType returns type of Service in uint16.
func (o *CloseSecureChannelRequest) ServiceType() uint16 {
	return ServiceTypeCloseSecureChannelRequest
}
