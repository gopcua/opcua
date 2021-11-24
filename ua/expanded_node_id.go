// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"encoding/base64"
	"math"
	"strconv"
	"strings"

	"github.com/gopcua/opcua/errors"
)

// ExpandedNodeID extends the NodeID structure by allowing the NamespaceURI to be
// explicitly specified instead of using the NamespaceIndex. The NamespaceURI is optional.
// If it is specified, then the NamespaceIndex inside the NodeID shall be ignored.
//
// Specification: Part 6, 5.2.2.10
type ExpandedNodeID struct {
	NodeID       *NodeID
	NamespaceURI string
	ServerIndex  uint32
}

func (a ExpandedNodeID) String() string {
	return a.NodeID.String()
}

// NewExpandedNodeID creates a new ExpandedNodeID.
func NewExpandedNodeID(nodeID *NodeID, uri string, idx uint32) *ExpandedNodeID {
	e := &ExpandedNodeID{
		NodeID:      nodeID,
		ServerIndex: idx,
	}

	if uri != "" {
		e.NodeID.SetURIFlag()
		e.NamespaceURI = uri
	}

	if idx > 0 {
		e.NodeID.SetIndexFlag()
	}

	return e
}

// NewTwoByteExpandedNodeID creates a two byte numeric expanded node id.
func NewTwoByteExpandedNodeID(id uint8) *ExpandedNodeID {
	return &ExpandedNodeID{
		NodeID: NewTwoByteNodeID(id),
	}
}

// NewFourByteExpandedNodeID creates a four byte numeric expanded node id.
func NewFourByteExpandedNodeID(ns uint8, id uint16) *ExpandedNodeID {
	return &ExpandedNodeID{
		NodeID: NewFourByteNodeID(ns, id),
	}
}

// NewNumericExpandedNodeID creates a numeric expanded node id.
func NewNumericExpandedNodeID(ns uint16, id uint32) *ExpandedNodeID {
	return &ExpandedNodeID{
		NodeID: NewNumericNodeID(ns, id),
	}
}

// NewStringExpandedNodeID creates a string expanded node id.
func NewStringExpandedNodeID(ns uint16, id string) *ExpandedNodeID {
	return &ExpandedNodeID{
		NodeID: NewStringNodeID(ns, id),
	}
}

// NewGUIDExpandedNodeID creates a GUID expanded node id.
func NewGUIDExpandedNodeID(ns uint16, id string) *ExpandedNodeID {
	return &ExpandedNodeID{
		NodeID: NewGUIDNodeID(ns, id),
	}
}

// NewByteStringExpandedNodeID creates a byte string expanded node id.
func NewByteStringExpandedNodeID(ns uint16, id []byte) *ExpandedNodeID {
	return &ExpandedNodeID{
		NodeID: NewByteStringNodeID(ns, id),
	}
}

func (e *ExpandedNodeID) Decode(b []byte) (int, error) {
	buf := NewBuffer(b)
	e.NodeID = new(NodeID)
	buf.ReadStruct(e.NodeID)
	if e.HasNamespaceURI() {
		e.NamespaceURI = buf.ReadString()
	}
	if e.HasServerIndex() {
		e.ServerIndex = buf.ReadUint32()
	}
	return buf.Pos(), buf.Error()
}

func (e *ExpandedNodeID) Encode() ([]byte, error) {
	buf := NewBuffer(nil)
	buf.WriteStruct(e.NodeID)
	if e.HasNamespaceURI() {
		buf.WriteString(e.NamespaceURI)
	}
	if e.HasServerIndex() {
		buf.WriteUint32(e.ServerIndex)
	}
	return buf.Bytes(), buf.Error()

}

// HasNamespaceURI checks if an ExpandedNodeID has NamespaceURI Flag.
func (e *ExpandedNodeID) HasNamespaceURI() bool {
	return e.NodeID.EncodingMask()>>7&0x1 == 1
}

