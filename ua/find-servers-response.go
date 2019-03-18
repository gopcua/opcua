// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// FindServersResponse returns the Servers known to a Server or Discovery Server. The behaviour of
// Discovery Servers is described in detail in Part 12.
//
// Specification: Part4, 5.4.2
// type FindServersResponse struct {
// 	ResponseHeader *ResponseHeader
// 	Servers        []*ApplicationDescription
// }

// NewFindServersResponse creates an FindServersResponse.
func NewFindServersResponse(resHeader *ResponseHeader, servers ...*ApplicationDescription) *FindServersResponse {
	return &FindServersResponse{
		ResponseHeader: resHeader,
		Servers:        servers,
	}
}
