// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/binary"

	"github.com/wmnsk/gopcua/datatypes"
)

// RequestHeader represents a Request Header in each services.
type RequestHeader struct {
	AuthenticationToken datatypes.NodeID
	Timestamp           uint64
	RequestHandle       uint32
	ReturnDiagnostics   uint32
	AuditEntryID        datatypes.String
	TimeoutHint         uint32
	AdditionalHeader    *AdditionalHeader
	Payload             []byte
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

	if err := r.AuditEntryID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += r.AuditEntryID.Len()

	r.TimeoutHint = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	if err := r.AdditionalHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += r.AdditionalHeader.Len()

	copy(r.Payload, b[offset:])

	return nil
}
