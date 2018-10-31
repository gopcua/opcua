// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"fmt"

	"github.com/wmnsk/gopcua/datatypes"

	"github.com/wmnsk/gopcua/errors"
)

// AsymmetricSecurityHeader represents a Asymmetric Algorithm Security Header in OPC UA Secure Conversation.
type AsymmetricSecurityHeader struct {
	SecurityPolicyURI             *datatypes.String
	SenderCertificate             *datatypes.ByteString
	ReceiverCertificateThumbprint *datatypes.ByteString
	Payload                       []byte
}

// NewAsymmetricSecurityHeader creates a new OPC UA Secure Conversation Asymmetric Algorithm Security Header.
func NewAsymmetricSecurityHeader(uri string, cert, thumbprint []byte, payload []byte) *AsymmetricSecurityHeader {
	return &AsymmetricSecurityHeader{
		SecurityPolicyURI:             datatypes.NewString(uri),
		SenderCertificate:             datatypes.NewByteString(cert),
		ReceiverCertificateThumbprint: datatypes.NewByteString(thumbprint),
		Payload:                       payload,
	}
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
		return errors.NewErrTooShortToDecode(a, "should be longer than 12 bytes")
	}

	var offset = 0
	a.SecurityPolicyURI = &datatypes.String{}
	if err := a.SecurityPolicyURI.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.SecurityPolicyURI.Len()

	a.SenderCertificate = &datatypes.ByteString{}
	if err := a.SenderCertificate.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.SenderCertificate.Len()

	a.ReceiverCertificateThumbprint = &datatypes.ByteString{}
	if err := a.ReceiverCertificateThumbprint.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.ReceiverCertificateThumbprint.Len()

	if len(b[offset:]) > 0 {
		a.Payload = b[offset:]
	}

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
	var offset = 0
	if a.SecurityPolicyURI != nil {
		if err := a.SecurityPolicyURI.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.SecurityPolicyURI.Len()
	}

	if a.SenderCertificate != nil {
		if err := a.SenderCertificate.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.SenderCertificate.Len()
	}

	if a.ReceiverCertificateThumbprint != nil {
		if err := a.ReceiverCertificateThumbprint.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.ReceiverCertificateThumbprint.Len()
	}

	copy(b[offset:], a.Payload)
	return nil
}

// Len returns the actual length of AsymmetricSecurityHeader in int.
func (a *AsymmetricSecurityHeader) Len() int {
	var l = 0

	if a.SecurityPolicyURI != nil {
		l += a.SecurityPolicyURI.Len()
	}
	if a.SenderCertificate != nil {
		l += a.SenderCertificate.Len()
	}
	if a.ReceiverCertificateThumbprint != nil {
		l += a.ReceiverCertificateThumbprint.Len()
	}
	if a.Payload != nil {
		l += len(a.Payload)
	}

	return l
}

// String returns Header in string.
func (a *AsymmetricSecurityHeader) String() string {
	return fmt.Sprintf(
		"SecurityPolicyURI: %v, SenderCertificate: %v, ReceiverCertificateThumbprint: %v, Payload: %x",
		a.SecurityPolicyURI,
		a.SenderCertificate,
		a.ReceiverCertificateThumbprint,
		a.Payload,
	)
}
