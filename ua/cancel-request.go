// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// CancelRequest is used to cancel outstanding Service requests. Successfully cancelled service
// requests shall respond with Bad_RequestCancelledByClient.
//
// Specification: Part4, 5.6.5
// type CancelRequest struct {
// 	*RequestHeader
// 	RequestHandle uint32
// }

// NewCancelRequest creates a new CancelRequest.
func NewCancelRequest(reqHeader *RequestHeader, reqHandle uint32) *CancelRequest {
	return &CancelRequest{
		RequestHeader: reqHeader,
		RequestHandle: reqHandle,
	}
}
