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

var closeSessionResponseCases = []struct {
	description string
	structured  *CloseSessionResponse
	serialized  []byte
}{
	{ // Without dummy nonce, results nor diags
		"nothing",
		NewCloseSessionResponse(
			NewResponseHeader(
				time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
				1, 0, datatypes.NewNullDiagnosticInfo(), []string{}, NewNullAdditionalHeader(), nil,
			),
		),
		[]byte{
			// TypeID
			0x01, 0x00, 0xdc, 0x01,
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
		},
	},
}

func TestDecodeCloseSessionResponse(t *testing.T) {
	for _, c := range closeSessionResponseCases {
		got, err := DecodeCloseSessionResponse(c.serialized)
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

func TestSerializeCloseSessionResponse(t *testing.T) {
	for _, c := range closeSessionResponseCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestCloseSessionResponseLen(t *testing.T) {
	for _, c := range closeSessionResponseCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestCloseSessionResponseServiceType(t *testing.T) {
	for _, c := range closeSessionResponseCases {
		if c.structured.ServiceType() != ServiceTypeCloseSessionResponse {
			t.Errorf(
				"ServiceType doesn't match. Want: %d, Got: %d",
				ServiceTypeCloseSessionResponse,
				c.structured.ServiceType(),
			)
		}
	}
}
