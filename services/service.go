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
	ServiceTypeGetEndpointsRequest        uint16 = 428
	ServiceTypeGetEndpointsResponse              = 431
	ServiceTypeOpenSecureChannelRequest          = 446
	ServiceTypeOpenSecureChannelResponse         = 449
	ServiceTypeCloseSecureChannelRequest         = 452
	ServiceTypeCloseSecureChannelResponse        = 455
	ServiceTypeCreateSessionRequest              = 461
	ServiceTypeCreateSessionResponse             = 464
	ServiceTypeActivateSessionRequest            = 467
	ServiceTypeActivateSessionResponse           = 470
	ServiceTypeCloseSessionRequest               = 473
	ServiceTypeCloseSessionResponse              = 476
	ServiceTypeReadRequest                       = 631
	ServiceTypeReadResponse                      = 634
)

// Decode decodes given bytes into Service, depending on the type of service.
func Decode(b []byte) (datatypes.Service, error) {
	var s datatypes.Service

	typeID, err := datatypes.DecodeExpandedNodeID(b)
	if err != nil {
		return nil, errors.NewErrUnsupported(typeID, "cannot decode TypeID.")
	}
	n, ok := typeID.NodeID.(*datatypes.FourByteNodeID)
	if !ok {
		return nil, errors.NewErrUnsupported(typeID.NodeID, "should be FourByteNodeID.")
	}

	switch n.Identifier {
	case ServiceTypeOpenSecureChannelRequest:
		s = &OpenSecureChannelRequest{}
	case ServiceTypeOpenSecureChannelResponse:
		s = &OpenSecureChannelResponse{}
	case ServiceTypeCloseSecureChannelRequest:
		s = &CloseSecureChannelRequest{}
	case ServiceTypeCloseSecureChannelResponse:
		s = &CloseSecureChannelResponse{}
	case ServiceTypeGetEndpointsRequest:
		s = &GetEndpointsRequest{}
	case ServiceTypeGetEndpointsResponse:
		s = &GetEndpointsResponse{}
	case ServiceTypeCreateSessionRequest:
		s = &CreateSessionRequest{}
	case ServiceTypeCreateSessionResponse:
		s = &CreateSessionResponse{}
	case ServiceTypeCloseSessionRequest:
		s = &CloseSessionRequest{}
	case ServiceTypeCloseSessionResponse:
		s = &CloseSessionResponse{}
	case ServiceTypeActivateSessionRequest:
		s = &ActivateSessionRequest{}
	case ServiceTypeActivateSessionResponse:
		s = &ActivateSessionResponse{}
	case ServiceTypeReadRequest:
		s = &ReadRequest{}
	default:
		return nil, errors.NewErrUnsupported(n.Identifier, "unsupported or not implemented yet.")
	}

	if err := s.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return s, nil
}
