// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"fmt"

	"github.com/wmnsk/gopcua"
)

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
	buf := gopcua.NewBuffer(b)
	h.TokenID = buf.ReadUint32()
	return buf.Pos(), buf.Error()
}

func (h *SymmetricSecurityHeader) Encode() ([]byte, error) {
	buf := gopcua.NewBuffer(nil)
	buf.WriteUint32(h.TokenID)
	return buf.Bytes(), buf.Error()
}

// String returns Header in string.
func (s *SymmetricSecurityHeader) String() string {
	return fmt.Sprintf(
		"TokenID: %d",
		s.TokenID,
	)
}
