// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"fmt"
)

// Generic is an alias of OPC UA Header.
// This is implemented for handling undefined type of messages.
type Generic struct {
	Header  *Header
	Payload []byte
}

// NewGeneric creates a new OPC UA Generic.
func NewGeneric(hdr *Header, payload []byte) *Generic {
	return &Generic{hdr, payload}
}

// String returns Generic in string.
func (g *Generic) String() string {
	return fmt.Sprintf(
		"Header: %v",
		g.Header,
	)
}
