// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"fmt"
)

// Hello represents a OPC UA Hello.
//
// Specification: Part6, 7.1.2.3
type Hello struct {
	Version        uint32
	ReceiveBufSize uint32
	SendBufSize    uint32
	MaxMessageSize uint32
	MaxChunkCount  uint32
	EndPointURL    string
}

// NewHello creates a new OPC UA Hello.
func NewHello(ver, rcvBuf, sndBuf, maxMsg uint32, endpoint string) *Hello {
	return &Hello{
		Version:        ver,
		ReceiveBufSize: rcvBuf,
		SendBufSize:    sndBuf,
		MaxMessageSize: maxMsg,
		MaxChunkCount:  0,
		EndPointURL:    endpoint,
	}
}

// String returns Hello in string.
func (h *Hello) String() string {
	return fmt.Sprintf(
		"Version: %d, ReceiveBufSize: %d, SendBufSize: %d, MaxMessageSize: %d, MaxChunkCount: %d, EndPointURL: %s",
		h.Version,
		h.ReceiveBufSize,
		h.SendBufSize,
		h.MaxMessageSize,
		h.MaxChunkCount,
		h.EndPointURL,
	)
}
