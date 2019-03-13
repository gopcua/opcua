// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

// FindServersRequest returns the Servers known to a Server or Discovery Server. The behaviour of
// Discovery Servers is described in detail in Part 12.
//
// The Client may reduce the number of results returned by specifying filter criteria. A Discovery
// Server returns an empty list if no Servers match the criteria specified by the client. The filter
// criteria supported by this Service are described in 5.4.2.2.
//
// Specification: Part 4, 5.4.2
type FindServersRequest struct {
	RequestHeader *RequestHeader
	EndpointURL   string
	LocaleIDs     []string
	ServerURIs    []string
}

// NewFindServersRequest creates a new FindServersRequest.
func NewFindServersRequest(reqHeader *RequestHeader, url string, locales, serverURIs []string) *FindServersRequest {
	f := &FindServersRequest{
		RequestHeader: reqHeader,
		EndpointURL:   url,
		LocaleIDs:     locales,
		ServerURIs:    serverURIs,
	}
	return f
}
