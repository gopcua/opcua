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

// ResponseHeader represents a Response Header in each services.
//
// Specification: Part 4, 7.29
type ResponseHeader struct {
	Timestamp          time.Time
	RequestHandle      uint32
	ServiceResult      uint32
	ServiceDiagnostics *DiagnosticInfo
	StringTable        *datatypes.StringArray
	AdditionalHeader   *AdditionalHeader
	Payload            []byte
}

// NewResponseHeader creates a new ResponseHeader.
func NewResponseHeader(timestamp time.Time, handle, code uint32, diag *DiagnosticInfo, strs []string, additionalHeader *AdditionalHeader, payload []byte) *ResponseHeader {
	r := &ResponseHeader{
		Timestamp:          timestamp,
		RequestHandle:      handle,
		ServiceResult:      code,
		ServiceDiagnostics: diag,
		StringTable:        datatypes.NewStringArray(strs),
		AdditionalHeader:   additionalHeader,
		Payload:            payload,
	}
	if diag == nil {
		r.ServiceDiagnostics = NewNullDiagnosticInfo()
	}

	return r
}

// DecodeResponseHeader decodes given bytes into ResponseHeader.
func DecodeResponseHeader(b []byte) (*ResponseHeader, error) {
	r := &ResponseHeader{}
	if err := r.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return r, nil
}

// DecodeFromBytes decodes given bytes into ResponseHeader.
func (r *ResponseHeader) DecodeFromBytes(b []byte) error {
	var offset = 0

	r.Timestamp = utils.DecodeTimestamp(b[offset : offset+8])
	offset += 8
	r.RequestHandle = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4
	r.ServiceResult = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	r.ServiceDiagnostics = &DiagnosticInfo{}
	if err := r.ServiceDiagnostics.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += r.ServiceDiagnostics.Len()

	r.StringTable = &datatypes.StringArray{}
	if err := r.StringTable.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += r.StringTable.Len()

	r.AdditionalHeader = &AdditionalHeader{}
	if err := r.AdditionalHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += r.AdditionalHeader.Len()

	r.Payload = b[offset:]

	return nil
}

// Serialize serializes ResponseHeader into bytes.
func (r *ResponseHeader) Serialize() ([]byte, error) {
	b := make([]byte, r.Len())
	if err := r.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes ResponseHeader into bytes.
func (r *ResponseHeader) SerializeTo(b []byte) error {
	var offset = 0
	utils.EncodeTimestamp(b[offset:offset+8], r.Timestamp)
	offset += 8
	binary.LittleEndian.PutUint32(b[offset:offset+4], r.RequestHandle)
	offset += 4
	binary.LittleEndian.PutUint32(b[offset:offset+4], r.ServiceResult)
	offset += 4

	if err := r.ServiceDiagnostics.SerializeTo(b[offset:]); err != nil {
		return err
	}
	offset += r.ServiceDiagnostics.Len()

	if err := r.StringTable.SerializeTo(b[offset:]); err != nil {
		return err
	}
	offset += r.StringTable.Len()

	if err := r.AdditionalHeader.SerializeTo(b[offset:]); err != nil {
		return err
	}
	offset += r.AdditionalHeader.Len()

	copy(b[offset:r.Len()], r.Payload)

	return nil
}

// Len returns the actual length of ResponseHeader.
func (r *ResponseHeader) Len() int {
	return 16 + r.ServiceDiagnostics.Len() + r.StringTable.Len() + r.AdditionalHeader.Len() + len(r.Payload)
}

// String returns ResponseHeader in string.
func (r *ResponseHeader) String() string {
	return fmt.Sprintf("%v, %d, %v, %v, %v, %v, %x",
		r.Timestamp,
		r.RequestHandle,
		r.ServiceResult,
		r.ServiceDiagnostics,
		r.StringTable,
		r.AdditionalHeader,
		r.Payload,
	)
}
