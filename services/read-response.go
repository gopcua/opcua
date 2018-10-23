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
	TypeID *datatypes.ExpandedNodeID
	*ResponseHeader
	Results         *datatypes.DataValueArray
	DiagnosticInfos *DiagnosticInfoArray
}

// NewReadResponse creates a new ReadResponse.
func NewReadResponse(resHeader *ResponseHeader, diag []*DiagnosticInfo, results ...*datatypes.DataValue) *ReadResponse {
	return &ReadResponse{
		TypeID:          datatypes.NewFourByteExpandedNodeID(0, ServiceTypeReadResponse),
		ResponseHeader:  resHeader,
		Results:         datatypes.NewDataValueArray(results),
		DiagnosticInfos: NewDiagnosticInfoArray(diag),
	}
}

// DecodeReadResponse decodes given bytes into ReadResponse.
func DecodeReadResponse(b []byte) (*ReadResponse, error) {
	r := &ReadResponse{}
	if err := r.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return r, nil
}

// DecodeFromBytes decodes given bytes into ReadResponse.
func (r *ReadResponse) DecodeFromBytes(b []byte) error {
	offset := 0

	r.TypeID = &datatypes.ExpandedNodeID{}
	if err := r.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += r.TypeID.Len()

	r.ResponseHeader = &ResponseHeader{}
	if err := r.ResponseHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += r.ResponseHeader.Len() - len(r.ResponseHeader.Payload)

	r.Results = &datatypes.DataValueArray{}
	if err := r.Results.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += r.Results.Len()

	r.DiagnosticInfos = &DiagnosticInfoArray{}
	return r.DiagnosticInfos.DecodeFromBytes(b[offset:])
}

// Serialize serializes ReadResponse into bytes.
func (r *ReadResponse) Serialize() ([]byte, error) {
	b := make([]byte, r.Len())
	if err := r.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes ReadResponse into bytes.
func (r *ReadResponse) SerializeTo(b []byte) error {
	offset := 0
	if r.TypeID != nil {
		if err := r.TypeID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += r.TypeID.Len()
	}

	if r.ResponseHeader != nil {
		if err := r.ResponseHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += r.ResponseHeader.Len()
	}

	if r.Results != nil {
		if err := r.Results.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += r.Results.Len()
	}

	if r.DiagnosticInfos != nil {
		if err := r.DiagnosticInfos.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += r.DiagnosticInfos.Len()
	}

	return nil
}

// Len returns the actual length of ReadResponse.
func (r *ReadResponse) Len() int {
	length := 0

	if r.TypeID != nil {
		length += r.TypeID.Len()
	}

	if r.ResponseHeader != nil {
		length += r.ResponseHeader.Len()
	}

	if r.Results != nil {
		length += r.Results.Len()
	}

	if r.DiagnosticInfos != nil {
		length += r.DiagnosticInfos.Len()
	}

	return length
}

// ServiceType returns type of Service.
func (r *ReadResponse) ServiceType() uint16 {
	return id.ReadResponse_Encoding_DefaultBinary
}
