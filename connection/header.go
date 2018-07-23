// Copyright 2018 gopc-ua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package connection

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/wmnsk/gopc-ua/utils"
)

// Header represents a OPC UA Connection Header.
type Header struct {
	MessageType uint32
	ChunkType   uint8
	MessageSize uint32
	Payload     []byte
}

// NewHeader creates a new OPC UA Connection Header.
func NewHeader(msgType, chunkType string, payload []byte) *Header {
	h := &Header{
		MessageType: utils.Uint24To32([]byte(msgType)),
		ChunkType:   []byte(chunkType)[0],
		Payload:     payload,
	}

	h.SetLength()
	return h
}

// DecodeHeader decodes given bytes into OPC UA Connection Header.
func DecodeHeader(b []byte) (*Header, error) {
	h := &Header{}
	if err := h.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return h, nil
}

// DecodeFromBytes decodes given bytes into OPC UA Header.
func (h *Header) DecodeFromBytes(b []byte) error {
	if len(b) < 8 {
		return errors.New("Too short to decode Header")
	}

	h.MessageType = utils.Uint24To32(b[:3])
	h.ChunkType = b[3]
	h.MessageSize = binary.LittleEndian.Uint32(b[4:8])
	h.Payload = b[8:]

	return nil
}

// Serialize serializes OPC UA Connection Header into bytes.
func (h *Header) Serialize() ([]byte, error) {
	b := make([]byte, int(h.MessageSize))
	if err := h.SerializeTo(b); err != nil {
		return nil, err
	}
	return b, nil
}

// SerializeTo serializes OPC UA Connection Header into given bytes.
// TODO: add error handling.
func (h *Header) SerializeTo(b []byte) error {
	copy(b[:3], utils.Uint32To24(h.MessageType))
	b[3] = h.ChunkType
	binary.LittleEndian.PutUint32(b[4:8], h.MessageSize)
	copy(b[8:], h.Payload)

	return nil
}

// MessageTypeString returns MessageType in string.
func (h *Header) MessageTypeString() string {
	if h == nil {
		return ""
	}

	x := make([]byte, 4)
	binary.BigEndian.PutUint32(x, h.MessageType)
	return string(x[1:])
}

// Len returns the actual length of Header in int.
func (h *Header) Len() int {
	return 8 + len(h.Payload)
}

// SetLength sets the length of Header.
func (h *Header) SetLength() {
	h.MessageSize = uint32(8 + len(h.Payload))
}

// ChunkTypeString returns ChunkType in string.
func (h *Header) ChunkTypeString() string {
	if h == nil {
		return ""
	}

	return string(h.ChunkType)
}

// String returns Header in string.
func (h *Header) String() string {
	return fmt.Sprintf(
		"MessageType: %d, ChunkType: %d, MessageSize: %d, Payload: %x,",
		h.MessageType,
		h.ChunkType,
		h.MessageSize,
		h.Payload,
	)
}
