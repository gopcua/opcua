// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
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
//
// Specification: Part6, 7.1.2.5
type Error struct {
	Error  uint32
	Reason string
}

// NewError creates a new OPC UA Error.
func NewError(err uint32, reason string) *Error {
	return &Error{
		Error:  err,
		Reason: reason,
	}
}

// String returns Error in string.
func (e *Error) String() string {
	return fmt.Sprintf(
		"Error: %d, Reason: %s",
		e.Error,
		e.Reason,
	)
}
