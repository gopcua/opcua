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
	h := &Error{
		Header: NewHeader(
			MessageTypeError,
			ChunkTypeFinal,
			nil,
		),
		Error:  err,
		Reason: []byte(reason),
	}
	h.SetLength()

	return h
}

// DecodeError decodes given bytes into OPC UA Error.
func DecodeError(b []byte) (*Error, error) {
	h := &Error{}
	if err := h.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return h, nil
}

// DecodeFromBytes decodes given bytes into OPC UA Error.
func (h *Error) DecodeFromBytes(b []byte) error {
	var err error
	if len(b) < 8 {
		return errors.New("Too short to decode Error")
	}

	h.Header, err = DecodeHeader(b)
	if err != nil {
		return err
	}
	b = h.Header.Payload

	h.Error = binary.LittleEndian.Uint32(b[:4])
	h.ReasonSize = binary.LittleEndian.Uint32(b[4:8])
	h.Reason = b[8:]

	return nil
}

// Serialize serializes OPC UA Error into bytes.
func (h *Error) Serialize() ([]byte, error) {
	b := make([]byte, int(h.MessageSize))
	if err := h.SerializeTo(b); err != nil {
		return nil, err
	}
	return b, nil
}

// SerializeTo serializes OPC UA Error into given bytes.
// TODO: add error handling.
func (h *Error) SerializeTo(b []byte) error {
	if h == nil {
		return errors.New("Error is nil")
	}
	h.Header.Payload = make([]byte, h.Len()-8)

	binary.LittleEndian.PutUint32(h.Header.Payload[:4], h.Error)
	binary.LittleEndian.PutUint32(h.Header.Payload[4:8], h.ReasonSize)
	copy(h.Header.Payload[8:], h.Reason)

	h.Header.SetLength()
	return h.Header.SerializeTo(b)
}

// Len returns the actual length of Error in int.
func (h *Error) Len() int {
	return 16 + len(h.Reason)
}

// SetLength sets the length of Error.
func (h *Error) SetLength() {
	h.MessageSize = uint32(16 + len(h.Reason))
	h.ReasonSize = uint32(len(h.Reason))
}

// String returns Error in string.
func (h *Error) String() string {
	return fmt.Sprintf(
		"Header: %v, Error: %d, Reason: %s",
		h.Header,
		h.Error,
		h.Reason,
	)
}
