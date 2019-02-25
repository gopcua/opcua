// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/id"
)

// ReadResponse represents the response to a ReadRequest.
//
// Specification Part 4, 5.10.2.2
type ReadResponse struct {
	TypeID          *datatypes.ExpandedNodeID
	ResponseHeader  *ResponseHeader
	Results         []*datatypes.DataValue
	DiagnosticInfos []*datatypes.DiagnosticInfo
}

// NewReadResponse creates a new ReadResponse.
func NewReadResponse(resHeader *ResponseHeader, diag []*datatypes.DiagnosticInfo, results ...*datatypes.DataValue) *ReadResponse {
	return &ReadResponse{
		TypeID:          datatypes.NewFourByteExpandedNodeID(0, ServiceTypeReadResponse),
		ResponseHeader:  resHeader,
		Results:         results,
		DiagnosticInfos: diag,
	}
}

// ServiceType returns type of Service.
func (r *ReadResponse) ServiceType() uint16 {
	return id.ReadResponse_Encoding_DefaultBinary
}

func (f *ReadResponse) RespHeader() *ResponseHeader {
	return f.ResponseHeader
}
