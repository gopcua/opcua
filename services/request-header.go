// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/datatypes"
)

// RequestHeader represents a Request Header in each services.
type RequestHeader struct {
	AuthenticationToken datatypes.NodeID
	Timestamp           uint64
	RequestHandle       uint32
	ReturnDiagnostics   uint32
	AuditEntryID        *datatypes.String
	TimeoutHint         uint32
	AdditionalHeader    *AdditionalHeader
	Payload             []byte
}

// NewRequestHeader creates a new RequestHeader.
// TODO: impl better time handling
func NewRequestHeader(authToken datatypes.NodeID, timestamp uint64, handle, diag, timeout uint32, auditID string, additionalHeader *AdditionalHeader, payload []byte) *RequestHeader {
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

	r.Timestamp = binary.LittleEndian.Uint64(b[offset : offset+8])
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

	binary.LittleEndian.PutUint64(b[offset:offset+8], r.Timestamp)
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

// String returns RequestHeader in string.
func (r *RequestHeader) String() string {
	return fmt.Sprintf("%v, %d, %d, %x, %v, %d, %v, %x",
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
