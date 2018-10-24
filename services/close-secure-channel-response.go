// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
)

// CloseSecureChannelResponse represents an CloseSecureChannelResponse.
// This Service is used to terminate a SecureChannel.
//
// Specification: Part 4, 5.5.3.2
type CloseSecureChannelResponse struct {
	TypeID         *datatypes.ExpandedNodeID
	ResponseHeader *ResponseHeader
}

// NewCloseSecureChannelResponse creates an CloseSecureChannelResponse.
func NewCloseSecureChannelResponse(resHeader *ResponseHeader) *CloseSecureChannelResponse {
	return &CloseSecureChannelResponse{
		TypeID:         datatypes.NewFourByteExpandedNodeID(0, ServiceTypeCloseSecureChannelResponse),
		ResponseHeader: resHeader,
	}
}

// ServiceType returns type of Service in uint16.
func (o *CloseSecureChannelResponse) ServiceType() uint16 {
	return ServiceTypeCloseSecureChannelResponse
}
