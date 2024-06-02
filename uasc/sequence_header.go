// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"fmt"
	"sync"

	"github.com/gopcua/opcua/ua"
)

func acquireSequenceHeader() *SequenceHeader {
	if v, ok := sequenceHeaderPool.Get().(*SequenceHeader); ok {
		return v
	}
	return &SequenceHeader{}
}

func releaseSequenceHeader(h *SequenceHeader) {
	h.RequestID = 0
	h.SequenceNumber = 0
	sequenceHeaderPool.Put(h)
}

var sequenceHeaderPool sync.Pool

// SequenceHeader represents a Sequence Header in OPC UA Secure Conversation.
type SequenceHeader struct {
	SequenceNumber uint32
	RequestID      uint32
}

// NewSequenceHeader creates a new OPC UA Secure Conversation Sequence Header.
func NewSequenceHeader(seq, req uint32) *SequenceHeader {
	return &SequenceHeader{
		SequenceNumber: seq,
		RequestID:      req,
	}
}

func (h *SequenceHeader) Decode(b []byte) (int, error) {
	buf := ua.NewBuffer(b)
	h.SequenceNumber = buf.ReadUint32()
	h.RequestID = buf.ReadUint32()
	return buf.Pos(), buf.Error()
}

// String returns Header in string.
func (s *SequenceHeader) String() string {
	return fmt.Sprintf(
		"SequenceNumber: %d, RequestID: %d",
		s.SequenceNumber,
		s.RequestID,
	)
}
