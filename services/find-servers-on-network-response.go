// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/errors"
	"github.com/wmnsk/gopcua/utils"
)

// FindServersOnNetworkResponse returns the Servers known to a Server or Discovery Server. The behaviour of
// Discovery Servers is described in detail in Part 12.
//
// Specification: Part4, 5.4.2
type FindServersOnNetworkResponse struct {
	TypeID *datatypes.ExpandedNodeID
	*ResponseHeader
	LastCounterResetTime time.Time
	Servers              *datatypes.ServersOnNetworkArray
}

// NewFindServersOnNetworkResponse creates an FindServersOnNetworkResponse.
func NewFindServersOnNetworkResponse(resHeader *ResponseHeader, resetTime time.Time, servers ...*datatypes.ServersOnNetwork) *FindServersOnNetworkResponse {
	return &FindServersOnNetworkResponse{
		TypeID:               datatypes.NewFourByteExpandedNodeID(0, ServiceTypeFindServersOnNetworkResponse),
		ResponseHeader:       resHeader,
		LastCounterResetTime: resetTime,
		Servers:              datatypes.NewServersOnNetworkArray(servers),
	}
}

// DecodeFindServersOnNetworkResponse decodes given bytes into FindServersOnNetworkResponse.
func DecodeFindServersOnNetworkResponse(b []byte) (*FindServersOnNetworkResponse, error) {
	f := &FindServersOnNetworkResponse{}
	if err := f.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return f, nil
}

// DecodeFromBytes decodes given bytes into FindServersOnNetworkResponse.
func (f *FindServersOnNetworkResponse) DecodeFromBytes(b []byte) error {
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

	f.LastCounterResetTime = utils.DecodeTimestamp(b[offset:])
	offset += 8

	f.Servers = &datatypes.ServersOnNetworkArray{}
	return f.Servers.DecodeFromBytes(b[offset:])
}

// Serialize serializes FindServersOnNetworkResponse into bytes.
func (f *FindServersOnNetworkResponse) Serialize() ([]byte, error) {
	b := make([]byte, f.Len())
	if err := f.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes FindServersOnNetworkResponse into bytes.
func (f *FindServersOnNetworkResponse) SerializeTo(b []byte) error {
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

	utils.EncodeTimestamp(b[offset:], f.LastCounterResetTime)
	offset += 8

	if f.Servers != nil {
		if err := f.Servers.SerializeTo(b[offset:]); err != nil {
			return err
		}
	}

	return nil
}

// Len returns the actual length of FindServersOnNetworkResponse.
func (f *FindServersOnNetworkResponse) Len() int {
	var l = 8
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
func (f *FindServersOnNetworkResponse) ServiceType() uint16 {
	return ServiceTypeFindServersOnNetworkResponse
}
