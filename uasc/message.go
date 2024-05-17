// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"math"

	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/ua"
)

type MessageHeader struct {
	*Header
	*AsymmetricSecurityHeader
	*SymmetricSecurityHeader
	*SequenceHeader
}

func (m *MessageHeader) Decode(b []byte) (int, error) {
	buf := ua.NewBuffer(b)

	m.Header = new(Header)
	buf.ReadStruct(m.Header)

	switch m.Header.MessageType {
	case "OPN":
		m.AsymmetricSecurityHeader = new(AsymmetricSecurityHeader)
		buf.ReadStruct(m.AsymmetricSecurityHeader)

	case "MSG", "CLO":
		m.SymmetricSecurityHeader = new(SymmetricSecurityHeader)
		buf.ReadStruct(m.SymmetricSecurityHeader)

	default:
		return buf.Pos(), errors.Errorf("invalid message type %q", m.Header.MessageType)
	}

	// Sequence header could be encrypted, defer decoding until after decryption
	m.SequenceHeader = new(SequenceHeader)
	//buf.ReadStruct(m.SequenceHeader)

	return buf.Pos(), buf.Error()
}

type MessageChunk struct {
	*MessageHeader
	Data []byte
}

func (m *MessageChunk) Decode(b []byte) (int, error) {
	m.MessageHeader = new(MessageHeader)
	n, err := m.MessageHeader.Decode(b)
	if err != nil {
		return n, err
	}
	m.Data = b[n:]
	return len(b), nil
}

// MessageAbort represents a non-terminal OPC UA Secure Channel error.
//
// Specification: Part6, 7.3
type MessageAbort struct {
	ErrorCode uint32
	Reason    string
}

func (m *MessageAbort) Decode(b []byte) (int, error) {
	buf := ua.NewBuffer(b)
	m.ErrorCode = buf.ReadUint32()
	m.Reason = buf.ReadString()
	return buf.Pos(), buf.Error()
}

func (m *MessageAbort) Encode(s *ua.Stream) error {
	s.WriteUint32(m.ErrorCode)
	s.WriteString(m.Reason)
	return s.Error()
}

func (m *MessageAbort) MessageAbort() string {
	return ua.StatusCode(m.ErrorCode).Error()
}

// Message represents a OPC UA Secure Conversation message.
type Message struct {
	*MessageHeader
	TypeID  *ua.ExpandedNodeID
	Service interface{}
}

func (m *Message) Decode(b []byte) (int, error) {
	m.MessageHeader = new(MessageHeader)
	var pos int
	n, err := m.MessageHeader.Decode(b)
	if err != nil {
		return n, err
	}
	pos += n

	m.SequenceHeader = new(SequenceHeader)
	n, err = m.SequenceHeader.Decode(b[pos:])
	if err != nil {
		return n, err
	}
	pos += n

	m.TypeID, m.Service, err = ua.DecodeService(b[pos:])
	return len(b), err
}

func (m *Message) Encode(s *ua.Stream) error {
	chunks, err := m.EncodeChunks(math.MaxUint32)
	if err != nil {
		return err
	}
	s.Write(chunks[0])
	return nil
}

func (m *Message) EncodeChunks(maxBodySize uint32) ([][]byte, error) {
	dataBody := ua.NewStream(ua.DefaultBufSize)
	dataBody.WriteStruct(m.TypeID)
	dataBody.WriteStruct(m.Service)

	if dataBody.Error() != nil {
		return nil, dataBody.Error()
	}

	nrChunks := uint32(dataBody.Len())/(maxBodySize) + 1
	chunks := make([][]byte, nrChunks)

	switch m.Header.MessageType {
	case "OPN":
		partialHeader := ua.NewStream(ua.DefaultBufSize)
		partialHeader.WriteStruct(m.AsymmetricSecurityHeader)
		partialHeader.WriteStruct(m.SequenceHeader)

		if partialHeader.Error() != nil {
			return nil, partialHeader.Error()
		}

		m.Header.MessageSize = uint32(12 + partialHeader.Len() + dataBody.Len())
		buf := ua.NewStream(ua.DefaultBufSize)
		buf.WriteStruct(m.Header)
		buf.Write(partialHeader.Bytes())
		buf.Write(dataBody.Bytes())

		return [][]byte{buf.Bytes()}, buf.Error()

	case "CLO", "MSG":

		for i := uint32(0); i < nrChunks-1; i++ {
			m.Header.MessageSize = maxBodySize + 24
			m.Header.ChunkType = ChunkTypeIntermediate
			chunk := ua.NewStream(ua.DefaultBufSize)
			chunk.WriteStruct(m.Header)
			chunk.WriteStruct(m.SymmetricSecurityHeader)
			chunk.WriteStruct(m.SequenceHeader)
			chunk.Write(dataBody.ReadN(int(maxBodySize)))
			if chunk.Error() != nil {
				return nil, chunk.Error()
			}

			chunks[i] = chunk.Bytes()
		}

		m.Header.ChunkType = ChunkTypeFinal
		m.Header.MessageSize = uint32(24 + dataBody.Len())
		chunk := ua.NewStream(ua.DefaultBufSize)
		chunk.WriteStruct(m.Header)
		chunk.WriteStruct(m.SymmetricSecurityHeader)
		chunk.WriteStruct(m.SequenceHeader)
		chunk.Write(dataBody.Bytes())
		if chunk.Error() != nil {
			return nil, chunk.Error()
		}

		chunks[nrChunks-1] = chunk.Bytes()
		return chunks, nil
	default:
		return nil, errors.Errorf("invalid message type %q", m.Header.MessageType)
	}
}
