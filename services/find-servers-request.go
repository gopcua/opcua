// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
)

// FindServersRequest returns the Servers known to a Server or Discovery Server. The behaviour of
// Discovery Servers is described in detail in Part 12.
//
// The Client may reduce the number of results returned by specifying filter criteria. A Discovery
// Server returns an empty list if no Servers match the criteria specified by the client. The filter
// criteria supported by this Service are described in 5.4.2.2.
//
// Specification: Part 4, 5.4.2
type FindServersRequest struct {
	TypeID *datatypes.ExpandedNodeID
	*RequestHeader
	EndpointURL *datatypes.String
	LocaleIDs   *datatypes.StringArray
	ServerURIs  *datatypes.StringArray
}

// NewFindServersRequest creates a new FindServersRequest.
func NewFindServersRequest(reqHeader *RequestHeader, url string, locales []string, serverURIs ...string) *FindServersRequest {
	f := &FindServersRequest{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(
				0, ServiceTypeFindServersRequest,
			),
			"", 0,
		),
		RequestHeader: reqHeader,
		EndpointURL:   datatypes.NewString(url),
		LocaleIDs:     datatypes.NewStringArray(locales),
	}
	// No ServerURIs
	if len(serverURIs) == 1 && serverURIs[0] == "" {
		f.ServerURIs = &datatypes.StringArray{ArraySize: 0}
		return f
	}
	f.ServerURIs = datatypes.NewStringArray(serverURIs)
	return f
}

// DecodeFindServersRequest decodes given bytes into FindServersRequest.
func DecodeFindServersRequest(b []byte) (*FindServersRequest, error) {
	f := &FindServersRequest{}
	if err := f.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return f, nil
}

// DecodeFromBytes decodes given bytes into FindServersRequest.
func (f *FindServersRequest) DecodeFromBytes(b []byte) error {
	offset := 0

	// type id
	f.TypeID = &datatypes.ExpandedNodeID{}
	if err := f.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += f.TypeID.Len()

	f.RequestHeader = &RequestHeader{}
	if err := f.RequestHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += f.RequestHeader.Len() - len(f.RequestHeader.Payload)

	f.EndpointURL = &datatypes.String{}
	if err := f.EndpointURL.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += f.EndpointURL.Len()

	f.LocaleIDs = &datatypes.StringArray{}
	if err := f.LocaleIDs.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += f.LocaleIDs.Len()

	// user identity token
	f.ServerURIs = &datatypes.StringArray{}
	return f.ServerURIs.DecodeFromBytes(b[offset:])
}

// Serialize serializes FindServersRequest into bytes.
func (f *FindServersRequest) Serialize() ([]byte, error) {
	b := make([]byte, f.Len())
	if err := f.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes FindServersRequest into bytes.
func (f *FindServersRequest) SerializeTo(b []byte) error {
	var offset = 0
	if f.TypeID != nil {
		if err := f.TypeID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += f.TypeID.Len()
	}

	if f.RequestHeader != nil {
		if err := f.RequestHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += f.RequestHeader.Len()
	}

	if f.EndpointURL != nil {
		if err := f.EndpointURL.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += f.EndpointURL.Len()
	}

	if f.LocaleIDs != nil {
		if err := f.LocaleIDs.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += f.LocaleIDs.Len()
	}

	if f.ServerURIs != nil {
		if err := f.ServerURIs.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += f.ServerURIs.Len()
	}

	return nil
}

// Len returns the actual length of FindServersRequest.
func (f *FindServersRequest) Len() int {
	l := 0

	if f.TypeID != nil {
		l += f.TypeID.Len()
	}

	if f.RequestHeader != nil {
		l += f.RequestHeader.Len()
	}

	if f.EndpointURL != nil {
		l += f.EndpointURL.Len()
	}

	if f.LocaleIDs != nil {
		l += f.LocaleIDs.Len()
	}

	if f.ServerURIs != nil {
		l += f.ServerURIs.Len()
	}

	return l
}

// ServiceType returns type of Service.
func (f *FindServersRequest) ServiceType() uint16 {
	return ServiceTypeFindServersRequest
}
