// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/errors"
)

// svcreg contains all known service request/response objects.
var svcreg = NewTypeRegistry()

// TypeRegistry provides a registry for Go types.
//
// Each type is registered with a unique identifier
// which cannot be changed for the lifetime of the component.
//
// Types can be registered multiple times under different
// identifiers.
//
// The implementation is safe for concurrent use.
type TypeRegistry struct {
	mu    sync.Mutex
	types map[string]reflect.Type
	ids   map[reflect.Type]string
}

func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		types: make(map[string]reflect.Type),
		ids:   make(map[reflect.Type]string),
	}
}

// New returns a new instance of the type with the given id.
//
// If the id is not known the function returns nil.
//
// New panics if id is nil.
func (r *TypeRegistry) New(id *NodeID) interface{} {
	if id == nil {
		panic("opcua: missing id in call to TypeRegistry.New")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	typ, ok := r.types[id.String()]
	if !ok {
		return nil
	}
	return reflect.New(typ.Elem()).Interface()
}

// Lookup returns the id of the type of v or nil if
// the type is not registered.
//
// If the type was registered multiple times the first
// registered id for this type is returned.
func (r *TypeRegistry) Lookup(v interface{}) *NodeID {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.ids[reflect.TypeOf(v)]; ok {
		return MustParseNodeID(id)
	}
	return nil
}

// Register adds a new type to the registry.
//
// If the id is already registered as a different type the function returns an error.
//
// Register panics if id is nil.
func (r *TypeRegistry) Register(id *NodeID, v interface{}) error {
	if id == nil {
		panic("opcua: missing id in call to TypeRegistry.Register")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	typ := reflect.TypeOf(v)
	ids := id.String()

	if cur := r.types[ids]; cur != nil && cur != typ {
		return errors.Errorf("%s is already registered as %v", id, cur)
	}
	r.types[ids] = typ

	if _, exists := r.ids[typ]; !exists {
		r.ids[typ] = ids
	}
	return nil
}

// RegisterService registers a new service object type.
// It panics if the type or the id is already registered.
func RegisterService(typeID uint16, v interface{}) {
	if err := svcreg.Register(NewFourByteNodeID(0, typeID), v); err != nil {
		panic("Service " + err.Error())
	}
}

// ServiceTypeID returns the id of the service object type as
// registered with RegisterService. If the service object is not
// known the function returns 0.
func ServiceTypeID(v interface{}) uint16 {
	id := svcreg.Lookup(v)
	if id == nil {
		return 0
	}
	return uint16(id.IntID())
}

func DecodeService(b []byte) (*ExpandedNodeID, interface{}, error) {
	typeID := new(ExpandedNodeID)
	n, err := typeID.Decode(b)
	if err != nil {
		return nil, nil, err
	}
	b = b[n:]

	v := svcreg.New(typeID.NodeID)
	if v == nil {
		return nil, nil, StatusBadServiceUnsupported
	}

	if debug.FlagSet("packet") {
		fmt.Printf("%T: %#v\n", v, b)
	}

	_, err = Decode(b, v)
	return typeID, v, err
}
