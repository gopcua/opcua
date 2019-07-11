// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"fmt"
	"reflect"
	"sync"
)

// TypeRegistry maps numeric ids to Go types.
// The implementation is safe for concurrent use.
type TypeRegistry struct {
	mu    sync.RWMutex
	types map[uint32]reflect.Type
	ids   map[reflect.Type]uint32
}

// NewTypeRegistry returns a new type registry.
func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		types: make(map[uint32]reflect.Type),
		ids:   make(map[reflect.Type]uint32),
	}
}

// New returns a new instance of the type with the given id.
// If the id is not known the function returns nil.
func (r *TypeRegistry) New(id uint32) interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	typ, ok := r.types[id]
	if !ok {
		return nil
	}
	return reflect.New(typ.Elem()).Interface()
}

// Lookup returns the id of the type of v or zero if the
// type is not registered.
func (r *TypeRegistry) Lookup(v interface{}) uint32 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.ids[reflect.TypeOf(v)]
}

// Register adds a new type to the registry. If either the type
// or the id is already registered the function returns an error.
func (r *TypeRegistry) Register(id uint32, v interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

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
