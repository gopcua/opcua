// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"github.com/gopcua/opcua/datatypes"
	"github.com/gopcua/opcua/errors"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/services"
	"github.com/gopcua/opcua/ua"
)

// Message represents a OPC UA Secure Conversation message.
type Message struct {
	*Header
	*AsymmetricSecurityHeader
	*SymmetricSecurityHeader
	*SequenceHeader
	TypeID  *datatypes.ExpandedNodeID
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
			Header:                   NewHeader(MessageTypeOpenSecureChannel, ChunkTypeFinal, cfg.SecureChannelID),
			AsymmetricSecurityHeader: NewAsymmetricSecurityHeader(cfg.SecurityPolicyURI, cfg.Certificate, cfg.Thumbprint),
			SequenceHeader:           NewSequenceHeader(cfg.SequenceNumber, cfg.RequestID),
			TypeID:                   datatypes.NewFourByteExpandedNodeID(0, typeID),
			Service:                  srv,
		}

	case id.CloseSecureChannelRequest_Encoding_DefaultBinary, id.CloseSecureChannelResponse_Encoding_DefaultBinary:
		return &Message{
			Header:                  NewHeader(MessageTypeCloseSecureChannel, ChunkTypeFinal, cfg.SecureChannelID),
			SymmetricSecurityHeader: NewSymmetricSecurityHeader(cfg.SecurityTokenID),
			SequenceHeader:          NewSequenceHeader(cfg.SequenceNumber, cfg.RequestID),
			TypeID:                  datatypes.NewFourByteExpandedNodeID(0, typeID),
			Service:                 srv,
		}

	default:
		return &Message{
			Header:                  NewHeader(MessageTypeMessage, ChunkTypeFinal, cfg.SecureChannelID),
			SymmetricSecurityHeader: NewSymmetricSecurityHeader(cfg.SecurityTokenID),
			SequenceHeader:          NewSequenceHeader(cfg.SequenceNumber, cfg.RequestID),
			TypeID:                  datatypes.NewFourByteExpandedNodeID(0, typeID),
			Service:                 srv,
		}
	}
}

func (m *Message) Decode(b []byte) (int, error) {
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
		return buf.Pos(), errors.NewErrInvalidType(m, "decode", "should be one of OPN, MSG, CLO")
	}

	m.SequenceHeader = new(SequenceHeader)
	buf.ReadStruct(m.SequenceHeader)

	m.TypeID = new(datatypes.ExpandedNodeID)
	buf.ReadStruct(m.TypeID)

	if buf.Error() != nil {
		return buf.Pos(), buf.Error()
	}

	svc, err := services.Decode(m.TypeID, buf.Bytes())
	if err != nil {
		return 0, err
	}
	m.Service = svc
	return 0, nil
}

func (m *Message) Encode() ([]byte, error) {
	body := ua.NewBuffer(nil)
	switch m.Header.MessageType {
	case "OPN":
		body.WriteStruct(m.AsymmetricSecurityHeader)
	case "CLO", "MSG":
		body.WriteStruct(m.SymmetricSecurityHeader)
	default:
		return nil, errors.NewErrInvalidType(m, "serialize", "should be one of OPN, MSG, CLO")
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
