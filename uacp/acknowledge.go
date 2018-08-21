// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/errors"
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
	a := &Acknowledge{
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
	a.SetLength()

	return a
}

// DecodeAcknowledge decodes given bytes into OPC UA Acknowledge.
func DecodeAcknowledge(b []byte) (*Acknowledge, error) {
	a := &Acknowledge{}
	if err := a.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return a, nil
}

// DecodeFromBytes decodes given bytes into OPC UA Acknowledge.
func (a *Acknowledge) DecodeFromBytes(b []byte) error {
	var err error
	if len(b) < 20 {
		return &errors.ErrTooShortToDecode{a, "should be longer than 20 bytes"}
	}

	a.Header, err = DecodeHeader(b)
	if err != nil {
		return err
	}
	b = a.Header.Payload

	a.Version = binary.LittleEndian.Uint32(b[:4])
	a.SendBufSize = binary.LittleEndian.Uint32(b[4:8])
	a.ReceiveBufSize = binary.LittleEndian.Uint32(b[8:12])
	a.MaxMessageSize = binary.LittleEndian.Uint32(b[12:16])
	a.MaxChunkCount = binary.LittleEndian.Uint32(b[16:20])

	return nil
}

// Serialize serializes OPC UA Acknowledge into bytes.
func (a *Acknowledge) Serialize() ([]byte, error) {
	b := make([]byte, int(a.MessageSize))
	if err := a.SerializeTo(b); err != nil {
		return nil, err
	}
	return b, nil
}

// SerializeTo serializes OPC UA Acknowledge into given bytes.
// TODO: add error handling.
func (a *Acknowledge) SerializeTo(b []byte) error {
	if a == nil {
		return &errors.ErrReceiverNil{a}
	}
	a.Header.Payload = make([]byte, a.Len()-8)

	binary.LittleEndian.PutUint32(a.Header.Payload[:4], a.Version)
	binary.LittleEndian.PutUint32(a.Header.Payload[4:8], a.SendBufSize)
	binary.LittleEndian.PutUint32(a.Header.Payload[8:12], a.ReceiveBufSize)
	binary.LittleEndian.PutUint32(a.Header.Payload[12:16], a.MaxMessageSize)
	binary.LittleEndian.PutUint32(a.Header.Payload[16:20], a.MaxChunkCount)

	a.Header.SetLength()
	return a.Header.SerializeTo(b)
}

// Len returns the actual length of Acknowledge in int.
func (a *Acknowledge) Len() int {
	return 28
}

// SetLength sets the length of Acknowledge.
func (a *Acknowledge) SetLength() {
	a.MessageSize = 28
}

// String returns Acknowledge in string.
func (a *Acknowledge) String() string {
	return fmt.Sprintf(
		"Header: %v, Version: %d, SendBufSize: %d, ReceiveBufSize: %d, MaxMessageSize: %d, MaxChunkCount: %d",
		a.Header,
		a.Version,
		a.SendBufSize,
		a.ReceiveBufSize,
		a.MaxMessageSize,
		a.MaxChunkCount,
	)
}
