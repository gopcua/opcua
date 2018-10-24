// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"fmt"

	"github.com/wmnsk/gopcua/datatypes"
)

// CancelResponse is used to cancel outstanding Service requests. Successfully cancelled service
// requests shall respond with Bad_RequestCancelledByClient.
//
// Specification: Part4, 5.6.5
type CancelResponse struct {
	TypeID         *datatypes.ExpandedNodeID
	ResponseHeader *ResponseHeader
	CancelCount    uint32
}

// NewCancelResponse creates a new CancelResponse.
func NewCancelResponse(resHeader *ResponseHeader, cancelCount uint32) *CancelResponse {
	return &CancelResponse{
		TypeID:         datatypes.NewFourByteExpandedNodeID(0, ServiceTypeCancelResponse),
		ResponseHeader: resHeader,
		CancelCount:    cancelCount,
	}
}

// String returns CancelResponse in string.
func (c *CancelResponse) String() string {
	return fmt.Sprintf("%v, %v, %d",
		c.TypeID,
		c.ResponseHeader,
		c.CancelCount,
	)
}

// ServiceType returns type of Service in uint16.
func (c *CancelResponse) ServiceType() uint16 {
	return ServiceTypeCancelResponse
}
