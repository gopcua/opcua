// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"github.com/wmnsk/gopcua"
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
func NewMessage(srv services.Service, cfg *Config) *Message {
	switch srv.ServiceType() {
	case services.ServiceTypeOpenSecureChannelRequest,
		services.ServiceTypeOpenSecureChannelResponse:
		return &Message{
			Header:                   NewHeader(MessageTypeOpenSecureChannel, ChunkTypeFinal, cfg.SecureChannelID),
			AsymmetricSecurityHeader: NewAsymmetricSecurityHeader(cfg.SecurityPolicyURI, cfg.Certificate, cfg.Thumbprint),
			SequenceHeader:           NewSequenceHeader(cfg.SequenceNumber, cfg.RequestID),
			Service:                  srv,
		}

	case services.ServiceTypeCloseSecureChannelRequest,
		services.ServiceTypeCloseSecureChannelResponse:
		return &Message{
			Header:                  NewHeader(MessageTypeCloseSecureChannel, ChunkTypeFinal, cfg.SecureChannelID),
			SymmetricSecurityHeader: NewSymmetricSecurityHeader(cfg.SecurityTokenID),
			SequenceHeader:          NewSequenceHeader(cfg.SequenceNumber, cfg.RequestID),
			Service:                 srv,
		}

	default:
		return &Message{
			Header:                  NewHeader(MessageTypeMessage, ChunkTypeFinal, cfg.SecureChannelID),
			SymmetricSecurityHeader: NewSymmetricSecurityHeader(cfg.SecurityTokenID),
			SequenceHeader:          NewSequenceHeader(cfg.SequenceNumber, cfg.RequestID),
			Service:                 srv,
		}
	}
}

func (m *Message) Decode(b []byte) (int, error) {
	buf := gopcua.NewBuffer(b)

	m.Header = new(Header)
	buf.ReadStruct(m.Header)

	switch m.Header.MessageType {
	case "OPN":
		m.AsymmetricSecurityHeader = &AsymmetricSecurityHeader{}
		buf.ReadStruct(m.AsymmetricSecurityHeader)

	case "MSG", "CLO":
		m.SymmetricSecurityHeader = &SymmetricSecurityHeader{}
		buf.ReadStruct(m.SymmetricSecurityHeader)

	default:
		return buf.Pos(), errors.NewErrInvalidType(m, "decode", "should be one of OPN, MSG, CLO")
	}

	m.SequenceHeader = &SequenceHeader{}
	buf.ReadStruct(m.SequenceHeader)
	if buf.Error() != nil {
		return buf.Pos(), buf.Error()
	}

	svc, err := services.Decode(buf.Bytes())
	if err != nil {
		return 0, err
	}
	m.Service = svc
	return 0, nil
}

func (m *Message) Encode() ([]byte, error) {
	var sechdr []byte
	var err error
	switch m.Header.MessageType {
	case "OPN":
		sechdr, err = m.AsymmetricSecurityHeader.Encode()
	case "CLO", "MSG":
		sechdr, err = m.SymmetricSecurityHeader.Encode()
	default:
		return nil, errors.NewErrInvalidType(m, "serialize", "should be one of OPN, MSG, CLO")
	}
	if err != nil {
		return nil, err
	}

	seqhdr, err := m.SequenceHeader.Encode()
	if err != nil {
		return nil, err
	}

	svc, err := gopcua.Encode(m.Service)
	if err != nil {
		return nil, err
	}

	m.Header.MessageSize = uint32(12 + len(sechdr) + len(seqhdr) + len(svc))
	buf := gopcua.NewBuffer(nil)
	buf.WriteStruct(m.Header)
	buf.Write(sechdr)
	buf.Write(seqhdr)
	buf.Write(svc)
	return buf.Bytes(), buf.Error()
}

type OPNMessage struct {
	*AsymmetricSecurityHeader
	*SequenceHeader
	Service services.Service
}

func (m *OPNMessage) Decode(b []byte) error {
	m.AsymmetricSecurityHeader = new(AsymmetricSecurityHeader)
	m.SequenceHeader = new(SequenceHeader)

	var err error
	buf := gopcua.NewBuffer(b)
	buf.ReadStruct(m.AsymmetricSecurityHeader)
	buf.ReadStruct(m.SequenceHeader)
	m.Service, err = services.Decode(buf.Bytes())
	return err
}

func (m *OPNMessage) Encode() ([]byte, error) {
	svc, err := gopcua.Encode(m.Service)
	if err != nil {
		return nil, err
	}

	buf := gopcua.NewBuffer(nil)
	buf.WriteStruct(m.AsymmetricSecurityHeader)
	buf.WriteStruct(m.SequenceHeader)
	buf.Write(svc)
	return buf.Bytes(), buf.Error()
}

type CLOMessage struct {
	*SymmetricSecurityHeader
	*SequenceHeader
	Service services.Service
}

func (m *CLOMessage) Decode(b []byte) error {
	m.SymmetricSecurityHeader = new(SymmetricSecurityHeader)
	m.SequenceHeader = new(SequenceHeader)

	var err error
	buf := gopcua.NewBuffer(b)
	buf.ReadStruct(m.SymmetricSecurityHeader)
	buf.ReadStruct(m.SequenceHeader)
	m.Service, err = services.Decode(buf.Bytes())
	return err
}

func (m *CLOMessage) Encode() ([]byte, error) {
	svc, err := gopcua.Encode(m.Service)
	if err != nil {
		return nil, err
	}

	buf := gopcua.NewBuffer(nil)
	buf.WriteStruct(m.SymmetricSecurityHeader)
	buf.WriteStruct(m.SequenceHeader)
	buf.Write(svc)
	return buf.Bytes(), buf.Error()
}

type MSGMessage struct {
	*SymmetricSecurityHeader
	*SequenceHeader
	Service services.Service
}

func (m *MSGMessage) Decode(b []byte) error {
	m.SymmetricSecurityHeader = new(SymmetricSecurityHeader)
	m.SequenceHeader = new(SequenceHeader)

	var err error
	buf := gopcua.NewBuffer(b)
	buf.ReadStruct(m.SymmetricSecurityHeader)
	buf.ReadStruct(m.SequenceHeader)
	m.Service, err = services.Decode(buf.Bytes())
	return err
}

func (m *MSGMessage) Encode() ([]byte, error) {
	svc, err := gopcua.Encode(m.Service)
	if err != nil {
		return nil, err
	}

	buf := gopcua.NewBuffer(nil)
	buf.WriteStruct(m.SymmetricSecurityHeader)
	buf.WriteStruct(m.SequenceHeader)
	buf.Write(svc)
	return buf.Bytes(), buf.Error()
}
