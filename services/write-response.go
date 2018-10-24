// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"fmt"

	"github.com/wmnsk/gopcua/datatypes"
)

// WriteResponse is used to write values to one or more Attributes of one or more Nodes. For
// constructed Attribute values whose elements are indexed, such as an array, this Service allows
// Clients to write the entire set of indexed values as a composite, to write individual elements or to
// write ranges of elements of the composite.
//
// Specification: Part 4, 5.10.4
type WriteResponse struct {
	TypeID          *datatypes.ExpandedNodeID
	ResponseHeader  *ResponseHeader
	Results         []uint32
	DiagnosticInfos []*datatypes.DiagnosticInfo
}

// NewWriteResponse creates a new WriteResponse.
func NewWriteResponse(resHeader *ResponseHeader, diags []*datatypes.DiagnosticInfo, results ...uint32) *WriteResponse {
	return &WriteResponse{
		TypeID:          datatypes.NewFourByteExpandedNodeID(0, ServiceTypeWriteResponse),
		ResponseHeader:  resHeader,
		Results:         results,
		DiagnosticInfos: diags,
	}
}

// String returns WriteResponse in string.
func (w *WriteResponse) String() string {
	return fmt.Sprintf("%v, %v, %v, %v",
		w.TypeID,
		w.ResponseHeader,
		w.Results,
		w.DiagnosticInfos,
	)
}

// ServiceType returns type of Service in uint16.
func (w *WriteResponse) ServiceType() uint16 {
	return ServiceTypeWriteResponse
}
