// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
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
