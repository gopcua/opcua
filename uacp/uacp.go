// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"bytes"
	"encoding/binary"

	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/ua"
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
	ChunkTypeAbort        = 'A'
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
	buf := ua.NewBuffer(b)
	h.MessageType = string(buf.ReadN(3))
	h.ChunkType = buf.ReadByte()
	h.MessageSize = buf.ReadUint32()
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
}

func (h *Header) MarshalOPCUA() ([]byte, error) {
	if len(h.MessageType) != 3 {
		return nil, errors.Errorf("invalid message type: %q", h.MessageType)
	}

	var buf bytes.Buffer
	buf.Write([]byte(h.MessageType))
	buf.WriteByte(h.ChunkType)
	buf.Write([]byte{byte(h.MessageSize), byte(h.MessageSize >> 8), byte(h.MessageSize >> 16), byte(h.MessageSize >> 24)})
	return buf.Bytes(), nil
}

// Hello represents a OPC UA Hello.
//
// Specification: Part6, 7.1.2.3
type Hello struct {
	Version        uint32
	ReceiveBufSize uint32
	SendBufSize    uint32
	MaxMessageSize uint32
	MaxChunkCount  uint32
	EndpointURL    string
}

func (h *Hello) Decode(b []byte) (int, error) {
	buf := ua.NewBuffer(b)
	h.Version = buf.ReadUint32()
	h.ReceiveBufSize = buf.ReadUint32()
	h.SendBufSize = buf.ReadUint32()
	h.MaxMessageSize = buf.ReadUint32()
	h.MaxChunkCount = buf.ReadUint32()
	h.EndpointURL = buf.ReadString()
	return buf.Pos(), buf.Error()
}

func (h *Hello) Encode(s *ua.Stream) {
	s.WriteUint32(h.Version)
	s.WriteUint32(h.ReceiveBufSize)
	s.WriteUint32(h.SendBufSize)
	s.WriteUint32(h.MaxMessageSize)
	s.WriteUint32(h.MaxChunkCount)
	s.WriteString(h.EndpointURL)
}

func (h *Hello) MarshalOPCUA() ([]byte, error) {
	var buf bytes.Buffer
	buf.Write([]byte{byte(h.Version), byte(h.Version >> 8), byte(h.Version >> 16), byte(h.Version >> 24)})
	buf.Write([]byte{byte(h.ReceiveBufSize), byte(h.ReceiveBufSize >> 8), byte(h.ReceiveBufSize >> 16), byte(h.ReceiveBufSize >> 24)})
	buf.Write([]byte{byte(h.SendBufSize), byte(h.SendBufSize >> 8), byte(h.SendBufSize >> 16), byte(h.SendBufSize >> 24)})
	buf.Write([]byte{byte(h.MaxMessageSize), byte(h.MaxMessageSize >> 8), byte(h.MaxMessageSize >> 16), byte(h.MaxMessageSize >> 24)})
	buf.Write([]byte{byte(h.MaxChunkCount), byte(h.MaxChunkCount >> 8), byte(h.MaxChunkCount >> 16), byte(h.MaxChunkCount >> 24)})
	if len(h.EndpointURL) == 0 {
		buf.Write([]byte{0xff, 0xff, 0xff, 0xff})
	} else {
		n := len(h.EndpointURL)
		buf.Write([]byte{byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24)})
		buf.WriteString(h.EndpointURL)
	}
	return buf.Bytes(), nil
}

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

func (a *Acknowledge) Decode(b []byte) (int, error) {
	buf := ua.NewBuffer(b)
	a.Version = buf.ReadUint32()
	a.ReceiveBufSize = buf.ReadUint32()
	a.SendBufSize = buf.ReadUint32()
	a.MaxMessageSize = buf.ReadUint32()
	a.MaxChunkCount = buf.ReadUint32()
	return buf.Pos(), buf.Error()
}

func (a *Acknowledge) Encode(s *ua.Stream) {
	s.WriteUint32(a.Version)
	s.WriteUint32(a.ReceiveBufSize)
	s.WriteUint32(a.SendBufSize)
	s.WriteUint32(a.MaxMessageSize)
	s.WriteUint32(a.MaxChunkCount)
}

func (a *Acknowledge) MarshalOPCUA() ([]byte, error) {
	buf := make([]byte, 0, 160)
	binary.LittleEndian.AppendUint32(buf, a.Version)
	binary.LittleEndian.AppendUint32(buf, a.ReceiveBufSize)
	binary.LittleEndian.AppendUint32(buf, a.SendBufSize)
	binary.LittleEndian.AppendUint32(buf, a.MaxMessageSize)
	binary.LittleEndian.AppendUint32(buf, a.MaxChunkCount)
	return buf, nil
}

// ReverseHello represents a OPC UA ReverseHello.
//
// Specification: Part6, 7.1.2.6
type ReverseHello struct {
	ServerURI   string
	EndpointURL string
}

func (r *ReverseHello) Decode(b []byte) (int, error) {
	buf := ua.NewBuffer(b)
	r.ServerURI = buf.ReadString()
	r.EndpointURL = buf.ReadString()
	return buf.Pos(), buf.Error()
}

func (r *ReverseHello) Encode(s *ua.Stream) {
	s.WriteString(r.ServerURI)
	s.WriteString(r.EndpointURL)
}

// Error represents a OPC UA Error.
//
// Specification: Part6, 7.1.2.5
type Error struct {
	ErrorCode uint32
	Reason    string
}

func (e *Error) Decode(b []byte) (int, error) {
	buf := ua.NewBuffer(b)
	e.ErrorCode = buf.ReadUint32()
	e.Reason = buf.ReadString()
	return buf.Pos(), buf.Error()
}

func (e *Error) Encode(s *ua.Stream) {
	s.WriteUint32(e.ErrorCode)
	s.WriteString(e.Reason)
}

func (e *Error) Error() string {
	return ua.StatusCode(e.ErrorCode).Error()
}

// Unwrap returns the wrapped error code.
func (e *Error) Unwrap() error {
	return ua.StatusCode(e.ErrorCode)
}

type Message struct {
	Data []byte
}

func (m *Message) Decode(b []byte) (int, error) {
	m.Data = b
	return len(b), nil
}

func (m *Message) Encode(s *ua.Stream) {
	s.Write(m.Data)
}
