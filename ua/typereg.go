// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"fmt"
	"reflect"
)

type TypeRegistry struct {
	types map[uint32]reflect.Type
	ids   map[reflect.Type]uint32
}

func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		types: make(map[uint32]reflect.Type),
		ids:   make(map[reflect.Type]uint32),
	}
}

func (r *TypeRegistry) New(id uint32) interface{} {
	typ, ok := r.types[id]
	if !ok {
		return nil
	}
	return reflect.New(typ.Elem()).Interface()
}

func (r *TypeRegistry) Lookup(v interface{}) uint32 {
	return r.ids[reflect.TypeOf(v)]
}

func (r *TypeRegistry) Register(id uint32, v interface{}) error {
	typ := reflect.TypeOf(v)

	if r.types[id] != nil {
		return fmt.Errorf("%d is already registered", id)
	}
	r.types[id] = typ

	if _, ok := r.ids[typ]; ok {
		return fmt.Errorf("%T is already registered", v)
	}
	r.ids[typ] = id
	return nil
}
