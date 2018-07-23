// Copyright 2018 gopc-ua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package connection

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Hello represents a OPC UA Hello.
type Hello struct {
	*Header
	Version        uint32
	SendBufSize    uint32
	ReceiveBufSize uint32
	MaxMessageSize uint32
	MaxChunkCount  uint32
	EndPointURL    []byte
}

// NewHello creates a new OPC UA Hello.
func NewHello(ver, sndBuf, rcvBuf, maxMsg uint32, endpoint string) *Hello {
	h := &Hello{
		Header: NewHeader(
			"HEL",
			"F",
			nil,
		),
		Version:        ver,
		SendBufSize:    sndBuf,
		ReceiveBufSize: rcvBuf,
		MaxMessageSize: maxMsg,
		MaxChunkCount:  0,
		EndPointURL:    []byte(endpoint),
	}
	h.SetLength()

	return h
}

// DecodeHello decodes given bytes into OPC UA Hello.
func DecodeHello(b []byte) (*Hello, error) {
	h := &Hello{}
	if err := h.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return h, nil
}

// DecodeFromBytes decodes given bytes into OPC UA Hello.
func (h *Hello) DecodeFromBytes(b []byte) error {
	var err error
	if len(b) < 32 {
		return errors.New("Too short to decode Hello")
	}

	h.Header, err = DecodeHeader(b)
	if err != nil {
		return err
	}
	b = h.Header.Payload

	h.Version = binary.LittleEndian.Uint32(b[:4])
	h.SendBufSize = binary.LittleEndian.Uint32(b[4:8])
	h.ReceiveBufSize = binary.LittleEndian.Uint32(b[8:12])
	h.MaxMessageSize = binary.LittleEndian.Uint32(b[12:16])
	h.MaxChunkCount = binary.LittleEndian.Uint32(b[16:20])
	h.EndPointURL = b[24:]

	return nil
}

// Serialize serializes OPC UA Hello into bytes.
func (h *Hello) Serialize() ([]byte, error) {
	b := make([]byte, int(h.MessageSize))
	if err := h.SerializeTo(b); err != nil {
		return nil, err
	}
	return b, nil
}

// SerializeTo serializes OPC UA Hello into given bytes.
// TODO: add error handling.
func (h *Hello) SerializeTo(b []byte) error {
	if h == nil {
		return errors.New("Hello is nil")
	}
	h.Header.Payload = make([]byte, h.Len()-8)

	binary.LittleEndian.PutUint32(h.Header.Payload[4:8], h.SendBufSize)
	binary.LittleEndian.PutUint32(h.Header.Payload[8:12], h.ReceiveBufSize)
	binary.LittleEndian.PutUint32(h.Header.Payload[12:16], h.MaxMessageSize)
	binary.LittleEndian.PutUint32(h.Header.Payload[16:20], h.MaxChunkCount)
	copy(h.Header.Payload[20:24], []byte{0x2f, 0x00, 0x00, 0x00})
	copy(h.Header.Payload[24:], h.EndPointURL)

	h.Header.SetLength()
	return h.Header.SerializeTo(b)
}

// EndPointURLString returns EndPointURL in string
func (h *Hello) EndPointURLString() string {
	if h == nil {
		return ""
	}

	return string(h.EndPointURL)
}

// Len returns the actual length of Hello in int.
func (h *Hello) Len() int {
	return 32 + len(h.EndPointURL)
}

// SetLength sets the length of Hello.
func (h *Hello) SetLength() {
	h.MessageSize = uint32(32 + len(h.EndPointURL))
}

// String returns Hello in string.
func (h *Hello) String() string {
	return fmt.Sprintf(
		"MessageType: %d, ChunkType: %d, MessageSize: %d, Version: %d, SendBufSize: %d, ReceiveBufSize: %d, MaxMessageSize: %d, MaxChunkCount: %d, EndPointURL: %s",
		h.MessageType,
		h.ChunkType,
		h.MessageSize,
		h.Version,
		h.SendBufSize,
		h.ReceiveBufSize,
		h.MaxMessageSize,
		h.MaxChunkCount,
		h.EndPointURL,
	)
}
