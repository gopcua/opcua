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
	TypeID *datatypes.ExpandedNodeID
	*ResponseHeader
	Results         *datatypes.Uint32Array
	DiagnosticInfos *DiagnosticInfoArray
}

// NewWriteResponse creates a new WriteResponse.
func NewWriteResponse(resHeader *ResponseHeader, results []uint32, diags []*DiagnosticInfo) *WriteResponse {
	return &WriteResponse{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(0, ServiceTypeWriteResponse),
			"", 0,
		),
		ResponseHeader:  resHeader,
		Results:         datatypes.NewUint32Array(results),
		DiagnosticInfos: NewDiagnosticInfoArray(diags),
	}
}

// DecodeWriteResponse decodes given bytes into WriteResponse.
func DecodeWriteResponse(b []byte) (*WriteResponse, error) {
	w := &WriteResponse{}
	if err := w.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return w, nil
}

// DecodeFromBytes decodes given bytes into WriteResponse.
func (w *WriteResponse) DecodeFromBytes(b []byte) error {
	var offset = 0
	w.TypeID = &datatypes.ExpandedNodeID{}
	if err := w.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += w.TypeID.Len()

	w.ResponseHeader = &ResponseHeader{}
	if err := w.ResponseHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += w.ResponseHeader.Len() - len(w.ResponseHeader.Payload)

	w.Results = &datatypes.Uint32Array{}
	if err := w.Results.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += w.Results.Len()

	w.DiagnosticInfos = &DiagnosticInfoArray{}
	return w.DiagnosticInfos.DecodeFromBytes(b[offset:])
}

// Serialize serializes WriteResponse into bytes.
func (w *WriteResponse) Serialize() ([]byte, error) {
	b := make([]byte, w.Len())
	if err := w.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes WriteResponse into bytes.
func (w *WriteResponse) SerializeTo(b []byte) error {
	var offset = 0
	if w.TypeID != nil {
		if err := w.TypeID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += w.TypeID.Len()
	}

	if w.ResponseHeader != nil {
		if err := w.ResponseHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += w.ResponseHeader.Len()
	}

	if w.Results != nil {
		if err := w.Results.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += w.Results.Len()
	}

	if w.DiagnosticInfos != nil {
		return w.DiagnosticInfos.SerializeTo(b[offset:])
	}

	return nil
}

// Len returns the actual length of WriteResponse in int.
func (w *WriteResponse) Len() int {
	l := 0
	if w.TypeID != nil {
		l += w.TypeID.Len()
	}

	if w.ResponseHeader != nil {
		l += w.ResponseHeader.Len()
	}

	if w.Results != nil {
		l += w.Results.Len()
	}

	if w.DiagnosticInfos != nil {
		l += w.DiagnosticInfos.Len()
	}

	return l
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
