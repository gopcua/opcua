// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

var openSecureChannelResponseCases = []struct {
	description string
	structured  *OpenSecureChannelResponse
	serialized  []byte
}{
	{
		"normal",
		NewOpenSecureChannelResponse(
			NewResponseHeader(
				time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
				1, 0, NewNullDiagnosticInfo(), []string{}, NewNullAdditionalHeader(), nil,
			),
			0,
			NewChannelSecurityToken(
				1, 2, time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC), 6000000,
			),
			[]byte{0xff},
		),
		[]byte{
			// TypeID
			0x01, 0x00, 0xc1, 0x01,
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
			// ServerProtocolVersion
			0x00, 0x00, 0x00, 0x00,
			// SecurityToken
			0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00,
			0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30, 0xd4, 0x01,
			0x80, 0x8d, 0x5b, 0x00,
			// ServerNonce
			0x01, 0x00, 0x00, 0x00, 0xff,
		},
	},
}

func TestDecodeOpenSecureChannelResponse(t *testing.T) {
	for _, c := range openSecureChannelResponseCases {
		got, err := DecodeOpenSecureChannelResponse(c.serialized)
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

func TestSerializeOpenSecureChannelResponse(t *testing.T) {
	for _, c := range openSecureChannelResponseCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestOpenSecureChannelResponseLen(t *testing.T) {
	for _, c := range openSecureChannelResponseCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestOpenSecureChannelResponseServiceType(t *testing.T) {
	for _, c := range openSecureChannelResponseCases {
		if c.structured.ServiceType() != ServiceTypeOpenSecureChannelResponse {
			t.Errorf(
				"ServiceType doesn't match. Want: %d, Got: %d",
				ServiceTypeOpenSecureChannelResponse,
				c.structured.ServiceType(),
			)
		}
	}
}
