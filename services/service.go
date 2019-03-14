// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"fmt"
	"reflect"

	"github.com/gopcua/opcua/datatypes"
	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

func init() {
	register(id.ActivateSessionRequest_Encoding_DefaultBinary, new(ActivateSessionRequest))
	register(id.ActivateSessionResponse_Encoding_DefaultBinary, new(ActivateSessionResponse))
	register(id.BrowseRequest_Encoding_DefaultBinary, new(BrowseRequest))
	register(id.BrowseResponse_Encoding_DefaultBinary, new(BrowseResponse))
	register(id.CancelRequest_Encoding_DefaultBinary, new(CancelRequest))
	register(id.CancelResponse_Encoding_DefaultBinary, new(CancelResponse))
	register(id.CloseSecureChannelRequest_Encoding_DefaultBinary, new(CloseSecureChannelRequest))
	register(id.CloseSecureChannelResponse_Encoding_DefaultBinary, new(CloseSecureChannelResponse))
	register(id.CloseSessionRequest_Encoding_DefaultBinary, new(CloseSessionRequest))
	register(id.CloseSessionResponse_Encoding_DefaultBinary, new(CloseSessionResponse))
	register(id.CreateSessionRequest_Encoding_DefaultBinary, new(CreateSessionRequest))
	register(id.CreateSessionResponse_Encoding_DefaultBinary, new(CreateSessionResponse))
	register(id.CreateSubscriptionRequest_Encoding_DefaultBinary, new(CreateSubscriptionRequest))
	register(id.CreateSubscriptionResponse_Encoding_DefaultBinary, new(CreateSubscriptionResponse))
	register(id.FindServersOnNetworkRequest_Encoding_DefaultBinary, new(FindServersOnNetworkRequest))
	register(id.FindServersOnNetworkResponse_Encoding_DefaultBinary, new(FindServersOnNetworkResponse))
	register(id.FindServersRequest_Encoding_DefaultBinary, new(FindServersRequest))
	register(id.FindServersResponse_Encoding_DefaultBinary, new(FindServersResponse))
	register(id.GetEndpointsRequest_Encoding_DefaultBinary, new(GetEndpointsRequest))
	register(id.GetEndpointsResponse_Encoding_DefaultBinary, new(GetEndpointsResponse))
	register(id.OpenSecureChannelRequest_Encoding_DefaultBinary, new(OpenSecureChannelRequest))
	register(id.OpenSecureChannelResponse_Encoding_DefaultBinary, new(OpenSecureChannelResponse))
	register(id.ReadRequest_Encoding_DefaultBinary, new(ReadRequest))
	register(id.ReadResponse_Encoding_DefaultBinary, new(ReadResponse))
	register(id.WriteRequest_Encoding_DefaultBinary, new(WriteRequest))
	register(id.WriteResponse_Encoding_DefaultBinary, new(WriteResponse))
}

var (
	serviceType   = map[uint16]reflect.Type{}
	serviceTypeID = map[reflect.Type]uint16{}
)

func register(typeID uint16, v interface{}) {
	typ := reflect.TypeOf(v) // *ServiceObject

	if serviceType[typeID] != nil {
		panic(fmt.Sprintf("Service %d is already registered", typeID))
	}
	serviceType[typeID] = typ

	if _, ok := serviceTypeID[typ]; ok {
		panic(fmt.Sprintf("Service %T is already registered", v))
	}
	serviceTypeID[typ] = typeID
}

func TypeID(v interface{}) uint16 {
	return serviceTypeID[reflect.TypeOf(v)]
}

func Decode(b []byte) (*datatypes.ExpandedNodeID, interface{}, error) {
	typeID := new(datatypes.ExpandedNodeID)
	n, err := typeID.Decode(b)
	if err != nil {
		return nil, nil, err
	}
	b = b[n:]

	id := uint16(typeID.NodeID.IntID())
	typ := serviceType[id]
	if typ == nil {
		return nil, nil, errors.NewErrUnsupported(id, "unsupported or not implemented yet.")
	}

	v := reflect.New(typ.Elem()).Interface()
	_, err = ua.Decode(b, v)
	return typeID, v, err
}
