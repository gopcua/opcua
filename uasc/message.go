// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"github.com/wmnsk/gopcua/errors"
	"github.com/wmnsk/gopcua/services"
)

// Message represents a OPC UA Secure Conversation message.
type Message struct {
	*Header
	*AsymmetricSecurityHeader
	*SymmetricSecurityHeader
	*SequenceHeader
	Service services.Service
}

// Config represents a configuration which UASC client/server has in common.
type Config struct {
	SecureChannelID   uint32
	SecurityPolicyURI string
	Certificate       []byte
	Thumbprint        []byte
	RequestID         uint32
	SecurityTokenID   uint32
	CurrentSequence   uint32
	SecurityMode      uint32
	ServerCertificate []byte
	ServerURI         string
}

// New creates a OPC UA Secure Conversation message, depending on the type of service given.
func New(srv services.Service, cfg *Config) *Message {
	switch srv.ServiceType() {
	case services.ServiceTypeGetEndpointsRequest:
		gep, ok := srv.(*services.GetEndpointsRequest)
		if !ok {
			return nil
		}
		return newGetEndpointsRequest(gep, cfg)
	case services.ServiceTypeGetEndpointsResponse:
		gep, ok := srv.(*services.GetEndpointsResponse)
		if !ok {
			return nil
		}
		return newGetEndpointsResponse(gep, cfg)
	case services.ServiceTypeOpenSecureChannelRequest:
		osc, ok := srv.(*services.OpenSecureChannelRequest)
		if !ok {
			return nil
		}
		return newOpenSecureChannelRequest(osc, cfg)
	case services.ServiceTypeOpenSecureChannelResponse:
		osc, ok := srv.(*services.OpenSecureChannelResponse)
		if !ok {
			return nil
		}
		return newOpenSecureChannelResponse(osc, cfg)
	default:
		return nil
	}
}

func newGetEndpointsRequest(gep *services.GetEndpointsRequest, cfg *Config) *Message {
	m := &Message{
		Header:                  NewHeader(MessageTypeMessage, ChunkTypeFinal, cfg.SecureChannelID, nil),
		SymmetricSecurityHeader: NewSymmetricSecurityHeader(cfg.SecurityTokenID, nil),
		SequenceHeader: NewSequenceHeader(
			cfg.CurrentSequence, cfg.RequestID, nil,
		),
		Service: gep,
	}

	return m
}

func newGetEndpointsResponse(gep *services.GetEndpointsResponse, cfg *Config) *Message {
	m := &Message{
		Header:                  NewHeader(MessageTypeMessage, ChunkTypeFinal, cfg.SecureChannelID, nil),
		SymmetricSecurityHeader: NewSymmetricSecurityHeader(cfg.SecurityTokenID, nil),
		SequenceHeader: NewSequenceHeader(
			cfg.CurrentSequence, cfg.RequestID, nil,
		),
		Service: gep,
	}

	return m
}

func newOpenSecureChannelRequest(osc *services.OpenSecureChannelRequest, cfg *Config) *Message {
	m := &Message{
		Header: NewHeader(MessageTypeOpenSecureChannel, ChunkTypeFinal, cfg.SecureChannelID, nil),
		AsymmetricSecurityHeader: NewAsymmetricSecurityHeader(
			cfg.SecurityPolicyURI, cfg.Certificate, cfg.Thumbprint, nil,
		),
		SequenceHeader: NewSequenceHeader(
			cfg.CurrentSequence, cfg.RequestID, nil,
		),
		Service: osc,
	}

	return m
}

func newOpenSecureChannelResponse(osc *services.OpenSecureChannelResponse, cfg *Config) *Message {
	m := &Message{
		Header: NewHeader(MessageTypeOpenSecureChannel, ChunkTypeFinal, cfg.SecureChannelID, nil),
		AsymmetricSecurityHeader: NewAsymmetricSecurityHeader(
			cfg.SecurityPolicyURI, cfg.Certificate, cfg.Thumbprint, nil,
		),
		SequenceHeader: NewSequenceHeader(
			cfg.CurrentSequence, cfg.RequestID, nil,
		),
		Service: osc,
	}

	return m
}

// Decode decodes given bytes into OPC UA Secure Conversation message.
func Decode(b []byte) (*Message, error) {
	m := &Message{}
	if err := m.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return m, nil
}

