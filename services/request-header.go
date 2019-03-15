// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"fmt"
	"time"

	"github.com/gopcua/opcua/ua"
)

// RequestHeader represents a Request Header in each services.
//
// Specification: Part 4, 7.28
type RequestHeader struct {
	AuthenticationToken *ua.NodeID
	Timestamp           time.Time
	RequestHandle       uint32
	ReturnDiagnostics   uint32
	AuditEntryID        string
	TimeoutHint         uint32
	AdditionalHeader    *AdditionalHeader
}

// NewRequestHeader creates a new RequestHeader.
func NewRequestHeader(authToken *ua.NodeID, timestamp time.Time, handle, diag, timeout uint32, auditID string, additionalHeader *AdditionalHeader) *RequestHeader {
	return &RequestHeader{
		AuthenticationToken: authToken,
		Timestamp:           timestamp,
		RequestHandle:       handle,
		ReturnDiagnostics:   diag,
		AuditEntryID:        auditID,
		TimeoutHint:         timeout,
		AdditionalHeader:    additionalHeader,
	}
}

// SetDiagServiceLevelSymbolicID sets the ServiceLevelSymbolicID bit in ReturnDiagnostics.
func (r *RequestHeader) SetDiagServiceLevelSymbolicID() {
	r.ReturnDiagnostics |= 0x1
}

// RequestsServiceLevelSymbolicID checks if the ServiceLevelSymbolicID is requested in ReturnDiagnostics.
func (r *RequestHeader) RequestsServiceLevelSymbolicID() bool {
	return r.ReturnDiagnostics&0x1 == 1
}

// SetDiagServiceLevelLocalizedText sets the ServiceLevelLocalizedText bit in ReturnDiagnostics.
func (r *RequestHeader) SetDiagServiceLevelLocalizedText() {
	r.ReturnDiagnostics |= 0x2
}

// RequestsServiceLevelLocalizedText checks if the ServiceLevelLocalizedText is requested in ReturnDiagnostics.
func (r *RequestHeader) RequestsServiceLevelLocalizedText() bool {
	return (r.ReturnDiagnostics>>1)&0x1 == 1
}

// SetDiagServiceLevelAdditionalInfo sets the ServiceLevelAdditionalInfo bit in ReturnDiagnostics.
func (r *RequestHeader) SetDiagServiceLevelAdditionalInfo() {
	r.ReturnDiagnostics |= 0x4
}

// RequestsServiceLevelAdditionalInfo checks if the ServiceLevelAdditionalInfo is requested in ReturnDiagnostics.
func (r *RequestHeader) RequestsServiceLevelAdditionalInfo() bool {
	return (r.ReturnDiagnostics>>2)&0x1 == 1
}

// SetDiagServiceLevelInnerStatusCode sets the ServiceLevelInnerStatusCode bit in ReturnDiagnostics.
func (r *RequestHeader) SetDiagServiceLevelInnerStatusCode() {
	r.ReturnDiagnostics |= 0x8
}

// RequestsServiceLevelInnerStatusCode checks if the ServiceLevelInnerStatusCode is requested in ReturnDiagnostics.
func (r *RequestHeader) RequestsServiceLevelInnerStatusCode() bool {
	return (r.ReturnDiagnostics>>3)&0x1 == 1
}

// SetDiagServiceLevelInnerDiagnostics sets the ServiceLevelInnerDiagnostics bit in ReturnDiagnostics.
func (r *RequestHeader) SetDiagServiceLevelInnerDiagnostics() {
	r.ReturnDiagnostics |= 0x10
}

// RequestsServiceLevelInnerDiagnostics checks if the ServiceLevelInnerDiagnostics is requested in ReturnDiagnostics.
func (r *RequestHeader) RequestsServiceLevelInnerDiagnostics() bool {
	return (r.ReturnDiagnostics>>4)&0x1 == 1
}

// SetDiagOperationLevelSymbolicID sets the OperationLevelSymbolicID bit in ReturnDiagnostics.
func (r *RequestHeader) SetDiagOperationLevelSymbolicID() {
	r.ReturnDiagnostics |= 0x20
}

// RequestsOperationLevelSymbolicID checks if the OperationLevelSymbolicID is requested in ReturnDiagnostics.
func (r *RequestHeader) RequestsOperationLevelSymbolicID() bool {
	return (r.ReturnDiagnostics>>5)&0x1 == 1
}

