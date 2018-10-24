// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
)

// FindServersResponse returns the Servers known to a Server or Discovery Server. The behaviour of
// Discovery Servers is described in detail in Part 12.
//
// Specification: Part4, 5.4.2
type FindServersResponse struct {
	TypeID         *datatypes.ExpandedNodeID
	ResponseHeader *ResponseHeader
	Servers        []*ApplicationDescription
}

// NewFindServersResponse creates an FindServersResponse.
func NewFindServersResponse(resHeader *ResponseHeader, servers ...*ApplicationDescription) *FindServersResponse {
	return &FindServersResponse{
		TypeID:         datatypes.NewFourByteExpandedNodeID(0, ServiceTypeFindServersResponse),
		ResponseHeader: resHeader,
		Servers:        servers,
	}
}

// ServiceType returns type of Service in uint16.
func (f *FindServersResponse) ServiceType() uint16 {
	return ServiceTypeFindServersResponse
}
