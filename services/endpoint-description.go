// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/errors"
)

// EndpointDescription represents an EndpointDescription.
//
// Specification: Part 4, 7.10
type EndpointDescription struct {
	EndpointURL         *datatypes.String
	Server              *ApplicationDescription
	ServerCertificate   *datatypes.ByteString
	MessageSecurityMode uint32
	SecurityPolicyURI   *datatypes.String
	UserIdentityTokens  *UserTokenPolicyArray
	TransportProfileURI *datatypes.String
	SecurityLevel       uint8
}

// NewEndpointDesctiption creates a new NewEndpointDesctiption.
func NewEndpointDesctiption(url string, server *ApplicationDescription, cert []byte, secMode uint32, secURI string, tokens *UserTokenPolicyArray, transportURI string, secLevel uint8) *EndpointDescription {
	return &EndpointDescription{
		EndpointURL:         datatypes.NewString(url),
		Server:              server,
		ServerCertificate:   datatypes.NewByteString(cert),
		MessageSecurityMode: secMode,
		SecurityPolicyURI:   datatypes.NewString(secURI),
		UserIdentityTokens:  tokens,
		TransportProfileURI: datatypes.NewString(transportURI),
		SecurityLevel:       secLevel,
	}
}

// DecodeEndpointDescription decodes given bytes into EndpointDescription.
func DecodeEndpointDescription(b []byte) (*EndpointDescription, error) {
	e := &EndpointDescription{}
	if err := e.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return e, nil
}

// DecodeFromBytes decodes given bytes into EndpointDescription.
func (e *EndpointDescription) DecodeFromBytes(b []byte) error {
	if len(b) < 6 {
		return &errors.ErrTooShortToDecode{e, "should be longer than 6 bytes."}
	}

	var offset = 0
	e.EndpointURL = &datatypes.String{}
	if err := e.EndpointURL.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += e.EndpointURL.Len()

	e.Server = &ApplicationDescription{}
	if err := e.Server.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += e.Server.Len()

	e.ServerCertificate = &datatypes.ByteString{}
	if err := e.ServerCertificate.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += e.ServerCertificate.Len()

	e.MessageSecurityMode = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	e.SecurityPolicyURI = &datatypes.String{}
	if err := e.SecurityPolicyURI.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += e.SecurityPolicyURI.Len()

	e.UserIdentityTokens = &UserTokenPolicyArray{}
	if err := e.UserIdentityTokens.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += e.UserIdentityTokens.Len()

	e.TransportProfileURI = &datatypes.String{}
	if err := e.TransportProfileURI.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += e.TransportProfileURI.Len()

	e.SecurityLevel = b[offset]

	return nil
}

// Serialize serializes EndpointDescription into bytes.
func (e *EndpointDescription) Serialize() ([]byte, error) {
	b := make([]byte, e.Len())
	if err := e.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes EndpointDescription into bytes.
func (e *EndpointDescription) SerializeTo(b []byte) error {
	var offset = 0
	if e.EndpointURL != nil {
		if err := e.EndpointURL.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += e.EndpointURL.Len()
	}

	if e.Server != nil {
		if err := e.Server.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += e.Server.Len()
	}

	if e.ServerCertificate != nil {
		if err := e.ServerCertificate.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += e.ServerCertificate.Len()
	}

	binary.LittleEndian.PutUint32(b[offset:offset+4], e.MessageSecurityMode)
	offset += 4

	if e.SecurityPolicyURI != nil {
		if err := e.SecurityPolicyURI.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += e.SecurityPolicyURI.Len()
	}

	if e.UserIdentityTokens != nil {
		if err := e.UserIdentityTokens.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += e.UserIdentityTokens.Len()
	}

	if e.TransportProfileURI != nil {
		if err := e.TransportProfileURI.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += e.TransportProfileURI.Len()
	}

	b[offset] = e.SecurityLevel

	return nil
}

// Len returns the actual length of EndpointDescription in int.
func (e *EndpointDescription) Len() int {
	var l = 5
	if e.EndpointURL != nil {
		l += e.EndpointURL.Len()
	}
	if e.Server != nil {
		l += e.Server.Len()
	}
	if e.ServerCertificate != nil {
		l += e.ServerCertificate.Len()
	}
	if e.SecurityPolicyURI != nil {
		l += e.SecurityPolicyURI.Len()
	}
	if e.UserIdentityTokens != nil {
		l += e.UserIdentityTokens.Len()
	}
	if e.TransportProfileURI != nil {
		l += e.TransportProfileURI.Len()
	}

	return l
}

// String returns EndpointDescription in string.
func (e *EndpointDescription) String() string {
	return fmt.Sprintf("%s, %v, %x, %d, %s, %s, %d",
		e.EndpointURL.Get(),
		e.Server,
		e.ServerCertificate.Get(),
		e.MessageSecurityMode,
		e.SecurityPolicyURI.Get(),
		e.TransportProfileURI.Get(),
		e.SecurityLevel,
	)
}

// EndpointDescriptionArray represents an EndpointDescriptionArray.
type EndpointDescriptionArray struct {
	ArraySize            int32
	EndpointDescriptions []*EndpointDescription
}

// NewEndpointDescriptionArray creates an NewEndpointDescriptionArray from multiple EndpointDesctiption.
func NewEndpointDescriptionArray(descs []*EndpointDescription) *EndpointDescriptionArray {
	e := &EndpointDescriptionArray{
		ArraySize: int32(len(descs)),
	}

	for _, desc := range descs {
		e.EndpointDescriptions = append(e.EndpointDescriptions, desc)
	}

	return e
}

// DecodeEndpointDescriptionArray decodes given bytes into EndpointDescriptionArray.
func DecodeEndpointDescriptionArray(b []byte) (*EndpointDescriptionArray, error) {
	e := &EndpointDescriptionArray{}
	if err := e.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return e, nil
}

// DecodeFromBytes decodes given bytes into EndpointDescriptionArray.
func (e *EndpointDescriptionArray) DecodeFromBytes(b []byte) error {
	if len(b) < 4 {
		return &errors.ErrTooShortToDecode{e, "should be longer than 4 bytes."}
	}

	e.ArraySize = int32(binary.LittleEndian.Uint32(b[:4]))
	if e.ArraySize <= 0 {
		return nil
	}

	var offset = 4
	for i := 0; i < int(e.ArraySize); i++ {
		ed, err := DecodeEndpointDescription(b[offset:])
		if err != nil {
			return err
		}
		e.EndpointDescriptions = append(e.EndpointDescriptions, ed)
		offset += e.EndpointDescriptions[i].Len()
	}

	return nil
}

// Serialize serializes EndpointDescriptionArray into bytes.
func (e *EndpointDescriptionArray) Serialize() ([]byte, error) {
	b := make([]byte, e.Len())
	if err := e.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes EndpointDescriptionArray into bytes.
func (e *EndpointDescriptionArray) SerializeTo(b []byte) error {
	binary.LittleEndian.PutUint32(b[:4], uint32(e.ArraySize))
	if e.ArraySize <= 0 {
		return nil
	}

	var offset = 4
	for _, ed := range e.EndpointDescriptions {
		if err := ed.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += ed.Len()
	}

	return nil
}

// Len returns the actual length of EndpointDescriptionArray in int.
func (e *EndpointDescriptionArray) Len() int {
	var l = 4
	for _, ed := range e.EndpointDescriptions {
		l += ed.Len()
	}

	return l
}