// SetDiagOperationLevelLocalizedText sets the OperationLevelLocalizedText bit in ReturnDiagnostics.
func (r *RequestHeader) SetDiagOperationLevelLocalizedText() {
	r.ReturnDiagnostics |= 0x40
}

// RequestsOperationLevelLocalizedText checks if the OperationLevelLocalizedText is requested in ReturnDiagnostics.
func (r *RequestHeader) RequestsOperationLevelLocalizedText() bool {
	return (r.ReturnDiagnostics>>6)&0x1 == 1
}

// SetDiagOperationLevelAdditionalInfo sets the OperationLevelAdditionalInfo bit in ReturnDiagnostics.
func (r *RequestHeader) SetDiagOperationLevelAdditionalInfo() {
	r.ReturnDiagnostics |= 0x80
}

// RequestsOperationLevelAdditionalInfo checks if the OperationLevelAdditionalInfo is requested in ReturnDiagnostics.
func (r *RequestHeader) RequestsOperationLevelAdditionalInfo() bool {
	return (r.ReturnDiagnostics>>7)&0x1 == 1
}

// SetDiagOperationLevelInnerStatusCode sets the OperationLevelInnerStatusCode bit in ReturnDiagnostics.
func (r *RequestHeader) SetDiagOperationLevelInnerStatusCode() {
	r.ReturnDiagnostics |= 0x100
}

// RequestsOperationLevelInnerStatusCode checks if the OperationLevelInnerStatusCode is requested in ReturnDiagnostics.
func (r *RequestHeader) RequestsOperationLevelInnerStatusCode() bool {
	return (r.ReturnDiagnostics>>8)&0x1 == 1
}

// SetDiagOperationLevelInnerDiagnostics sets the OperationLevelInnerDiagnostics bit in ReturnDiagnostics.
func (r *RequestHeader) SetDiagOperationLevelInnerDiagnostics() {
	r.ReturnDiagnostics |= 0x200
}

// RequestsOperationLevelInnerDiagnostics checks if the OperationLevelInnerDiagnostics is requested in ReturnDiagnostics.
func (r *RequestHeader) RequestsOperationLevelInnerDiagnostics() bool {
	return (r.ReturnDiagnostics>>9)&0x1 == 1
}

// SetDiagServiceAll sets all the service level bit in ReturnDiagnostics.
func (r *RequestHeader) SetDiagServiceAll() {
	r.SetDiagServiceLevelSymbolicID()
	r.SetDiagServiceLevelLocalizedText()
	r.SetDiagServiceLevelAdditionalInfo()
	r.SetDiagServiceLevelInnerStatusCode()
	r.SetDiagServiceLevelInnerDiagnostics()
}

// SetDiagOperationAll sets all the operation level bit in ReturnDiagnostics.
func (r *RequestHeader) SetDiagOperationAll() {
	r.SetDiagOperationLevelSymbolicID()
	r.SetDiagOperationLevelLocalizedText()
	r.SetDiagOperationLevelAdditionalInfo()
	r.SetDiagOperationLevelInnerStatusCode()
	r.SetDiagOperationLevelInnerDiagnostics()
}

// SetDiagAll sets all the bit in ReturnDiagnostics.
func (r *RequestHeader) SetDiagAll() {
	r.SetDiagServiceLevelSymbolicID()
	r.SetDiagServiceLevelLocalizedText()
	r.SetDiagServiceLevelAdditionalInfo()
	r.SetDiagServiceLevelInnerStatusCode()
	r.SetDiagServiceLevelInnerDiagnostics()
	r.SetDiagOperationLevelSymbolicID()
	r.SetDiagOperationLevelLocalizedText()
	r.SetDiagOperationLevelAdditionalInfo()
	r.SetDiagOperationLevelInnerStatusCode()
	r.SetDiagOperationLevelInnerDiagnostics()
}

// String returns RequestHeader in string.
func (r *RequestHeader) String() string {
	return fmt.Sprintf("%v, %v, %d, %x, %v, %d, %v",
		r.AuthenticationToken,
		r.Timestamp,
		r.RequestHandle,
		r.ReturnDiagnostics,
		r.AuditEntryID,
		r.TimeoutHint,
		r.AdditionalHeader,
	)
}