// HasServerIndex checks if an ExpandedNodeID has ServerIndex Flag.
func (e *ExpandedNodeID) HasServerIndex() bool {
	return e.NodeID.EncodingMask()>>6&0x1 == 1
}

// ParseExpandedNodeID returns a node id from a string definition of the format
// '{ns,nsu}=<namespace>;{s,i,b,g}=<identifier>'.
//
// The 's=' prefix can be omitted for string node ids in namespace 0.
//
// For numeric ids the smallest possible type which can store the namespace
// and id value is returned.
//
// Namespace URIs are resolved to ids from the provided list of namespaces.
func ParseExpandedNodeID(s string, ns []string) (*ExpandedNodeID, error) {
	if s == "" {
		return NewTwoByteExpandedNodeID(0), nil
	}

	var nsval, idval string

	p := strings.SplitN(s, ";", 3)
	switch len(p) {
	case 1:
		nsval, idval = "ns=0", p[0]
	case 2:
		nsval, idval = p[0], p[1]
	default:
		return nil, errors.Errorf("invalid node id: %s", s)
	}

	// parse namespace
	var nsid uint16
	var nsu string
	switch {
	case strings.HasPrefix(nsval, "nsu="):
		if ns == nil {
			return nil, errors.Errorf("namespace urls require a server NamespaceArray")
		}

		nsuval := strings.TrimPrefix(nsval, "nsu=")
		ok := false
		for id, uri := range ns {
			if uri == nsuval {
				nsid, nsu = uint16(id), uri
				ok = true
				break
			}
		}
		if !ok {
			return nil, errors.Errorf("namespace uri %s not found in the server NamespaceArray %#v", nsval, ns)
		}

	case strings.HasPrefix(nsval, "ns="):
		n, err := strconv.Atoi(nsval[3:])
		if err != nil {
			return nil, errors.Errorf("invalid namespace id: %s", s)
		}
		if n < 0 || n > math.MaxUint16 {
			return nil, errors.Errorf("namespace id out of range (0..65535): %s", s)
		}
		nsid = uint16(n)

	default:
		return nil, errors.Errorf("invalid node id: %s", s)
	}

	// parse identifier
	switch {
	case strings.HasPrefix(idval, "i="):
		id, err := strconv.ParseUint(idval[2:], 10, 64)
		if err != nil {
			return nil, errors.Errorf("invalid numeric id: %s", s)
		}
		switch {
		case nsid == 0 && id < 256:
			return NewExpandedNodeID(NewTwoByteNodeID(byte(id)), "", 0), nil
		case nsid < 256 && id < math.MaxUint16:
			return NewExpandedNodeID(NewFourByteNodeID(byte(nsid), uint16(id)), nsu, 0), nil
		case id <= math.MaxUint32:
			return NewExpandedNodeID(NewNumericNodeID(nsid, uint32(id)), nsu, 0), nil
		default:
			return nil, errors.Errorf("numeric id out of range (0..2^32-1): %s", s)
		}

	case strings.HasPrefix(idval, "s="):
		return NewExpandedNodeID(NewStringNodeID(nsid, idval[2:]), nsu, 0), nil

	case strings.HasPrefix(idval, "g="):
		n := NewGUIDNodeID(nsid, idval[2:])
		if n == nil || n.StringID() == "" {
			return nil, errors.Errorf("invalid guid node id: %s", s)
		}
		return NewExpandedNodeID(n, nsu, 0), nil

	case strings.HasPrefix(idval, "b="):
		b, err := base64.StdEncoding.DecodeString(idval[2:])
		if err != nil {
			return nil, errors.Errorf("invalid opaque node id: %s", s)
		}
		return NewExpandedNodeID(NewByteStringNodeID(nsid, b), nsu, 0), nil

	case strings.HasPrefix(idval, "ns="):
		return nil, errors.Errorf("invalid node id: %s", s)

	default:
		return NewExpandedNodeID(NewStringNodeID(nsid, idval), nsu, 0), nil
	}
}
