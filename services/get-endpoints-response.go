// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

// GetEndpointsResponse represents an GetEndpointsResponse.
type GetEndpointsResponse struct {
	ResponseHeader *ResponseHeader
	Endpoints      []*EndpointDescription
}

// NewGetEndpointsResponse creates an GetEndpointsResponse.
func NewGetEndpointsResponse(resHeader *ResponseHeader, endpoints ...*EndpointDescription) *GetEndpointsResponse {
	return &GetEndpointsResponse{
		ResponseHeader: resHeader,
		Endpoints:      endpoints,
	}
}
