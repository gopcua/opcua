// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Acknowledge represents a OPC UA Acknowledge.
type Acknowledge struct {
	*Header
	Version        uint32
	SendBufSize    uint32
	ReceiveBufSize uint32
	MaxMessageSize uint32
	MaxChunkCount  uint32
}

// NewAcknowledge creates a new OPC UA Acknowledge.
func NewAcknowledge(ver, sndBuf, rcvBuf, maxMsg uint32) *Acknowledge {
	h := &Acknowledge{
		Header: NewHeader(
			MessageTypeAcknowledge,
			ChunkTypeFinal,
			nil,
		),
		Version:        ver,
		SendBufSize:    sndBuf,
		ReceiveBufSize: rcvBuf,
		MaxMessageSize: maxMsg,
		MaxChunkCount:  0,
	}
	h.SetLength()

	return h
}

// DecodeAcknowledge decodes given bytes into OPC UA Acknowledge.
func DecodeAcknowledge(b []byte) (*Acknowledge, error) {
	h := &Acknowledge{}
	if err := h.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return h, nil
}

// DecodeFromBytes decodes given bytes into OPC UA Acknowledge.
func (h *Acknowledge) DecodeFromBytes(b []byte) error {
	var err error
	if len(b) < 20 {
		return errors.New("Too short to decode Acknowledge")
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

	return nil
}

// Serialize serializes OPC UA Acknowledge into bytes.
func (h *Acknowledge) Serialize() ([]byte, error) {
	b := make([]byte, int(h.MessageSize))
	if err := h.SerializeTo(b); err != nil {
		return nil, err
	}
	return b, nil
}

// SerializeTo serializes OPC UA Acknowledge into given bytes.
// TODO: add error handling.
func (h *Acknowledge) SerializeTo(b []byte) error {
	if h == nil {
		return errors.New("Acknowledge is nil")
	}
	h.Header.Payload = make([]byte, h.Len()-8)

	binary.LittleEndian.PutUint32(h.Header.Payload[:4], h.Version)
	binary.LittleEndian.PutUint32(h.Header.Payload[4:8], h.SendBufSize)
	binary.LittleEndian.PutUint32(h.Header.Payload[8:12], h.ReceiveBufSize)
	binary.LittleEndian.PutUint32(h.Header.Payload[12:16], h.MaxMessageSize)
	binary.LittleEndian.PutUint32(h.Header.Payload[16:20], h.MaxChunkCount)

	h.Header.SetLength()
	return h.Header.SerializeTo(b)
}

// Len returns the actual length of Acknowledge in int.
func (h *Acknowledge) Len() int {
	return 28
}

// SetLength sets the length of Acknowledge.
func (h *Acknowledge) SetLength() {
	h.MessageSize = 28
}

// String returns Acknowledge in string.
func (h *Acknowledge) String() string {
	return fmt.Sprintf(
		"Header: %v, Version: %d, SendBufSize: %d, ReceiveBufSize: %d, MaxMessageSize: %d, MaxChunkCount: %d",
		h.Header,
		h.Version,
		h.SendBufSize,
		h.ReceiveBufSize,
		h.MaxMessageSize,
		h.MaxChunkCount,
	)
}
