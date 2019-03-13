// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

// GetEndpointsRequest represents an GetEndpointsRequest.
// This Service returns the Endpoints supported by a Server and all of the configuration information
// required to establish a SecureChannel and a Session.
//
// Specification: Part 4, 5.4.4.2
type GetEndpointsRequest struct {
	RequestHeader *RequestHeader
	EndpointURL   string
	LocaleIDs     []string
	ProfileURIs   []string
}

// NewGetEndpointsRequest creates an GetEndpointsRequest.
func NewGetEndpointsRequest(reqHeader *RequestHeader, endpoint string, localIDs, profileURIs []string) *GetEndpointsRequest {
	return &GetEndpointsRequest{
		RequestHeader: reqHeader,
		EndpointURL:   endpoint,
		LocaleIDs:     localIDs,
		ProfileURIs:   profileURIs,
	}
}
