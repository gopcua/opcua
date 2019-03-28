// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"fmt"

	"github.com/gopcua/opcua/id"
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
		return buf.Pos(), fmt.Errorf("invalid message type %q", m.Header.MessageType)
	}

	// Sequence Header could be encrypted; delay decoding until after decryption
	m.SequenceHeader = new(SequenceHeader)
	// buf.ReadStruct(m.SequenceHeader)

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

// Message represents a OPC UA Secure Conversation message.
type Message struct {
	*MessageHeader
	TypeID  *ua.ExpandedNodeID
	Service interface{}
}

// New creates a OPC UA Secure Conversation message.New
// MessageType of UASC is determined depending on the type of service given as below.
//
// Service type: OpenSecureChannel => Message type: OPN.
//
// Service type: CloseSecureChannel => Message type: CLO.
//
// Service type: Others => Message type: MSG.
//
// todo(fs): this feels wrong and we should move this switching into the secure channel.
func NewMessage(srv interface{}, typeID uint16, cfg *Config) *Message {
	switch typeID {
	case id.OpenSecureChannelRequest_Encoding_DefaultBinary, id.OpenSecureChannelResponse_Encoding_DefaultBinary:
		return &Message{
			MessageHeader: &MessageHeader{
				Header:                   NewHeader(MessageTypeOpenSecureChannel, ChunkTypeFinal, cfg.SecureChannelID),
				AsymmetricSecurityHeader: NewAsymmetricSecurityHeader(cfg.ServerEndpoint, cfg.LocalCertificate),
				SequenceHeader:           NewSequenceHeader(cfg.SequenceNumber, cfg.RequestID),
			},
			TypeID:  ua.NewFourByteExpandedNodeID(0, typeID),
			Service: srv,
		}

	case id.CloseSecureChannelRequest_Encoding_DefaultBinary, id.CloseSecureChannelResponse_Encoding_DefaultBinary:
		return &Message{
			MessageHeader: &MessageHeader{
				Header:                  NewHeader(MessageTypeCloseSecureChannel, ChunkTypeFinal, cfg.SecureChannelID),
				SymmetricSecurityHeader: NewSymmetricSecurityHeader(cfg.SecurityTokenID),
				SequenceHeader:          NewSequenceHeader(cfg.SequenceNumber, cfg.RequestID),
			},
			TypeID:  ua.NewFourByteExpandedNodeID(0, typeID),
			Service: srv,
		}

	default:
		return &Message{
			MessageHeader: &MessageHeader{
				Header:                  NewHeader(MessageTypeMessage, ChunkTypeFinal, cfg.SecureChannelID),
				SymmetricSecurityHeader: NewSymmetricSecurityHeader(cfg.SecurityTokenID),
				SequenceHeader:          NewSequenceHeader(cfg.SequenceNumber, cfg.RequestID),
			},

			TypeID:  ua.NewFourByteExpandedNodeID(0, typeID),
			Service: srv,
		}
	}
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

func (m *Message) Encode() ([]byte, error) {
	body := ua.NewBuffer(nil)
	switch m.Header.MessageType {
	case "OPN":
		body.WriteStruct(m.AsymmetricSecurityHeader)
	case "CLO", "MSG":
		body.WriteStruct(m.SymmetricSecurityHeader)
	default:
		return nil, fmt.Errorf("invalid message type %q", m.Header.MessageType)
	}
	body.WriteStruct(m.SequenceHeader)
	body.WriteStruct(m.TypeID)
	body.WriteStruct(m.Service)
	if body.Error() != nil {
		return nil, body.Error()
	}

	m.Header.MessageSize = uint32(12 + body.Len())
	buf := ua.NewBuffer(nil)
	buf.WriteStruct(m.Header)
	buf.Write(body.Bytes())
	return buf.Bytes(), buf.Error()
}
