// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"fmt"
	"math"

	"github.com/gopcua/opcua/codec"
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

func (m *MessageAbort) Encode(s *ua.Stream) {
	s.WriteUint32(m.ErrorCode)
	s.WriteString(m.Reason)
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

func (m *Message) Encode(s *ua.Stream) {
	chunks, err := m.EncodeChunks(math.MaxUint32)
	if err != nil {
		return
	}
	s.Write(chunks[0])
}

func (m *Message) EncodeChunks(maxBodySize uint32) ([][]byte, error) {
	dataBody := ua.NewStream(ua.DefaultBufSize)
	dataBody.WriteAny(m.TypeID)
	dataBody.WriteAny(m.Service)
	if dataBody.Error() != nil {
		return nil, errors.Errorf("failed to encode databody: %s", dataBody.Error())
	}

	nrChunks := uint32(dataBody.Len())/(maxBodySize) + 1
	chunks := make([][]byte, nrChunks)

	switch m.Header.MessageType {
	case "OPN":
		partialHeader := ua.NewStream(ua.DefaultBufSize)
		partialHeader.WriteAny(m.AsymmetricSecurityHeader)
		partialHeader.WriteAny(m.SequenceHeader)
		if partialHeader.Error() != nil {
			return nil, errors.Errorf("failed to encode partial header: %s", partialHeader.Error())
		}

		m.Header.MessageSize = uint32(12 + partialHeader.Len() + dataBody.Len())
		buf := ua.NewStream(ua.DefaultBufSize)
		buf.WriteAny(m.Header)
		buf.Write(partialHeader.Bytes())
		buf.Write(dataBody.Bytes())
		if buf.Error() != nil {
			return nil, errors.Errorf("failed to encode chunk: %s", buf.Error())
		}
		for _, v := range buf.Bytes() {
			fmt.Printf("0x%02x,", v)
		}
		fmt.Println()
		return [][]byte{buf.Bytes()}, nil

	case "CLO", "MSG":
		chunk := ua.NewStream(ua.DefaultBufSize)
		for i := uint32(0); i < nrChunks-1; i++ {
			chunk.Reset()
			m.Header.MessageSize = maxBodySize + 24
			m.Header.ChunkType = ChunkTypeIntermediate
			chunk.WriteAny(m.Header)
			chunk.WriteAny(m.SymmetricSecurityHeader)
			chunk.WriteAny(m.SequenceHeader)
			chunk.Write(dataBody.ReadN(int(maxBodySize)))
			if chunk.Error() != nil {
				return nil, errors.Errorf("failed to encode chunk: %s", chunk.Error())
			}

			chunks[i] = append(chunks[i], chunk.Bytes()...)
		}

		m.Header.ChunkType = ChunkTypeFinal
		m.Header.MessageSize = uint32(24 + dataBody.Len())
		chunk.Reset()
		chunk.WriteAny(m.Header)
		chunk.WriteAny(m.SymmetricSecurityHeader)
		chunk.WriteAny(m.SequenceHeader)
		chunk.Write(dataBody.Bytes())
		if chunk.Error() != nil {
			return nil, errors.Errorf("failed to encode chunk: %s", chunk.Error())
		}

		chunks[nrChunks-1] = append(chunks[nrChunks-1], chunk.Bytes()...)
		return chunks, nil
	default:
		return nil, errors.Errorf("invalid message type %q", m.Header.MessageType)
	}
}

func (m *Message) MarshalOPCUA() ([]byte, error) {
	chunks, err := m.MarshalChunks(math.MaxUint32)
	if err != nil {
		return nil, err
	}

	return chunks[0], nil
}

func (m *Message) MarshalChunks(maxBodySize uint32) ([][]byte, error) {
	typeID, err := codec.Marshal(m.TypeID)
	if err != nil {
		return nil, errors.Errorf("failed to encode typeid: %s", err)
	}
	service, err := codec.Marshal(m.Service)
	if err != nil {
		return nil, errors.Errorf("failed to encode service: %s", err)
	}

	dataBody := make([]byte, len(typeID)+len(service))
	copy(dataBody, typeID)
	copy(dataBody[len(typeID):], service)

	nrChunks := uint32(len(dataBody))/(maxBodySize) + 1
	chunks := make([][]byte, nrChunks)

	switch m.Header.MessageType {
	case "OPN":
		asymmetricSecurityHeader, err := codec.Marshal(m.AsymmetricSecurityHeader)
		if err != nil {
			return nil, errors.Errorf("failed to encode asymmetric security header: %s", err)
		}
		sequenceHeader, err := codec.Marshal(m.SequenceHeader)
		if err != nil {
			return nil, errors.Errorf("failed to encode sequence header: %s", err)
		}

		m.Header.MessageSize = uint32(12 + len(asymmetricSecurityHeader) + len(sequenceHeader) + len(dataBody))
		header, err := codec.Marshal(m.Header)
		if err != nil {
			return nil, errors.Errorf("failed to encode header: %s", err)
		}
		chunks[0] = append(chunks[0], header...)
		chunks[0] = append(chunks[0], asymmetricSecurityHeader...)
		chunks[0] = append(chunks[0], sequenceHeader...)
		chunks[0] = append(chunks[0], dataBody...)
		return chunks, nil

	case "CLO", "MSG":
		symmetricSecurityHeader, err := codec.Marshal(m.SymmetricSecurityHeader)
		if err != nil {
			return nil, errors.Errorf("failed to encode symmetric security header: %s", err)
		}
		sequenceHeader, err := codec.Marshal(m.SequenceHeader)
		if err != nil {
			return nil, errors.Errorf("failed to encode sequence header: %s", err)
		}

		start, end := 0, int(maxBodySize)
		for i := uint32(0); i < nrChunks-1; i++ {
			m.Header.MessageSize = maxBodySize + 24
			m.Header.ChunkType = ChunkTypeIntermediate

			header, err := codec.Marshal(m.Header)
			if err != nil {
				return nil, errors.Errorf("failed to encode header: %s", err)
			}

			chunks[i] = append(chunks[i], header...)
			chunks[i] = append(chunks[i], symmetricSecurityHeader...)
			chunks[i] = append(chunks[i], sequenceHeader...)
			chunks[i] = append(chunks[i], dataBody[start:end]...)
			start, end = end, end+int(maxBodySize)
		}

		m.Header.ChunkType = ChunkTypeFinal
		m.Header.MessageSize = uint32(24 + len(dataBody))

		header, err := codec.Marshal(m.Header)
		if err != nil {
			return nil, err
		}
		chunks[nrChunks-1] = append(chunks[nrChunks-1], header...)
		chunks[nrChunks-1] = append(chunks[nrChunks-1], symmetricSecurityHeader...)
		chunks[nrChunks-1] = append(chunks[nrChunks-1], sequenceHeader...)
		chunks[nrChunks-1] = append(chunks[nrChunks-1], dataBody...)
		return chunks, nil
	default:
		return nil, errors.Errorf("invalid message type %q", m.Header.MessageType)
	}
}
