// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/errors"
)

// GetEndpointRequest represents an GetEndpointRequest.
type GetEndpointRequest struct {
	TypeID *datatypes.ExpandedNodeID
	*RequestHeader
	EndpointURL *datatypes.String
	LocalIDs    *datatypes.StringArray
	ProfileURIs *datatypes.StringArray
}

// NewGetEndpointRequest creates an GetEndpointRequest.
func NewGetEndpointRequest(ts time.Time, handle, diag, timeout uint32, auditID string, endpoint string, localIDs, profileURIs []string) *GetEndpointRequest {
	g := &GetEndpointRequest{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(
				0, ServiceTypeGetEndpointRequest,
			),
			"", 0,
		),
		RequestHeader: NewRequestHeader(
			datatypes.NewTwoByteNodeID(0x00),
			ts,
			handle,
			diag,
			timeout,
			auditID,
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
		EndpointURL: datatypes.NewString(endpoint),
		LocalIDs:    datatypes.NewStringArray(localIDs),
		ProfileURIs: datatypes.NewStringArray(profileURIs),
	}

	return g
}

// DecodeGetEndpointRequest decodes given bytes into GetEndpointRequest.
func DecodeGetEndpointRequest(b []byte) (*GetEndpointRequest, error) {
	g := &GetEndpointRequest{}
	if err := g.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return g, nil
}

// DecodeFromBytes decodes given bytes into GetEndpointRequest.
func (g *GetEndpointRequest) DecodeFromBytes(b []byte) error {
	if len(b) < 16 {
		return &errors.ErrTooShortToDecode{g, "should be longer than 16 bytes"}
	}

	var offset = 0
	g.TypeID = &datatypes.ExpandedNodeID{}
	if err := g.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += g.TypeID.Len()

	g.RequestHeader = &RequestHeader{}
	if err := g.RequestHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += g.RequestHeader.Len() - len(g.RequestHeader.Payload)

	g.EndpointURL = &datatypes.String{}
	if err := g.EndpointURL.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += g.EndpointURL.Len()

	g.LocalIDs = &datatypes.StringArray{}
	if err := g.LocalIDs.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += g.LocalIDs.Len()

	g.ProfileURIs = &datatypes.StringArray{}
	if err := g.ProfileURIs.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += g.ProfileURIs.Len()

	return nil
}

// Serialize serializes GetEndpointRequest into bytes.
func (g *GetEndpointRequest) Serialize() ([]byte, error) {
	b := make([]byte, g.Len())
	if err := g.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes GetEndpointRequest into bytes.
func (g *GetEndpointRequest) SerializeTo(b []byte) error {
	var offset = 0
	if g.TypeID != nil {
		if err := g.TypeID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += g.TypeID.Len()
	}

	if g.RequestHeader != nil {
		if err := g.RequestHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += g.RequestHeader.Len() - len(g.Payload)
	}

	if g.EndpointURL != nil {
		if err := g.EndpointURL.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += g.EndpointURL.Len()
	}

	if g.LocalIDs != nil {
		if err := g.LocalIDs.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += g.LocalIDs.Len()
	}

	if g.ProfileURIs != nil {
		if err := g.ProfileURIs.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += g.ProfileURIs.Len()
	}

	return nil
}

// Len returns the actual length of GetEndpointRequest.
func (g *GetEndpointRequest) Len() int {
	var l = 0
	if g.TypeID != nil {
		l += g.TypeID.Len()
	}
	if g.RequestHeader != nil {
		l += (g.RequestHeader.Len() - len(g.Payload))
	}
	if g.EndpointURL != nil {
		l += g.EndpointURL.Len()
	}
	if g.LocalIDs != nil {
		l += g.LocalIDs.Len()
	}
	if g.ProfileURIs != nil {
		l += g.ProfileURIs.Len()
	}

	return l
}

// ServiceType returns type of Service in uint16.
func (g *GetEndpointRequest) ServiceType() uint16 {
	return ServiceTypeGetEndpointRequest
}
