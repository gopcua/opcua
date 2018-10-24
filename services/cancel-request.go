// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"fmt"

	"github.com/wmnsk/gopcua/datatypes"
)

// CancelRequest is used to cancel outstanding Service requests. Successfully cancelled service
// requests shall respond with Bad_RequestCancelledByClient.
//
// Specification: Part4, 5.6.5
type CancelRequest struct {
	TypeID *datatypes.ExpandedNodeID
	*RequestHeader
	RequestHandle uint32
}

// NewCancelRequest creates a new CancelRequest.
func NewCancelRequest(reqHeader *RequestHeader, reqHandle uint32) *CancelRequest {
	return &CancelRequest{
		TypeID:        datatypes.NewFourByteExpandedNodeID(0, ServiceTypeCancelRequest),
		RequestHeader: reqHeader,
		RequestHandle: reqHandle,
	}
}

// String returns CancelRequest in string.
func (c *CancelRequest) String() string {
	return fmt.Sprintf("%v, %v, %d",
		c.TypeID,
		c.RequestHeader,
		c.RequestHandle,
	)
}

// ServiceType returns type of Service in uint16.
func (c *CancelRequest) ServiceType() uint16 {
	return ServiceTypeCancelRequest
}
