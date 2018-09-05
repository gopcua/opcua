// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/errors"
)

// GetEndpointsRequest represents an GetEndpointsRequest.
// This Service returns the Endpoints supported by a Server and all of the configuration information
// required to establish a SecureChannel and a Session.
//
// Specification: Part 4, 5.4.4.2
type GetEndpointsRequest struct {
	TypeID *datatypes.ExpandedNodeID
	*RequestHeader
	EndpointURL *datatypes.String
	LocaleIDs   *datatypes.StringArray
	ProfileURIs *datatypes.StringArray
}

// NewGetEndpointsRequest creates an GetEndpointsRequest.
func NewGetEndpointsRequest(ts time.Time, handle, diag, timeout uint32, auditID string, endpoint string, localIDs, profileURIs []string) *GetEndpointsRequest {
	return &GetEndpointsRequest{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(
				0, ServiceTypeGetEndpointsRequest,
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
		LocaleIDs:   datatypes.NewStringArray(localIDs),
		ProfileURIs: datatypes.NewStringArray(profileURIs),
	}
}

// DecodeGetEndpointsRequest decodes given bytes into GetEndpointsRequest.
func DecodeGetEndpointsRequest(b []byte) (*GetEndpointsRequest, error) {
	g := &GetEndpointsRequest{}
	if err := g.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return g, nil
}

// DecodeFromBytes decodes given bytes into GetEndpointsRequest.
func (g *GetEndpointsRequest) DecodeFromBytes(b []byte) error {
	if len(b) < 16 {
		return errors.NewErrTooShortToDecode(g, "should be longer than 16 bytes")
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

	g.LocaleIDs = &datatypes.StringArray{}
	if err := g.LocaleIDs.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += g.LocaleIDs.Len()

	g.ProfileURIs = &datatypes.StringArray{}
	if err := g.ProfileURIs.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += g.ProfileURIs.Len()

	return nil
}

// Serialize serializes GetEndpointsRequest into bytes.
func (g *GetEndpointsRequest) Serialize() ([]byte, error) {
	b := make([]byte, g.Len())
	if err := g.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes GetEndpointsRequest into bytes.
func (g *GetEndpointsRequest) SerializeTo(b []byte) error {
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

	if g.LocaleIDs != nil {
		if err := g.LocaleIDs.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += g.LocaleIDs.Len()
	}

	if g.ProfileURIs != nil {
		if err := g.ProfileURIs.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += g.ProfileURIs.Len()
	}

	return nil
}

// Len returns the actual length of GetEndpointsRequest.
func (g *GetEndpointsRequest) Len() int {
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
	if g.LocaleIDs != nil {
		l += g.LocaleIDs.Len()
	}
	if g.ProfileURIs != nil {
		l += g.ProfileURIs.Len()
	}

	return l
}

// ServiceType returns type of Service in uint16.
func (g *GetEndpointsRequest) ServiceType() uint16 {
	return ServiceTypeGetEndpointsRequest
}
