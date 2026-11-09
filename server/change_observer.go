// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package server

import (
	"time"

	"github.com/gopcua/opcua/ua"
)

// ChangeEvent describes a change of a node in the server's address space.
type ChangeEvent struct {
	// NodeID of the changed node.
	NodeID *ua.NodeID
	// Value is the current Value attribute of the node.
	Value *ua.DataValue
	// Time is the server time of the change notification.
	Time time.Time
}

// ChangeObserver receives node change events.
//
// Observers are called synchronously on the goroutine that changed the node,
// possibly while namespace locks are held. They MUST NOT block and MUST NOT
// call back into the namespace that triggered the change.
type ChangeObserver func(ChangeEvent)

// OnChange registers fn to be called for every node change: writes from OPC UA
// clients as well as application side changes that call ChangeNotification
// (e.g. MapNamespace.SetValue).
func OnChange(fn ChangeObserver) Option {
	return func(s *serverConfig) {
		if fn != nil {
			s.changeObservers = append(s.changeObservers, fn)
		}
	}
}

func (s *Server) notifyObservers(n *ua.NodeID) {
	if len(s.cfg.changeObservers) == 0 || n == nil {
		return
	}
	ns, err := s.Namespace(int(n.Namespace()))
	if err != nil {
		return
	}
	ev := ChangeEvent{
		NodeID: n,
		Value:  ns.Attribute(n, ua.AttributeIDValue),
		Time:   time.Now(),
	}
	for _, fn := range s.cfg.changeObservers {
		fn(ev)
	}
}
