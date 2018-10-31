// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/errors"
	"github.com/wmnsk/gopcua/utils"
)

// MessageType definitions.
const (
	MessageTypeMessage            = "MSG"
	MessageTypeOpenSecureChannel  = "OPN"
	MessageTypeCloseSecureChannel = "CLO"
)

// ChunkType definitions.
const (
	ChunkTypeIntermediate = "C"
	ChunkTypeFinal        = "F"
	ChunkTypeError        = "A"
)

// Header represents a OPC UA Secure Conversation Header.
type Header struct {
	MessageType     uint32
	ChunkType       uint8
	MessageSize     uint32
	SecureChannelID uint32
	Payload         []byte
}

// NewHeader creates a new OPC UA Secure Conversation Header.
func NewHeader(msgType, chunkType string, chanID uint32, payload []byte) *Header {
	h := &Header{
		MessageType:     utils.Uint24To32([]byte(msgType)),
		ChunkType:       []byte(chunkType)[0],
		SecureChannelID: chanID,
		Payload:         payload,
	}

	h.SetLength()
	return h
}

// DecodeHeader decodes given bytes into OPC UA Secure Conversation Header.
func DecodeHeader(b []byte) (*Header, error) {
	h := &Header{}
	if err := h.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return h, nil
}

// DecodeFromBytes decodes given bytes into OPC UA Secure Conversation Header.
func (h *Header) DecodeFromBytes(b []byte) error {
	if len(b) < 12 {
		return errors.NewErrTooShortToDecode(h, "should be longer than 12 bytes")
	}

	h.MessageType = utils.Uint24To32(b[:3])
	h.ChunkType = b[3]
	h.MessageSize = binary.LittleEndian.Uint32(b[4:8])
	h.SecureChannelID = binary.LittleEndian.Uint32(b[8:12])
	if len(b[12:]) > 0 {
		h.Payload = b[12:]
	}

	return nil
}

// Serialize serializes OPC UA Secure Conversation Header into bytes.
func (h *Header) Serialize() ([]byte, error) {
	b := make([]byte, int(h.MessageSize))
	if err := h.SerializeTo(b); err != nil {
		return nil, err
	}
	return b, nil
}

// SerializeTo serializes OPC UA Secure Conversation Header into given bytes.
// TODO: add error handling.
func (h *Header) SerializeTo(b []byte) error {
	copy(b[:3], utils.Uint32To24(h.MessageType))
	b[3] = h.ChunkType
	binary.LittleEndian.PutUint32(b[4:8], h.MessageSize)
	binary.LittleEndian.PutUint32(b[8:12], h.SecureChannelID)
	copy(b[12:], h.Payload)

	return nil
}

// Len returns the actual length of Header in int.
func (h *Header) Len() int {
	return 12 + len(h.Payload)
}

// SetLength sets the length of Header.
func (h *Header) SetLength() {
	h.MessageSize = uint32(12 + len(h.Payload))
}

// MessageTypeValue returns MessageType in string.
func (h *Header) MessageTypeValue() string {
	if h == nil {
		return ""
	}

	x := make([]byte, 4)
	binary.BigEndian.PutUint32(x, h.MessageType)
	return string(x[1:])
}

// ChunkTypeValue returns ChunkType in string.
func (h *Header) ChunkTypeValue() string {
	if h == nil {
		return ""
	}

	return string(h.ChunkType)
}

// SecureChannelIDValue returns ChunkType in int.
func (h *Header) SecureChannelIDValue() int {
	if h == nil {
		return 0
	}

	return int(h.SecureChannelID)
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
