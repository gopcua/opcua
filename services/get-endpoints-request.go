// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
)

// GetEndpointsRequest represents an GetEndpointsRequest.
// This Service returns the Endpoints supported by a Server and all of the configuration information
// required to establish a SecureChannel and a Session.
//
// Specification: Part 4, 5.4.4.2
type GetEndpointsRequest struct {
	TypeID        *datatypes.ExpandedNodeID
	RequestHeader *RequestHeader
	EndpointURL   string
	LocaleIDs     []string
	ProfileURIs   []string
}

// NewGetEndpointsRequest creates an GetEndpointsRequest.
func NewGetEndpointsRequest(reqHeader *RequestHeader, endpoint string, localIDs, profileURIs []string) *GetEndpointsRequest {
	return &GetEndpointsRequest{
		TypeID:        datatypes.NewFourByteExpandedNodeID(0, ServiceTypeGetEndpointsRequest),
		RequestHeader: reqHeader,
		EndpointURL:   endpoint,
		LocaleIDs:     localIDs,
		ProfileURIs:   profileURIs,
	}
}

// ServiceType returns type of Service in uint16.
func (g *GetEndpointsRequest) ServiceType() uint16 {
	return ServiceTypeGetEndpointsRequest
}
