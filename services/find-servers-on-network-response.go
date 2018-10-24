// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"time"

	"github.com/wmnsk/gopcua/datatypes"
)

// FindServersOnNetworkResponse returns the Servers known to a Server or Discovery Server. The behaviour of
// Discovery Servers is described in detail in Part 12.
//
// Specification: Part4, 5.4.2
type FindServersOnNetworkResponse struct {
	TypeID               *datatypes.ExpandedNodeID
	ResponseHeader       *ResponseHeader
	LastCounterResetTime time.Time
	Servers              []*ServersOnNetwork
}

// NewFindServersOnNetworkResponse creates an FindServersOnNetworkResponse.
func NewFindServersOnNetworkResponse(resHeader *ResponseHeader, resetTime time.Time, servers ...*ServersOnNetwork) *FindServersOnNetworkResponse {
	return &FindServersOnNetworkResponse{
		TypeID:               datatypes.NewFourByteExpandedNodeID(0, ServiceTypeFindServersOnNetworkResponse),
		ResponseHeader:       resHeader,
		LastCounterResetTime: resetTime,
		Servers:              servers,
	}
}

// ServiceType returns type of Service in uint16.
func (f *FindServersOnNetworkResponse) ServiceType() uint16 {
	return ServiceTypeFindServersOnNetworkResponse
}
