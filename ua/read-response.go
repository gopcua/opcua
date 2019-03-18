// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// ReadResponse represents the response to a ReadRequest.
//
// Specification Part 4, 5.10.2.2
// type ReadResponse struct {
// 	ResponseHeader  *ResponseHeader
// 	Results         []*DataValue
// 	DiagnosticInfos []*DiagnosticInfo
// }

// NewReadResponse creates a new ReadResponse.
func NewReadResponse(resHeader *ResponseHeader, diag []*DiagnosticInfo, results ...*DataValue) *ReadResponse {
	return &ReadResponse{
		ResponseHeader:  resHeader,
		Results:         results,
		DiagnosticInfos: diag,
	}
}
