// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// CancelResponse is used to cancel outstanding Service requests. Successfully cancelled service
// requests shall respond with Bad_RequestCancelledByClient.
//
// Specification: Part4, 5.6.5
// type CancelResponse struct {
// 	ResponseHeader *ResponseHeader
// 	CancelCount    uint32
// }

// NewCancelResponse creates a new CancelResponse.
// func NewCancelResponse(resHeader *ResponseHeader, cancelCount uint32) *CancelResponse {
// 	return &CancelResponse{
// 		ResponseHeader: resHeader,
// 		CancelCount:    cancelCount,
// 	}
// }
