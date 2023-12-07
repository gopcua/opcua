// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"sync"

	"github.com/gopcua/opcua/errors"
)

type FuncRegistry struct {
	mu          sync.Mutex
	encodeFuncs map[string]encodefunc
	decodeFuncs map[string]decodefunc
}

// NewFuncRegistry returns a new func registry.
func NewFuncRegistry() *FuncRegistry {
	return &FuncRegistry{
		encodeFuncs: make(map[string]encodefunc),
		decodeFuncs: make(map[string]decodefunc),
	}
}

// EncodeFunc returns the function registered to encode Node with ID id
//
// If the id is not known the function returns nil.
//
// New panics if id is nil.
func (r *FuncRegistry) EncodeFunc(id *NodeID) encodefunc {
	if id == nil {
		panic("opcua: missing id in call to FuncRegistry.New")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	f, ok := r.encodeFuncs[id.String()]
	if !ok {
		return nil
	}
	return f
}

// DecodeFunc returns the function registered to decode Node with ID id
//
// If the id is not known the function returns nil.
//
// New panics if id is nil.
func (r *FuncRegistry) DecodeFunc(id *NodeID) decodefunc {
	if id == nil {
		panic("opcua: missing id in call to FuncRegistry.New")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	f, ok := r.decodeFuncs[id.String()]
	if !ok {
		return nil
	}
	return f
}

// Register adds a new node to the registry.
//
// If the id is already registered the function returns an error.
//
// Register panics if id is nil.
func (r *FuncRegistry) Register(id *NodeID, ef encodefunc, df decodefunc) error {
	if id == nil {
		panic("opcua: missing id in call to FuncRegistry.Register")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	ids := id.String()

	if cur := r.encodeFuncs[ids]; cur != nil {
		return errors.Errorf("%s is already registered", id)
	}
	r.encodeFuncs[ids] = ef

	if _, exists := r.decodeFuncs[ids]; !exists {
		r.decodeFuncs[ids] = df
	}
	return nil
}

// Deregister removes a node from the registry
func (r *FuncRegistry) Deregister(id *NodeID) {
	if id == nil {
		panic("opcua: missing id in call to FuncRegistry.Register")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	ids := id.String()
	delete(r.encodeFuncs, ids)
	delete(r.decodeFuncs, ids)
}
