// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

//go:generate go run ../cmd/predefined-nodes/main.go

package server

import (
	"sync"
	"time"

	"github.com/gopcua/opcua/ua"
)

type Node interface {
	ID() *ua.NodeID
	Attribute(ua.AttributeID) (*ua.Variant, time.Time, error)
}

type AddressSpace struct {
	mu    sync.RWMutex
	nodes map[string]Node
}

func newAddressSpace(nodes ...Node) *AddressSpace {
	as := &AddressSpace{nodes: make(map[string]Node)}
	as.AddNode(&currentTime{})
	as.AddNode(&serverStatus{
		&ua.ServerStatusDataType{
			StartTime:   time.Now(),
			CurrentTime: time.Now(),
			State:       ua.ServerStateRunning,
			BuildInfo: &ua.BuildInfo{
				ProductURI:       "http://open62541.org",
				ManufacturerName: "open62541",
				ProductName:      "open62541 OPC UA Server",
				SoftwareVersion:  "0.4.0-dev",
				BuildNumber:      "Mar  4 2019 15:22:43",
				BuildDate:        time.Time{},
			},
			SecondsTillShutdown: 0,
			ShutdownReason:      &ua.LocalizedText{},
		},
	})
	return as
}

func (a *AddressSpace) AddNode(n Node) error {
	a.mu.Lock()
	a.nodes[n.ID().String()] = n
	a.mu.Unlock()
	return nil
}

func (a *AddressSpace) Attribute(id *ua.NodeID, attr ua.AttributeID) (*ua.Variant, time.Time, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	n := a.nodes[id.String()]
	if n == nil {
		return nil, time.Time{}, ua.StatusBadNodeIDUnknown
	}
	return n.Attribute(attr)
}

type node struct {
	id   *ua.NodeID
	attr map[ua.AttributeID]*ua.Variant
	ts   map[ua.AttributeID]time.Time
}

func (n *node) ID() *ua.NodeID {
	return n.id
}

func (n *node) Attribute(id ua.AttributeID) (*ua.Variant, time.Time, error) {
	if n.attr == nil {
		return nil, time.Time{}, ua.StatusBadAttributeIDInvalid
	}
	v := n.attr[id]
	if v == nil {
		return nil, time.Time{}, ua.StatusBadAttributeIDInvalid
	}
	return v, n.ts[id], nil
}

type currentTime struct{}

func (n *currentTime) ID() *ua.NodeID {
	return ua.NewNumericNodeID(0, 2258)
}

func (n *currentTime) Attribute(attr ua.AttributeID) (*ua.Variant, time.Time, error) {
	switch attr {
	case ua.AttributeIDBrowseName:
		return ua.MustVariant("CurrentTime"), time.Time{}, nil
	case ua.AttributeIDValue:
		return ua.MustVariant(time.Now()), time.Time{}, nil
	default:
		return nil, time.Time{}, ua.StatusBadAttributeIDInvalid
	}
}

type serverStatus struct {
	v *ua.ServerStatusDataType
}

func (n *serverStatus) ID() *ua.NodeID {
	return ua.NewNumericNodeID(0, 2256)
}

func (n *serverStatus) Attribute(attr ua.AttributeID) (*ua.Variant, time.Time, error) {
	switch attr {
	case ua.AttributeIDBrowseName:
		return ua.MustVariant("ServerStatus"), time.Time{}, nil
	case ua.AttributeIDValue:
		return ua.MustVariant(ua.NewExtensionObject(n.v)), time.Time{}, nil
	default:
		return nil, time.Time{}, ua.StatusBadAttributeIDInvalid
	}
}