// DecodeFromBytes decodes given bytes into OPC UA Secure Conversation message.
func (m *Message) DecodeFromBytes(b []byte) error {
	if len(b) < 32 {
		return &errors.ErrTooShortToDecode{m, "should be longer than 32 bytes"}
	}

	m.Header = &Header{}
	if err := m.Header.DecodeFromBytes(b); err != nil {
		return err
	}
	switch m.Header.MessageTypeValue() {
	case MessageTypeOpenSecureChannel, MessageTypeCloseSecureChannel:
		return m.decodeSecChanFromBytes(m.Header.Payload)
	case MessageTypeMessage:
		return m.decodeMSGFromBytes(m.Header.Payload)
	default:
		return &errors.ErrInvalidType{m, "decode", "should be one of OPN, MSG, CLO"}
	}
}

func (m *Message) decodeSecChanFromBytes(b []byte) error {
	m.AsymmetricSecurityHeader = &AsymmetricSecurityHeader{}
	if err := m.AsymmetricSecurityHeader.DecodeFromBytes(b); err != nil {
		return err
	}
	m.SequenceHeader = &SequenceHeader{}
	if err := m.SequenceHeader.DecodeFromBytes(m.AsymmetricSecurityHeader.Payload); err != nil {
		return err
	}

	var err error
	m.Service, err = services.Decode(m.SequenceHeader.Payload)
	if err != nil {
		return err
	}

	return nil
}

func (m *Message) decodeMSGFromBytes(b []byte) error {
	m.SymmetricSecurityHeader = &SymmetricSecurityHeader{}
	if err := m.SymmetricSecurityHeader.DecodeFromBytes(b); err != nil {
		return err
	}
	m.SequenceHeader = &SequenceHeader{}
	if err := m.SequenceHeader.DecodeFromBytes(m.SymmetricSecurityHeader.Payload); err != nil {
		return err
	}

	var err error
	m.Service, err = services.Decode(m.SequenceHeader.Payload)
	if err != nil {
		return err
	}

	return nil
}

// Serialize serializes Message into bytes.
func (m *Message) Serialize() ([]byte, error) {
	b := make([]byte, m.Len())
	if err := m.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes Message into bytes.
func (m *Message) SerializeTo(b []byte) error {
	var offset = 0
	if m.Header != nil {
		m.Header.MessageSize = uint32(m.Len())
		if err := m.Header.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += m.Header.Len() - len(m.Header.Payload)
	}
	switch m.Header.MessageTypeValue() {
	case MessageTypeOpenSecureChannel, MessageTypeCloseSecureChannel:
		return m.serializeSecChanTo(b[offset:])
	case MessageTypeMessage:
		return m.serializeMSGTo(b[offset:])
	default:
		return &errors.ErrInvalidType{m, "serialize", "should be one of OPN, MSG, CLO"}
	}
}

func (m *Message) serializeSecChanTo(b []byte) error {
	var offset = 0
	if m.AsymmetricSecurityHeader != nil {
		if err := m.AsymmetricSecurityHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += m.AsymmetricSecurityHeader.Len() - len(m.AsymmetricSecurityHeader.Payload)
	}
	if m.SequenceHeader != nil {
		if err := m.SequenceHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += m.SequenceHeader.Len() - len(m.SequenceHeader.Payload)
	}
	if m.Service != nil {
		if err := m.Service.SerializeTo(b[offset:]); err != nil {
			return err
		}
	}

	return nil
}

func (m *Message) serializeMSGTo(b []byte) error {
	var offset = 0
	if m.SymmetricSecurityHeader != nil {
		if err := m.SymmetricSecurityHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += m.SymmetricSecurityHeader.Len() - len(m.SymmetricSecurityHeader.Payload)
	}
	if m.SequenceHeader != nil {
		if err := m.SequenceHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += m.SequenceHeader.Len() - len(m.SequenceHeader.Payload)
	}
	if m.Service != nil {
		if err := m.Service.SerializeTo(b[offset:]); err != nil {
			return err
		}
	}

	return nil
}

// Len returns the actual length of Message.
func (m *Message) Len() int {
	var l = 0
	if m.Header != nil {
		l += m.Header.Len()
	}
	if m.AsymmetricSecurityHeader != nil {
		l += m.AsymmetricSecurityHeader.Len()
	}
	if m.SymmetricSecurityHeader != nil {
		l += m.SymmetricSecurityHeader.Len()
	}
	if m.SequenceHeader != nil {
		l += m.SequenceHeader.Len()
	}
	if m.Service != nil {
		l += m.Service.Len()
	}

	return l
}
