// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/wmnsk/gopcua/status"
)

var writeResponse = []struct {
	description string
	structured  *WriteResponse
	serialized  []byte
}{
	{
		"single-result",
		NewWriteResponse(
			NewResponseHeader(
				time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
				1, 0, NewNullDiagnosticInfo(), []string{}, NewNullAdditionalHeader(), nil,
			),
			[]uint32{0},
			nil,
		),
		[]byte{
			// TypeID
			0x01, 0x00, 0xa4, 0x02,
			// Timestamp
			0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30, 0xd4, 0x01,
			// RequestHandle
			0x01, 0x00, 0x00, 0x00,
			// ServiceResult
			0x00, 0x00, 0x00, 0x00,
			// ServiceDiagnostics
			0x00,
			// StringTable
			0x00, 0x00, 0x00, 0x00,
			// AdditionalHeader
			0x00, 0x00, 0x00,
			// Results
			0x01, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
			// DiagnosticInfos
			0x00, 0x00, 0x00, 0x00,
		},
	},
	{
		"multiple-results",
		NewWriteResponse(
			NewResponseHeader(
				time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
				1, 0, NewNullDiagnosticInfo(), []string{}, NewNullAdditionalHeader(), nil,
			),
			[]uint32{0, status.BadUserAccessDenied},
			nil,
		),
		[]byte{
			// TypeID
			0x01, 0x00, 0xa4, 0x02,
			// Timestamp
			0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30, 0xd4, 0x01,
			// RequestHandle
			0x01, 0x00, 0x00, 0x00,
			// ServiceResult
			0x00, 0x00, 0x00, 0x00,
			// ServiceDiagnostics
			0x00,
			// StringTable
			0x00, 0x00, 0x00, 0x00,
			// AdditionalHeader
			0x00, 0x00, 0x00,
			// Results
			0x02, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x1f, 0x80,
			// DiagnosticInfos
			0x00, 0x00, 0x00, 0x00,
		},
	},
}

func TestDecodeWriteResponse(t *testing.T) {
	for _, c := range writeResponse {
		got, err := DecodeWriteResponse(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		// need to clear Payload here.
		got.Payload = nil

		if diff := cmp.Diff(got, c.structured, decodeCmpOpt); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeWriteResponse(t *testing.T) {
	for _, c := range writeResponse {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestWriteResponseLen(t *testing.T) {
	for _, c := range writeResponse {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestWriteResponseServiceType(t *testing.T) {
	for _, c := range writeResponse {
		if c.structured.ServiceType() != ServiceTypeWriteResponse {
			t.Errorf(
				"ServiceType doesn't match. Want: %d, Got: %d",
				ServiceTypeWriteResponse,
				c.structured.ServiceType(),
			)
		}
	}
}
