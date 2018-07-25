// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Error definitions.
// Part 6 - Mappings Release 1.04 Specification, Table 55 – OPC UA Connection Protocol error codes
// NOTE: BadCertificateUnknown is not implemented, as it cannot be found in http://www.opcfoundation.org/UA/schemas/1.04/StatusCode.csv.
const (
	BadTCPServerTooBusy                   uint32 = 0x807d0000
	BadTCPMessageTypeInvalid                     = 0x807e0000
	BadTCPSecureChannelUnknown                   = 0x807f0000
	BadTCPMessageTooLarge                        = 0x80800000
	BadTimeout                                   = 0x800a0000
	BadTCPNotEnoughResources                     = 0x80810000
	BadTCPInternalError                          = 0x80820000
	BadTCPEndpointURLInvalid                     = 0x80830000
	BadSecurityChecksFailed                      = 0x80130000
	BadRequestInterrupted                        = 0x80840000
	BadRequestTimeout                            = 0x80850000
	BadSecureChannelClosed                       = 0x80860000
	BadSecureChannelTokenUnknown                 = 0x80870000
	BadCertificateUntrusted                      = 0x801a0000
	BadCertificateTimeInvalid                    = 0x80140000
	BadCertificateIssuerTimeInvalid              = 0x80150000
	BadCertificateUseNotAllowed                  = 0x80180000
	BadCertificateIssuerUseNotAllowed            = 0x80190000
	BadCertificateRevocationUnknown              = 0x801b0000
	BadCertificateIssuerRevocationUnknown        = 0x801c0000
	BadCertificateRevoked                        = 0x801d0000
	BadCertificateIssuerRevoked                  = 0x801e0000
	//BadCertificateUnknown = N/A
)

// Error represents a OPC UA Error.
type Error struct {
	*Header
	Error      uint32
	ReasonSize uint32
	Reason     []byte
}

// NewError creates a new OPC UA Error.
func NewError(err uint32, reason string) *Error {
	e := &Error{
		Header: NewHeader(
			MessageTypeError,
			ChunkTypeFinal,
			nil,
		),
		Error:  err,
		Reason: []byte(reason),
	}
	e.SetLength()

	return e
}

// DecodeError decodes given bytes into OPC UA Error.
func DecodeError(b []byte) (*Error, error) {
	e := &Error{}
	if err := e.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return e, nil
}

// DecodeFromBytes decodes given bytes into OPC UA Error.
func (e *Error) DecodeFromBytes(b []byte) error {
	var err error
	if len(b) < 8 {
		return errors.New("Too short to decode Error")
	}

	e.Header, err = DecodeHeader(b)
	if err != nil {
		return err
	}
	b = e.Header.Payload

	e.Error = binary.LittleEndian.Uint32(b[:4])
	e.ReasonSize = binary.LittleEndian.Uint32(b[4:8])
	e.Reason = b[8:]

	return nil
}

// Serialize serializes OPC UA Error into bytes.
func (e *Error) Serialize() ([]byte, error) {
	b := make([]byte, int(e.MessageSize))
	if err := e.SerializeTo(b); err != nil {
		return nil, err
	}
	return b, nil
}

// SerializeTo serializes OPC UA Error into given bytes.
// TODO: add error handling.
func (e *Error) SerializeTo(b []byte) error {
	if e == nil {
		return errors.New("Error is nil")
	}
	e.Header.Payload = make([]byte, e.Len()-8)

	binary.LittleEndian.PutUint32(e.Header.Payload[:4], e.Error)
	binary.LittleEndian.PutUint32(e.Header.Payload[4:8], e.ReasonSize)
	copy(e.Header.Payload[8:], e.Reason)

	e.Header.SetLength()
	return e.Header.SerializeTo(b)
}

// Len returns the actual length of Error in int.
func (e *Error) Len() int {
	return 16 + len(e.Reason)
}

// SetLength sets the length of Error.
func (e *Error) SetLength() {
	e.MessageSize = uint32(16 + len(e.Reason))
	e.ReasonSize = uint32(len(e.Reason))
}

// String returns Error in string.
func (e *Error) String() string {
	return fmt.Sprintf(
		"Header: %v, Error: %d, Reason: %s",
		e.Header,
		e.Error,
		e.Reason,
	)
}
