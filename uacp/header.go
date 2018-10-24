// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"fmt"

	"github.com/wmnsk/gopcua"
)

// MessageType definitions.
//
// Specification: Part 6, 7.1.2.2
const (
	MessageTypeHello        = "HEL"
	MessageTypeAcknowledge  = "ACK"
	MessageTypeError        = "ERR"
	MessageTypeReverseHello = "RHE"
)

// ChunkType definitions.
//
// Specification: Part 6, 6.7.2.2
const (
	ChunkTypeIntermediate = 'C'
	ChunkTypeFinal        = 'F'
	ChunkTypeError        = 'A'
)

// Header represents a OPC UA Connection Header.
//
// Specification: Part 6, 7.1.2.2
type Header struct {
	MessageType string
	ChunkType   byte
	MessageSize uint32
}

func (h *Header) Decode(b []byte) (int, error) {
	buf := gopcua.NewBuffer(b)
	h.MessageType = string(buf.ReadN(3))
	h.ChunkType = buf.ReadByte()
	h.MessageSize = buf.ReadUint32()
	return buf.Pos(), buf.Error()
}

func (h *Header) Encode() ([]byte, error) {
	buf := gopcua.NewBuffer(nil)
	if len(h.MessageType) != 3 {
		return nil, fmt.Errorf("invalid message type: %q", h.MessageType)
	}
	buf.Write([]byte(h.MessageType))
	buf.WriteByte(h.ChunkType)
	buf.WriteUint32(h.MessageSize)
	return buf.Bytes(), buf.Error()
}

// String returns Header in string.
func (h *Header) String() string {
	return fmt.Sprintf(
		"MessageType: %s, ChunkType: %c, MessageSize: %d",
		h.MessageType,
		h.ChunkType,
		h.MessageSize,
	)
}
