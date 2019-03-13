// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"fmt"
)

// ReverseHello represents a OPC UA ReverseHello.
//
// Specification: Part6, 7.1.2.6
type ReverseHello struct {
	ServerURI   string
	EndPointURL string
}

// NewReverseHello creates a new OPC UA ReverseHello.
func NewReverseHello(serverURI, endpoint string) *ReverseHello {
	return &ReverseHello{
		ServerURI:   serverURI,
		EndPointURL: endpoint,
	}
}

// String returns ReverseHello in string.
func (r *ReverseHello) String() string {
	return fmt.Sprintf(
		"ServerURI: %s, EndPointURL: %s",
		r.ServerURI,
		r.EndPointURL,
	)
}
