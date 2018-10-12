// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/wmnsk/gopcua/datatypes"
)

var activateSessionResponseCases = []struct {
	description string
	structured  *ActivateSessionResponse
	serialized  []byte
}{
	{ // Without dummy nonce, results nor diags
		"nothing",
		NewActivateSessionResponse(
			NewResponseHeader(
				time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
				1, 0, datatypes.NewNullDiagnosticInfo(), []string{}, NewNullAdditionalHeader(), nil,
			),
			nil,
			nil,
			nil,
		),
		[]byte{
			// TypeID
			0x01, 0x00, 0xd6, 0x01,
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
			// ServerNonce
			0xff, 0xff, 0xff, 0xff,
			// Results
			0x00, 0x00, 0x00, 0x00,
			// DiagnosticInfos
			0x00, 0x00, 0x00, 0x00,
		},
	}, { // With dummy nonce, no results and diags
		"with-nonce",
		NewActivateSessionResponse(
			NewResponseHeader(
				time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
				1, 0, datatypes.NewNullDiagnosticInfo(), []string{}, NewNullAdditionalHeader(), nil,
			),
			[]byte{0xde, 0xad, 0xbe, 0xef},
			nil,
			nil,
		),
		[]byte{
			// TypeID
			0x01, 0x00, 0xd6, 0x01,
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
			// ServerNonce
			0x04, 0x00, 0x00, 0x00,
			0xde, 0xad, 0xbe, 0xef,
			// Results
			0x00, 0x00, 0x00, 0x00,
			// DiagnosticInfos
			0x00, 0x00, 0x00, 0x00,
		},
	},
}

func TestDecodeActivateSessionResponse(t *testing.T) {
	for _, c := range activateSessionResponseCases {
		got, err := DecodeActivateSessionResponse(c.serialized)
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

func TestSerializeActivateSessionResponse(t *testing.T) {
	for _, c := range activateSessionResponseCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestActivateSessionResponseLen(t *testing.T) {
	for _, c := range activateSessionResponseCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestActivateSessionResponseServiceType(t *testing.T) {
	for _, c := range activateSessionResponseCases {
		if c.structured.ServiceType() != ServiceTypeActivateSessionResponse {
			t.Errorf(
				"ServiceType doesn't match. Want: %d, Got: %d",
				ServiceTypeActivateSessionResponse,
				c.structured.ServiceType(),
			)
		}
	}
}
