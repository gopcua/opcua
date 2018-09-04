// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/errors"
)

// Hello represents a OPC UA Hello.
//
// Specification: Part6, 7.1.2.3
type Hello struct {
	*Header
	Version        uint32
	ReceiveBufSize uint32
	SendBufSize    uint32
	MaxMessageSize uint32
	MaxChunkCount  uint32
	EndPointURL    *datatypes.String
}

// NewHello creates a new OPC UA Hello.
func NewHello(ver, rcvBuf, sndBuf, maxMsg uint32, endpoint string) *Hello {
	h := &Hello{
		Header: NewHeader(
			MessageTypeHello,
			ChunkTypeFinal,
			nil,
		),
		Version:        ver,
		ReceiveBufSize: rcvBuf,
		SendBufSize:    sndBuf,
		MaxMessageSize: maxMsg,
		MaxChunkCount:  0,
		EndPointURL:    datatypes.NewString(endpoint),
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
	if len(b) < 24 {
		return errors.NewErrTooShortToDecode(h, "should be longer than 24 bytes")
	}

	h.Header, err = DecodeHeader(b)
	if err != nil {
		return err
	}
	b = h.Header.Payload

	h.Version = binary.LittleEndian.Uint32(b[:4])
	h.ReceiveBufSize = binary.LittleEndian.Uint32(b[4:8])
	h.SendBufSize = binary.LittleEndian.Uint32(b[8:12])
	h.MaxMessageSize = binary.LittleEndian.Uint32(b[12:16])
	h.MaxChunkCount = binary.LittleEndian.Uint32(b[16:20])

	h.EndPointURL = &datatypes.String{}
	return h.EndPointURL.DecodeFromBytes(b[20:])
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
func (h *Hello) SerializeTo(b []byte) error {
	if h == nil {
		return errors.NewErrReceiverNil(h)
	}
	h.Header.Payload = make([]byte, h.Len()-8)

	binary.LittleEndian.PutUint32(h.Header.Payload[:4], h.Version)
	binary.LittleEndian.PutUint32(h.Header.Payload[4:8], h.ReceiveBufSize)
	binary.LittleEndian.PutUint32(h.Header.Payload[8:12], h.SendBufSize)
	binary.LittleEndian.PutUint32(h.Header.Payload[12:16], h.MaxMessageSize)
	binary.LittleEndian.PutUint32(h.Header.Payload[16:20], h.MaxChunkCount)

	if h.EndPointURL != nil {
		if err := h.EndPointURL.SerializeTo(h.Header.Payload[20:]); err != nil {
			return err
		}
	}

	h.Header.SetLength()
	return h.Header.SerializeTo(b)
}

// Len returns the actual length of Hello in int.
func (h *Hello) Len() int {
	return 28 + h.EndPointURL.Len()
}

// SetLength sets the length of Hello.
func (h *Hello) SetLength() {
	h.MessageSize = uint32(28 + h.EndPointURL.Len())
}

// String returns Hello in string.
func (h *Hello) String() string {
	return fmt.Sprintf(
		"Header: %v, Version: %d, ReceiveBufSize: %d, SendBufSize: %d, MaxMessageSize: %d, MaxChunkCount: %d, EndPointURL: %s",
		h.Header,
		h.Version,
		h.ReceiveBufSize,
		h.SendBufSize,
		h.MaxMessageSize,
		h.MaxChunkCount,
		h.EndPointURL.Get(),
	)
}
