// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
)

// GetEndpointsResponse represents an GetEndpointsResponse.
type GetEndpointsResponse struct {
	TypeID         *datatypes.ExpandedNodeID
	ResponseHeader *ResponseHeader
	Endpoints      []*EndpointDescription
}

// NewGetEndpointsResponse creates an GetEndpointsResponse.
func NewGetEndpointsResponse(resHeader *ResponseHeader, endpoints ...*EndpointDescription) *GetEndpointsResponse {
	return &GetEndpointsResponse{
		TypeID:         datatypes.NewFourByteExpandedNodeID(0, ServiceTypeGetEndpointsResponse),
		ResponseHeader: resHeader,
		Endpoints:      endpoints,
	}
}

// ServiceType returns type of Service in uint16.
func (g *GetEndpointsResponse) ServiceType() uint16 {
	return ServiceTypeGetEndpointsResponse
}
