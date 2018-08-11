// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// AsymmetricSecurityHeader represents a Asymmetric Algorithm Security Header in OPC UA Secure Conversation.
type AsymmetricSecurityHeader struct {
	SecurityPolicyURILength             int32
	SecurityPolicyURI                   []byte
	SenderCertificateLength             int32
	SenderCertificate                   []byte
	ReceiverCertificateThumbprintLength int32
	ReceiverCertificateThumbprint       []byte
	Payload                             []byte
}

// NewAsymmetricSecurityHeader creates a new OPC UA Secure Conversation Asymmetric Algorithm Security Header.
func NewAsymmetricSecurityHeader(uri, cert, thumbprint string, payload []byte) *AsymmetricSecurityHeader {
	a := &AsymmetricSecurityHeader{
		SecurityPolicyURI:             []byte(uri),
		SenderCertificate:             []byte(cert),
		ReceiverCertificateThumbprint: []byte(thumbprint),
		Payload: payload,
	}
	a.SetLength()

	return a
}

// DecodeAsymmetricSecurityHeader decodes given bytes into OPC UA Secure Conversation Asymmetric Algorithm Security Header.
func DecodeAsymmetricSecurityHeader(b []byte) (*AsymmetricSecurityHeader, error) {
	a := &AsymmetricSecurityHeader{}
	if err := a.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return a, nil
}

// DecodeFromBytes decodes given bytes into OPC UA Secure Conversation Asymmetric Algorithm Security Header.
// XXX - May be crashed when the length value and actual size is inconsistent.
func (a *AsymmetricSecurityHeader) DecodeFromBytes(b []byte) error {
	l := len(b)
	if l < 12 {
		return errors.New("Too short to decode AsymmetricSecurityHeader")
	}

	var offset = 4
	a.SecurityPolicyURILength = int32(binary.LittleEndian.Uint32(b[0:offset]))
	if a.SecurityPolicyURILength > 0 {
		a.SecurityPolicyURI = b[offset : offset+int(a.SecurityPolicyURILength)]
		offset += int(a.SecurityPolicyURILength)
	}
	if l < offset+4 {
		return errors.New("Too short to decode AsymmetricSecurityHeader")
	}

	a.SenderCertificateLength = int32(binary.LittleEndian.Uint32(b[offset : offset+4]))
	offset += 4
	if a.SenderCertificateLength > 0 {
		a.SenderCertificate = b[offset : offset+int(a.SenderCertificateLength)]
		offset += int(a.SenderCertificateLength)
	}
	if l < offset+4 {
		return errors.New("Too short to decode AsymmetricSecurityHeader")
	}

	a.ReceiverCertificateThumbprintLength = int32(binary.LittleEndian.Uint32(b[offset : offset+4]))
	offset += 4
	if a.ReceiverCertificateThumbprintLength > 0 {
		a.ReceiverCertificateThumbprint = b[offset : offset+int(a.ReceiverCertificateThumbprintLength)]
		offset += int(a.ReceiverCertificateThumbprintLength)
	}

	a.Payload = b[offset:]

	return nil
}

// Serialize serializes OPC UA Secure Conversation Asymmetric Algorithm Security Header into bytes.
func (a *AsymmetricSecurityHeader) Serialize() ([]byte, error) {
	b := make([]byte, int(a.Len()))
	if err := a.SerializeTo(b); err != nil {
		return nil, err
	}
	return b, nil
}

// SerializeTo serializes OPC UA Secure Conversation AsymmetricSecurityHeader into given bytes.
// TODO: add error handling.
func (a *AsymmetricSecurityHeader) SerializeTo(b []byte) error {
	var offset = 4
	binary.LittleEndian.PutUint32(b[:offset], uint32(a.SecurityPolicyURILength))
	if a.SecurityPolicyURI != nil {
		copy(b[offset:offset+int(a.SecurityPolicyURILength)], a.SecurityPolicyURI)
		offset += int(a.SecurityPolicyURILength)
	}

	binary.LittleEndian.PutUint32(b[offset:offset+4], uint32(a.SenderCertificateLength))
	offset += 4
	if a.SenderCertificate != nil {
		copy(b[offset:offset+int(a.SenderCertificateLength)], a.SenderCertificate)
		offset += int(a.SenderCertificateLength)
	}

	binary.LittleEndian.PutUint32(b[offset:offset+4], uint32(a.ReceiverCertificateThumbprintLength))
	offset += 4
	if a.ReceiverCertificateThumbprint != nil {
		copy(b[offset:offset+int(a.ReceiverCertificateThumbprintLength)], a.ReceiverCertificateThumbprint)
		offset += int(a.ReceiverCertificateThumbprintLength)
	}

	copy(b[offset:], a.Payload)

	return nil
}

// Len returns the actual length of AsymmetricSecurityHeader in int.
func (a *AsymmetricSecurityHeader) Len() int {
	return 12 + len(a.SecurityPolicyURI) + len(a.SenderCertificate) + len(a.ReceiverCertificateThumbprint) + len(a.Payload)
}

// SetLength sets each length field in AsymmetricSecurityHeader.
func (a *AsymmetricSecurityHeader) SetLength() {
	a.SecurityPolicyURILength = int32(len(a.SecurityPolicyURI))
	a.SenderCertificateLength = int32(len(a.SenderCertificate))
	a.ReceiverCertificateThumbprintLength = int32(len(a.ReceiverCertificateThumbprint))
}

// SecurityPolicyURIValue returns SecurityPolicyURI in string.
func (a *AsymmetricSecurityHeader) SecurityPolicyURIValue() string {
	return string(a.SecurityPolicyURI)
}

// SenderCertificateValue returns SenderCertificate in string.
func (a *AsymmetricSecurityHeader) SenderCertificateValue() string {
	return string(a.SenderCertificate)
}

// ReceiverCertificateThumbprintValue returns ReceiverCertificateThumbprint in string.
func (a *AsymmetricSecurityHeader) ReceiverCertificateThumbprintValue() string {
	return string(a.ReceiverCertificateThumbprint)
}

// String returns Header in string.
func (a *AsymmetricSecurityHeader) String() string {
	return fmt.Sprintf(
		"SecurityPolicyURILength: %d, SecurityPolicyURI: %s, SenderCertificateLength: %d, SenderCertificate: %s, ReceiverCertificateThumbprintLength: %d, ReceiverCertificateThumbprint: %s, Payload: %x",
		a.SecurityPolicyURILength,
		a.SecurityPolicyURI,
		a.SenderCertificateLength,
		a.SenderCertificate,
		a.ReceiverCertificateThumbprintLength,
		a.ReceiverCertificateThumbprint,
		a.Payload,
	)
}
