// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/errors"
)

// ServiceType definitions.
const (
	ServiceTypeGetEndpointsRequest       uint16 = 428
	ServiceTypeGetEndpointsResponse             = 431
	ServiceTypeOpenSecureChannelRequest         = 446
	ServiceTypeOpenSecureChannelResponse        = 449
	ServiceTypeCreateSessionRequest             = 461
)

// Service is an interface to handle any kind of OPC UA Services.
type Service interface {
	DecodeFromBytes([]byte) error
	Serialize() ([]byte, error)
	SerializeTo([]byte) error
	Len() int
	String() string
	ServiceType() uint16
}

// Decode decodes given bytes into Service, depending on the type of service.
func Decode(b []byte) (Service, error) {
	var s Service

	typeID, err := datatypes.DecodeExpandedNodeID(b)
	if err != nil {
		return nil, &errors.ErrUnsupported{typeID, "cannot decode TypeID."}
	}
	n, ok := typeID.NodeID.(*datatypes.FourByteNodeID)
	if !ok {
		return nil, &errors.ErrUnsupported{typeID.NodeID, "should be FourByteNodeID."}
	}

	switch n.Identifier {
	case ServiceTypeOpenSecureChannelRequest:
		s = &OpenSecureChannelRequest{}
	case ServiceTypeOpenSecureChannelResponse:
		s = &OpenSecureChannelResponse{}
	case ServiceTypeGetEndpointsRequest:
		s = &GetEndpointsRequest{}
	case ServiceTypeGetEndpointsResponse:
		s = &GetEndpointsResponse{}
	case ServiceTypeCreateSessionRequest:
		s = &CreateSessionRequest{}
	default:
		return nil, &errors.ErrUnsupported{n.Identifier, "unsupported or not implemented yet."}
	}

	if err := s.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return s, nil
}
