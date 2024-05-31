// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"bytes"
	"fmt"

	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/ua"
)

// MessageType definitions.
const (
	MessageTypeMessage            = "MSG"
	MessageTypeOpenSecureChannel  = "OPN"
	MessageTypeCloseSecureChannel = "CLO"
)

// ChunkType definitions.
const (
	ChunkTypeIntermediate = 'C'
	ChunkTypeFinal        = 'F'
	ChunkTypeError        = 'A'
)

// Header represents a OPC UA Secure Conversation Header.
type Header struct {
	MessageType     string
	ChunkType       byte
	MessageSize     uint32
	SecureChannelID uint32
}

// NewHeader creates a new OPC UA Secure Conversation Header.
func NewHeader(msgType string, chunkType byte, chanID uint32) *Header {
	return &Header{
		MessageType:     msgType,
		ChunkType:       chunkType,
		SecureChannelID: chanID,
	}
}

func (h *Header) Decode(b []byte) (int, error) {
	buf := ua.NewBuffer(b)
	h.MessageType = string(buf.ReadN(3))
	h.ChunkType = buf.ReadByte()
	h.MessageSize = buf.ReadUint32()
	h.SecureChannelID = buf.ReadUint32()
	return buf.Pos(), buf.Error()
}

func (h *Header) Encode(s *ua.Stream) {
	if len(h.MessageType) != 3 {
		s.WrapError(errors.Errorf("invalid message type: %q", h.MessageType))
		return
	}
	s.Write([]byte(h.MessageType))
	s.WriteByte(h.ChunkType)
	s.WriteUint32(h.MessageSize)
	s.WriteUint32(h.SecureChannelID)
}

func (h *Header) MarshalOPCUA() ([]byte, error) {
	if len(h.MessageType) != 3 {
		return nil, errors.Errorf("invalid message type: %q", h.MessageType)
	}

	var buf bytes.Buffer
	buf.WriteString(h.MessageType)
	buf.WriteByte(h.ChunkType)
	buf.Write([]byte{byte(h.MessageSize), byte(h.MessageSize >> 8), byte(h.MessageSize >> 16), byte(h.MessageSize >> 24)})
	buf.Write([]byte{byte(h.SecureChannelID), byte(h.SecureChannelID >> 8), byte(h.SecureChannelID >> 16), byte(h.SecureChannelID >> 24)})
	return buf.Bytes(), nil
}

// String returns Header in string.
func (h *Header) String() string {
	return fmt.Sprintf(
		"MessageType: %s, ChunkType: %c, MessageSize: %d, SecureChannelID: %d",
		h.MessageType,
		h.ChunkType,
		h.MessageSize,
		h.SecureChannelID,
	)
}
