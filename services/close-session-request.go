// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
)

// CloseSessionRequest represents an CloseSessionRequest.
// This Service is used to terminate a Session.
//
// Specification: Part 4, 5.6.4.2
type CloseSessionRequest struct {
	TypeID              *datatypes.ExpandedNodeID
	RequestHeader       *RequestHeader
	DeleteSubscriptions bool
}

// NewCloseSessionRequest creates a CloseSessionRequest.
func NewCloseSessionRequest(reqHeader *RequestHeader, deleteSubs bool) *CloseSessionRequest {
	return &CloseSessionRequest{
		TypeID:              datatypes.NewFourByteExpandedNodeID(0, ServiceTypeCloseSessionRequest),
		RequestHeader:       reqHeader,
		DeleteSubscriptions: deleteSubs,
	}
}

// ServiceType returns type of Service in uint16.
func (o *CloseSessionRequest) ServiceType() uint16 {
	return ServiceTypeCloseSessionRequest
}
