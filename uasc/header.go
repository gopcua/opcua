// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"fmt"
	"sync"

	"github.com/gopcua/opcua/codec"
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

func acquireHeader() *Header {
	v := headerPool.Get()
	if v == nil {
		return &Header{}
	}
	return v.(*Header)
}

func releaseHeader(h *Header) {
	h.MessageType = ""
	h.MessageSize = 0
	h.SecureChannelID = 0
	headerPool.Put(h)
}

var headerPool sync.Pool

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

func (h *Header) EncodeOPCUA(buf *codec.Stream) error {
	if len(h.MessageType) != 3 {
		return errors.Errorf("invalid message type: %q", h.MessageType)
	}

	// var buf bytes.Buffer
	buf.WriteString(h.MessageType)
	buf.WriteByte(h.ChunkType)
	buf.Write([]byte{byte(h.MessageSize), byte(h.MessageSize >> 8), byte(h.MessageSize >> 16), byte(h.MessageSize >> 24)})
	buf.Write([]byte{byte(h.SecureChannelID), byte(h.SecureChannelID >> 8), byte(h.SecureChannelID >> 16), byte(h.SecureChannelID >> 24)})
	return nil
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
