// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"fmt"
)

// Acknowledge represents a OPC UA Acknowledge.
//
// Specification: Part6, 7.1.2.4
type Acknowledge struct {
	Version        uint32
	ReceiveBufSize uint32
	SendBufSize    uint32
	MaxMessageSize uint32
	MaxChunkCount  uint32
}

// NewAcknowledge creates a new OPC UA Acknowledge.
func NewAcknowledge(ver, rcvBuf, sndBuf, maxMsg uint32) *Acknowledge {
	return &Acknowledge{
		Version:        ver,
		ReceiveBufSize: rcvBuf,
		SendBufSize:    sndBuf,
		MaxMessageSize: maxMsg,
		MaxChunkCount:  0,
	}
}

// String returns Acknowledge in string.
func (a *Acknowledge) String() string {
	return fmt.Sprintf(
		"Version: %d, ReceiveBufSize: %d, SendBufSize: %d, MaxMessageSize: %d, MaxChunkCount: %d",
		a.Version,
		a.ReceiveBufSize,
		a.SendBufSize,
		a.MaxMessageSize,
		a.MaxChunkCount,
	)
}
