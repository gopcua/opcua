// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// FindServersOnNetworkResponse returns the Servers known to a Server or Discovery Server. The behaviour of
// Discovery Servers is described in detail in Part 12.
//
// Specification: Part4, 5.4.2
// type FindServersOnNetworkResponse struct {
// 	ResponseHeader       *ResponseHeader
// 	LastCounterResetTime time.Time
// 	Servers              []*ServersOnNetwork
// }

// NewFindServersOnNetworkResponse creates an FindServersOnNetworkResponse.
// func NewFindServersOnNetworkResponse(resHeader *ResponseHeader, resetTime time.Time, servers ...*ServerOnNetwork) *FindServersOnNetworkResponse {
// 	return &FindServersOnNetworkResponse{
// 		ResponseHeader:       resHeader,
// 		LastCounterResetTime: resetTime,
// 		Servers:              servers,
// 	}
// }
