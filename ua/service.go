// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"fmt"
	"reflect"

	"github.com/gopcua/opcua/debug"
)

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

func ServiceTypeID(v interface{}) uint16 {
	return serviceTypeID[reflect.TypeOf(v)]
}

func DecodeService(b []byte) (*ExpandedNodeID, interface{}, error) {
	typeID := new(ExpandedNodeID)
	n, err := typeID.Decode(b)
	if err != nil {
		return nil, nil, err
	}
	b = b[n:]

	id := uint16(typeID.NodeID.IntID())
	typ := serviceType[id]
	if typ == nil {
		return nil, nil, StatusBadServiceUnsupported
	}

	v := reflect.New(typ.Elem()).Interface()

	if debug.FlagSet("packet") {
		fmt.Printf("%T: %#v\n", v, b)
	}
	_, err = Decode(b, v)
	return typeID, v, err
}
