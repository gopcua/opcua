// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
)

// CloseSessionResponse represents an CloseSessionResponse.
// This Service is used to terminate a Session.
//
// Specification: Part 4, 5.6.4.2
type CloseSessionResponse struct {
	TypeID         *datatypes.ExpandedNodeID
	ResponseHeader *ResponseHeader
}

// NewCloseSessionResponse creates an CloseSessionResponse.
func NewCloseSessionResponse(resHeader *ResponseHeader) *CloseSessionResponse {
	return &CloseSessionResponse{
		TypeID:         datatypes.NewFourByteExpandedNodeID(0, ServiceTypeCloseSessionResponse),
		ResponseHeader: resHeader,
	}
}

// ServiceType returns type of Service in uint16.
func (o *CloseSessionResponse) ServiceType() uint16 {
	return ServiceTypeCloseSessionResponse
}
