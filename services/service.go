// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/errors"
)

// ServiceType definitions.
const (
	ServiceTypeOpenSecureChannelRequest  uint16 = 446
	ServiceTypeOpenSecureChannelResponse        = 449
)

// Service is an interface to handle any kind of OPC UA Services.
type Service interface {
	DecodeFromBytes([]byte) error
	Serialize() ([]byte, error)
	SerializeTo([]byte) error
	Len() int
	String() string
	ServiceType() uint16
}

// DecodeService decodes given bytes into Service, depending on the type of service.
func DecodeService(b []byte) (Service, error) {
	var s Service

	typeID, err := datatypes.DecodeExpandedNodeID(b)
	if err != nil {
		return nil, &errors.ErrUnsupported{typeID, "cannot decode TypeID"}
	}
	n, ok := typeID.NodeID.(*datatypes.FourByteNodeID)
	if !ok {
		return nil, &errors.ErrUnsupported{typeID.NodeID, "should be FourByteNodeID"}
	}

	switch n.Identifier {
	case ServiceTypeOpenSecureChannelRequest:
		s = &OpenSecureChannelRequest{}
	case ServiceTypeOpenSecureChannelResponse:
		s = &OpenSecureChannelResponse{}
	default:
		return nil, &errors.ErrUnsupported{n.Identifier, "unsupported type"}
	}

	if err := s.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return s, nil
}

/*
// Message represents a Message layer in OPC UA Binary Protocol.
type Message struct {
	TypeID *datatypes.ExpandedNodeID
	Service
}

// NewMessage creates a new Message.
func NewMessage(idx uint8, service Service) *Message {
	m := &Message{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(idx, service.ServiceType()),
			"", 0,
		),
		Service: service,
	}

	return m
}

// DecodeMessage decodes given bytes into Message.
func DecodeMessage(b []byte) (*Message, error) {
	m := &Message{}
	if err := m.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return m, nil
}

// DecodeFromBytes decodes given bytes into Message.
func (m *Message) DecodeFromBytes(b []byte) error {
	if len(b) < 2 {
		return &errors.ErrTooShortToDecode{m, "should be longer than 2 bytes"}
	}

	var offset = 0
	m.TypeID = &datatypes.ExpandedNodeID{}
	if err := m.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += m.TypeID.Len()

	n, ok := m.TypeID.NodeID.(*datatypes.FourByteNodeID)
	if !ok {
		return &errors.ErrUnsupported{m.TypeID.NodeID, "should be FourByteNodeID."}
	}

	switch n.Identifier {
	case ServiceTypeOpenSecureChannelRequest:
		m.Service = &OpenSecureChannelRequest{}
	}

	if err := m.Service.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}

	return nil
}

// Serialize serializes Message into bytes.
func (m *Message) Serialize() ([]byte, error) {
	return nil, nil
}

// SerializeTo serializes Message into bytes.
func (m *Message) SerializeTo(b []byte) error {
	return nil
}

// Len returns the actual length of Message.
func (m *Message) Len() int {
	var l = 0
	if m.TypeID != nil {
		l += m.TypeID.Len()
	}
	if m.Service != nil {
		l += m.Service.Len()
	}

	return l
}

// String returns Message in string.
func (m *Message) String() string {
	return fmt.Sprintf("%v, %v", m.TypeID, m.Service)
}

*/
