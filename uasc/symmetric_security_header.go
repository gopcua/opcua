// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"fmt"
	"sync"

	"github.com/gopcua/opcua/ua"
)

func acquireSymmetricSecurityHeader() *SymmetricSecurityHeader {
	v := symmetricSecurityHeaderPool.Get()
	if v == nil {
		return &SymmetricSecurityHeader{}
	}
	return v.(*SymmetricSecurityHeader)
}

func releaseSymmetricSecurityHeader(h *SymmetricSecurityHeader) {
	h.TokenID = 0
	symmetricSecurityHeaderPool.Put(h)
}

var symmetricSecurityHeaderPool sync.Pool

// SymmetricSecurityHeader represents a Symmetric Algorithm Security Header in OPC UA Secure Conversation.
type SymmetricSecurityHeader struct {
	TokenID uint32
}

// NewSymmetricSecurityHeader creates a new OPC UA Secure Conversation Symmetric Algorithm Security Header.
func NewSymmetricSecurityHeader(token uint32) *SymmetricSecurityHeader {
	return &SymmetricSecurityHeader{
		TokenID: token,
	}
}

func (h *SymmetricSecurityHeader) Decode(b []byte) (int, error) {
	buf := ua.NewBuffer(b)
	h.TokenID = buf.ReadUint32()
	return buf.Pos(), buf.Error()
}

// String returns Header in string.
func (h *SymmetricSecurityHeader) String() string {
	return fmt.Sprintf(
		"TokenID: %d",
		h.TokenID,
	)
}

// Len returns the Header Length in bytes.
func (h *SymmetricSecurityHeader) Len() int {
	return 4
}
