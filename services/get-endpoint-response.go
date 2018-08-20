// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/errors"
)

// GetEndpointResponse represents an GetEndpointResponse.
type GetEndpointResponse struct {
	TypeID *datatypes.ExpandedNodeID
	*ResponseHeader
	Endpoints *EndpointDescriptionArray
}

// NewGetEndpointResponse creates an GetEndpointResponse.
func NewGetEndpointResponse(ts time.Time, handle, code uint32, diag *DiagnosticInfo, strs []string, endpoints ...*EndpointDescription) *GetEndpointResponse {
	g := &GetEndpointResponse{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(
				0, ServiceTypeGetEndpointResponse,
			),
			"", 0,
		),
		ResponseHeader: NewResponseHeader(
			ts,
			handle,
			code,
			diag,
			strs,
			NewAdditionalHeader(
				datatypes.NewExpandedNodeID(
					false, false,
					datatypes.NewTwoByteNodeID(0),
					"", 0,
				),
				0x00,
			),
			nil,
		),
		Endpoints: NewEndpointDesctiptionArray(endpoints),
	}

	return g
}

// DecodeGetEndpointResponse decodes given bytes into GetEndpointResponse.
func DecodeGetEndpointResponse(b []byte) (*GetEndpointResponse, error) {
	g := &GetEndpointResponse{}
	if err := g.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return g, nil
}

// DecodeFromBytes decodes given bytes into GetEndpointResponse.
func (g *GetEndpointResponse) DecodeFromBytes(b []byte) error {
	if len(b) < 16 {
		return &errors.ErrTooShortToDecode{g, "should be longer than 16 bytes"}
	}

	var offset = 0
	g.TypeID = &datatypes.ExpandedNodeID{}
	if err := g.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += g.TypeID.Len()

	g.ResponseHeader = &ResponseHeader{}
	if err := g.ResponseHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += g.ResponseHeader.Len() - len(g.ResponseHeader.Payload)

	g.Endpoints = &EndpointDescriptionArray{}
	if err := g.Endpoints.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += g.Endpoints.Len()

	return nil
}

// Serialize serializes GetEndpointResponse into bytes.
func (g *GetEndpointResponse) Serialize() ([]byte, error) {
	b := make([]byte, g.Len())
	if err := g.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes GetEndpointResponse into bytes.
func (g *GetEndpointResponse) SerializeTo(b []byte) error {
	var offset = 0
	if g.TypeID != nil {
		if err := g.TypeID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += g.TypeID.Len()
	}

	if g.ResponseHeader != nil {
		if err := g.ResponseHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += g.ResponseHeader.Len() - len(g.Payload)
	}

	if g.Endpoints != nil {
		if err := g.Endpoints.SerializeTo(b[offset:]); err != nil {
			return err
		}
	}

	return nil
}

// Len returns the actual length of GetEndpointResponse.
func (g *GetEndpointResponse) Len() int {
	var l = 0
	if g.TypeID != nil {
		l += g.TypeID.Len()
	}
	if g.ResponseHeader != nil {
		l += (g.ResponseHeader.Len() - len(g.Payload))
	}
	if g.Endpoints != nil {
		l += g.Endpoints.Len()
	}

	return l
}

// ServiceType returns type of Service in uint16.
func (g *GetEndpointResponse) ServiceType() uint16 {
	return ServiceTypeGetEndpointResponse
}
