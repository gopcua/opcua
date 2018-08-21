// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/utils"
)

// RequestHeader represents a Request Header in each services.
//
// Specification: Part 4, 7.28
type RequestHeader struct {
	AuthenticationToken datatypes.NodeID
	Timestamp           time.Time
	RequestHandle       uint32
	ReturnDiagnostics   uint32
	AuditEntryID        *datatypes.String
	TimeoutHint         uint32
	AdditionalHeader    *AdditionalHeader
	Payload             []byte
}

// NewRequestHeader creates a new RequestHeader.
// TODO: impl better time handling
func NewRequestHeader(authToken datatypes.NodeID, timestamp time.Time, handle, diag, timeout uint32, auditID string, additionalHeader *AdditionalHeader, payload []byte) *RequestHeader {
	return &RequestHeader{
		AuthenticationToken: authToken,
		Timestamp:           timestamp,
		RequestHandle:       handle,
		ReturnDiagnostics:   diag,
		AuditEntryID:        datatypes.NewString(auditID),
		TimeoutHint:         timeout,
		AdditionalHeader:    additionalHeader,
		Payload:             payload,
	}
}

// DecodeRequestHeader decodes given bytes into RequestHeader.
func DecodeRequestHeader(b []byte) (*RequestHeader, error) {
	r := &RequestHeader{}
	if err := r.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return r, nil
}

// DecodeFromBytes decodes given bytes into RequestHeader.
func (r *RequestHeader) DecodeFromBytes(b []byte) error {
	var (
		err    error
		offset int
	)

	r.AuthenticationToken, err = datatypes.DecodeNodeID(b)
	if err != nil {
		return err
	}
	offset += r.AuthenticationToken.Len()

	r.Timestamp = utils.DecodeTimestamp(b[offset : offset+8])
	offset += 8

	r.RequestHandle = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	r.ReturnDiagnostics = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	r.AuditEntryID = &datatypes.String{}
	if err := r.AuditEntryID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += r.AuditEntryID.Len()

	r.TimeoutHint = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	r.AdditionalHeader = &AdditionalHeader{}
	if err := r.AdditionalHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += r.AdditionalHeader.Len()

	r.Payload = b[offset:]

	return nil
}

// Serialize serializes RequestHeader into bytes.
func (r *RequestHeader) Serialize() ([]byte, error) {
	b := make([]byte, r.Len())
	if err := r.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes RequestHeader into bytes.
func (r *RequestHeader) SerializeTo(b []byte) error {
	var offset = 0
	if err := r.AuthenticationToken.SerializeTo(b); err != nil {
		return err
	}
	offset += r.AuthenticationToken.Len()

	utils.EncodeTimestamp(b[offset:offset+8], r.Timestamp)
	offset += 8
	binary.LittleEndian.PutUint32(b[offset:offset+4], r.RequestHandle)
	offset += 4
	binary.LittleEndian.PutUint32(b[offset:offset+4], r.ReturnDiagnostics)
	offset += 4

	if err := r.AuditEntryID.SerializeTo(b[offset:]); err != nil {
		return err
	}
	offset += r.AuditEntryID.Len()

	binary.LittleEndian.PutUint32(b[offset:offset+4], r.TimeoutHint)
	offset += 4

	if err := r.AdditionalHeader.SerializeTo(b[offset:]); err != nil {
		return err
	}
	offset += r.AdditionalHeader.Len()

	copy(b[offset:r.Len()], r.Payload)

	return nil
}

// Len returns the actual length of RequestHeader.
func (r *RequestHeader) Len() int {
	return 20 + r.AuthenticationToken.Len() + r.AuditEntryID.Len() + r.AdditionalHeader.Len() + len(r.Payload)
}

// SetDiagServiceLevelSymboricID sets the ServiceLevelSymboricID bit in ReturnDiagnostics.
func (r *RequestHeader) SetDiagServiceLevelSymboricID() {
	r.ReturnDiagnostics |= 0x1
}

// RequestsServiceLevelSymboricID checks if the ServiceLevelSymboricID is requested in ReturnDiagnostics.
func (r *RequestHeader) RequestsServiceLevelSymboricID() bool {
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

// SetDiagOperationLevelSymboricID sets the OperationLevelSymbolicID bit in ReturnDiagnostics.
func (r *RequestHeader) SetDiagOperationLevelSymboricID() {
	r.ReturnDiagnostics |= 0x20
}

// RequestsOperationLevelSymboricID checks if the OperationLevelSymboricID is requested in ReturnDiagnostics.
func (r *RequestHeader) RequestsOperationLevelSymboricID() bool {
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
	r.SetDiagServiceLevelSymboricID()
	r.SetDiagServiceLevelLocalizedText()
	r.SetDiagServiceLevelAdditionalInfo()
	r.SetDiagServiceLevelInnerStatusCode()
	r.SetDiagServiceLevelInnerDiagnostics()
}

// SetDiagOperationAll sets all the operation level bit in ReturnDiagnostics.
func (r *RequestHeader) SetDiagOperationAll() {
	r.SetDiagOperationLevelSymboricID()
	r.SetDiagOperationLevelLocalizedText()
	r.SetDiagOperationLevelAdditionalInfo()
	r.SetDiagOperationLevelInnerStatusCode()
	r.SetDiagOperationLevelInnerDiagnostics()
}

// SetDiagAll sets all the bit in ReturnDiagnostics.
func (r *RequestHeader) SetDiagAll() {
	r.SetDiagServiceLevelSymboricID()
	r.SetDiagServiceLevelLocalizedText()
	r.SetDiagServiceLevelAdditionalInfo()
	r.SetDiagServiceLevelInnerStatusCode()
	r.SetDiagServiceLevelInnerDiagnostics()
	r.SetDiagOperationLevelSymboricID()
	r.SetDiagOperationLevelLocalizedText()
	r.SetDiagOperationLevelAdditionalInfo()
	r.SetDiagOperationLevelInnerStatusCode()
	r.SetDiagOperationLevelInnerDiagnostics()
}

// String returns RequestHeader in string.
func (r *RequestHeader) String() string {
	return fmt.Sprintf("%v, %v, %d, %x, %v, %d, %v, %x",
		r.AuthenticationToken,
		r.Timestamp,
		r.RequestHandle,
		r.ReturnDiagnostics,
		r.AuditEntryID,
		r.TimeoutHint,
		r.AdditionalHeader,
		r.Payload,
	)
}
