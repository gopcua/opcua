// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

var closeSecureChannelResponseCases = []struct {
	description string
	structured  *CloseSecureChannelResponse
	serialized  []byte
}{
	{
		"normal",
		NewCloseSecureChannelResponse(
			NewResponseHeader(
				time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
				1, 0, NewNullDiagnosticInfo(), []string{}, NewNullAdditionalHeader(), nil,
			),
		),
		[]byte{
			// TypeID
			0x01, 0x00, 0xc7, 0x01,
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

func TestDecodeCloseSecureChannelResponse(t *testing.T) {
	for _, c := range closeSecureChannelResponseCases {
		got, err := DecodeCloseSecureChannelResponse(c.serialized)
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

func TestSerializeCloseSecureChannelResponse(t *testing.T) {
	for _, c := range closeSecureChannelResponseCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestCloseSecureChannelResponseLen(t *testing.T) {
	for _, c := range closeSecureChannelResponseCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestCloseSecureChannelResponseServiceType(t *testing.T) {
	for _, c := range closeSecureChannelResponseCases {
		if c.structured.ServiceType() != ServiceTypeCloseSecureChannelResponse {
			t.Errorf(
				"ServiceType doesn't match. Want: %d, Got: %d",
				ServiceTypeCloseSecureChannelResponse,
				c.structured.ServiceType(),
			)
		}
	}
}
