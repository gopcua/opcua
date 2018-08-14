// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/binary"
)

// ExpandedNodeID represents the ExpandedNodeID.
type ExpandedNodeID struct {
	NodeID
	NamespaceURI *String
	ServerIndex  uint32
}

// NewExpandedNodeID creates a new ExpandedNodeID.
func NewExpandedNodeID(hasURI, hasIndex bool, nodeID NodeID, uri string, idx uint32) *ExpandedNodeID {
	e := &ExpandedNodeID{
		NodeID:       nodeID,
		NamespaceURI: NewString(uri),
		ServerIndex:  idx,
	}

	if hasURI {
		e.NodeID.SetURIFlag()
	}
	if hasIndex {
		e.NodeID.SetIndexFlag()
	}

	return e
}

// DecodeExpandedNodeID decodes given bytes into ExpandedNodeID.
func DecodeExpandedNodeID(b []byte) (*ExpandedNodeID, error) {
	e := &ExpandedNodeID{}
	if err := e.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return e, nil
}

// DecodeFromBytes decodes given bytes into ExpandedNodeID.
func (e *ExpandedNodeID) DecodeFromBytes(b []byte) error {
	node, err := DecodeNodeID(b)
	if err != nil {
		return err
	}
	e.NodeID = node
	b = b[node.Len():]
	if len(b) < 2 {
		return nil
	}

	if e.HasNamespaceURI() {
		e.NamespaceURI = &String{}
		if err := e.NamespaceURI.DecodeFromBytes(b); err != nil {
			return err
		}
		b = b[e.NamespaceURI.Len():]
	}

	if e.HasServerIndex() {
		e.ServerIndex = binary.LittleEndian.Uint32(b[:4])
	}

	return nil
}

// Serialize serializes ExpandedNodeID into bytes.
func (e *ExpandedNodeID) Serialize() ([]byte, error) {
	b := make([]byte, e.Len())
	if err := e.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes ExpandedNodeID into bytes.
func (e *ExpandedNodeID) SerializeTo(b []byte) error {
	var offset = 0
	if err := e.NodeID.SerializeTo(b); err != nil {
		return err
	}
	offset += e.NodeID.Len()

	if e.HasNamespaceURI() {
		if err := e.NamespaceURI.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += e.NamespaceURI.Len()
	}

	if e.HasServerIndex() {
		binary.LittleEndian.PutUint32(b[offset:offset+4], e.ServerIndex)
		offset += 4
	}

	return nil
}

// Len returns the actual length of ExpandedNodeID in int.
func (e *ExpandedNodeID) Len() int {
	if e.NodeID == nil {
		return 0
	}

	l := e.NodeID.Len()
	if e.HasNamespaceURI() {
		l += e.NamespaceURI.Len()
	}
	if e.HasServerIndex() {
		l += 4
	}

	return l
}

// HasNamespaceURI checks if an ExpandedNodeID has NamespaceURI Flag.
func (e *ExpandedNodeID) HasNamespaceURI() bool {
	return e.NodeID.EncodingMaskValue()>>7&0x1 == 1
}

// HasServerIndex checks if an ExpandedNodeID has ServerIndex Flag.
func (e *ExpandedNodeID) HasServerIndex() bool {
	return e.NodeID.EncodingMaskValue()>>6&0x1 == 1
}
