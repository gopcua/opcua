// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"fmt"
	"reflect"

	"github.com/gopcua/opcua/debug"
)

// svcreg contains all known service request/response objects.
var svcreg = NewFuncRegistry()
var svctypeids = map[reflect.Type]uint16{}

// RegisterService registers a new service object type.
// It panics if the type or the id is already registered.
func RegisterService(typeID uint16, v interface{}) {
	ef := func(vv interface{}) ([]byte, error) {
		// TODO check/ensure vv is of type v ?
		// TODO
		return Encode(vv)
	}
	df := func(b []byte, vv interface{}) error {
		rv := reflect.ValueOf(vv)
		if rv.Kind() != reflect.Pointer || rv.IsNil() {
			return fmt.Errorf("incorrect type to decode into")
		}
		r := reflect.New(reflect.TypeOf(v).Elem()).Interface()
		buf := NewBuffer(b)
		buf.ReadStruct(r)
		reflect.Indirect(rv).Set(reflect.ValueOf(r))
		return nil
	}
	nodeID := NewFourByteExpandedNodeID(0, typeID).NodeID
	if err := svcreg.Register(nodeID, ef, df); err != nil {
		panic("Service " + err.Error())
	}
	typ := reflect.TypeOf(v)
	svctypeids[typ] = uint16(nodeID.IntID())
}

// ServiceTypeID returns the id of the service object type as
// registered with RegisterService. If the service object is not
// known the function returns 0.
func ServiceTypeID(v interface{}) uint16 {
	id, ok := svctypeids[reflect.TypeOf(v)]
	if !ok {
		return 0
	}
	return id
}

func DecodeService(b []byte) (*ExpandedNodeID, interface{}, error) {
	typeID := new(ExpandedNodeID)
	n, err := typeID.Decode(b)
	if err != nil {
		return nil, nil, err
	}
	b = b[n:]

	decode := svcreg.DecodeFunc(typeID.NodeID)
	if decode == nil {
		return nil, nil, StatusBadServiceUnsupported
	}

	if debug.FlagSet("packet") {
		fmt.Printf("%T: %#v\n", decode, b)
	}

	var v interface{}
	err = decode(b, &v)
	return typeID, v, err
}
