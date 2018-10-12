// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/errors"
)

// FindServersResponse returns the Servers known to a Server or Discovery Server. The behaviour of
// Discovery Servers is described in detail in Part 12.
//
// Specification: Part4, 5.4.2
type FindServersResponse struct {
	TypeID *datatypes.ExpandedNodeID
	*ResponseHeader
	Servers *datatypes.ApplicationDescriptionArray
}

// NewFindServersResponse creates an FindServersResponse.
func NewFindServersResponse(resHeader *ResponseHeader, servers ...*datatypes.ApplicationDescription) *FindServersResponse {
	return &FindServersResponse{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(
				0, ServiceTypeFindServersResponse,
			),
			"", 0,
		),
		ResponseHeader: resHeader,
		Servers:        datatypes.NewApplicationDescriptionArray(servers),
	}
}

// DecodeFindServersResponse decodes given bytes into FindServersResponse.
func DecodeFindServersResponse(b []byte) (*FindServersResponse, error) {
	f := &FindServersResponse{}
	if err := f.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return f, nil
}

// DecodeFromBytes decodes given bytes into FindServersResponse.
func (f *FindServersResponse) DecodeFromBytes(b []byte) error {
	if len(b) < 16 {
		return errors.NewErrTooShortToDecode(f, "should be longer than 16 bytes")
	}

	var offset = 0
	f.TypeID = &datatypes.ExpandedNodeID{}
	if err := f.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += f.TypeID.Len()

	f.ResponseHeader = &ResponseHeader{}
	if err := f.ResponseHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += f.ResponseHeader.Len() - len(f.ResponseHeader.Payload)

	f.Servers = &datatypes.ApplicationDescriptionArray{}
	return f.Servers.DecodeFromBytes(b[offset:])
}

// Serialize serializes FindServersResponse into bytes.
func (f *FindServersResponse) Serialize() ([]byte, error) {
	b := make([]byte, f.Len())
	if err := f.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes FindServersResponse into bytes.
func (f *FindServersResponse) SerializeTo(b []byte) error {
	var offset = 0
	if f.TypeID != nil {
		if err := f.TypeID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += f.TypeID.Len()
	}

	if f.ResponseHeader != nil {
		if err := f.ResponseHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += f.ResponseHeader.Len() - len(f.Payload)
	}

	if f.Servers != nil {
		if err := f.Servers.SerializeTo(b[offset:]); err != nil {
			return err
		}
	}

	return nil
}

// Len returns the actual length of FindServersResponse.
func (f *FindServersResponse) Len() int {
	var l = 0
	if f.TypeID != nil {
		l += f.TypeID.Len()
	}
	if f.ResponseHeader != nil {
		l += (f.ResponseHeader.Len() - len(f.Payload))
	}
	if f.Servers != nil {
		l += f.Servers.Len()
	}

	return l
}

// ServiceType returns type of Service in uint16.
func (f *FindServersResponse) ServiceType() uint16 {
	return ServiceTypeFindServersResponse
}
