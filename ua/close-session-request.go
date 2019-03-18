// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// CloseSessionRequest represents an CloseSessionRequest.
// This Service is used to terminate a Session.
//
// Specification: Part 4, 5.6.4.2
// type CloseSessionRequest struct {
// 	RequestHeader       *RequestHeader
// 	DeleteSubscriptions bool
// }

// // NewCloseSessionRequest creates a CloseSessionRequest.
// func NewCloseSessionRequest(reqHeader *RequestHeader, deleteSubs bool) *CloseSessionRequest {
// 	return &CloseSessionRequest{
// 		RequestHeader:       reqHeader,
// 		DeleteSubscriptions: deleteSubs,
// 	}
// }
