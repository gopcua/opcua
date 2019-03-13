// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
)

// ReadResponse represents the response to a ReadRequest.
//
// Specification Part 4, 5.10.2.2
type ReadResponse struct {
	ResponseHeader  *ResponseHeader
	Results         []*datatypes.DataValue
	DiagnosticInfos []*datatypes.DiagnosticInfo
}

// NewReadResponse creates a new ReadResponse.
func NewReadResponse(resHeader *ResponseHeader, diag []*datatypes.DiagnosticInfo, results ...*datatypes.DataValue) *ReadResponse {
	return &ReadResponse{
		ResponseHeader:  resHeader,
		Results:         results,
		DiagnosticInfos: diag,
	}
}
