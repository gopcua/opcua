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

var findServersRequestCases = []struct {
	description string
	structured  *FindServersRequest
	serialized  []byte
}{
	{
		"no-locales, no-uris",
		NewFindServersRequest(
			NewRequestHeader(
				datatypes.NewOpaqueNodeID(0x00, []byte{
					0x08, 0x22, 0x87, 0x62, 0xba, 0x81, 0xe1, 0x11,
					0xa6, 0x43, 0xf8, 0x77, 0x7b, 0xc6, 0x2f, 0xc8,
				}),
				time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
				1, 0, 0, "", NewNullAdditionalHeader(), nil,
			),
			"opc.tcp://wow.its.easy:11111/UA/Server",
			[]string{},
			[]string{},
		),
		[]byte{
			// TypeID
			0x01, 0x00, 0xa6, 0x01,
			// AuthenticationToken
			0x05, 0x00, 0x00, 0x10, 0x00, 0x00, 0x00, 0x08,
			0x22, 0x87, 0x62, 0xba, 0x81, 0xe1, 0x11, 0xa6,
			0x43, 0xf8, 0x77, 0x7b, 0xc6, 0x2f, 0xc8,
			// Timestamp
			0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30, 0xd4, 0x01,
			// RequestHandle
			0x01, 0x00, 0x00, 0x00,
			// ReturnDiagnostics
			0x00, 0x00, 0x00, 0x00,
			// AuditEntryID
			0xff, 0xff, 0xff, 0xff,
			// TimeoutHint
			0x00, 0x00, 0x00, 0x00,
			// AdditionalHeader
			0x00, 0x00, 0x00,
			// EndpointURL
			0x26, 0x00, 0x00, 0x00, 0x6f, 0x70, 0x63, 0x2e,
			0x74, 0x63, 0x70, 0x3a, 0x2f, 0x2f, 0x77, 0x6f,
			0x77, 0x2e, 0x69, 0x74, 0x73, 0x2e, 0x65, 0x61,
			0x73, 0x79, 0x3a, 0x31, 0x31, 0x31, 0x31, 0x31,
			0x2f, 0x55, 0x41, 0x2f, 0x53, 0x65, 0x72, 0x76,
			0x65, 0x72,
			// LocaleIDs
			0x00, 0x00, 0x00, 0x00,
			// ServerURIs
			0x00, 0x00, 0x00, 0x00,
		},
	},
}

func TestDecodeFindServersRequest(t *testing.T) {
	for _, c := range findServersRequestCases {
		got, err := DecodeFindServersRequest(c.serialized)
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

func TestSerializeFindServersRequest(t *testing.T) {
	for _, c := range findServersRequestCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestFindServersRequestLen(t *testing.T) {
	for _, c := range findServersRequestCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestFindServersRequestServiceType(t *testing.T) {
	for _, c := range findServersRequestCases {
		if c.structured.ServiceType() != ServiceTypeFindServersRequest {
			t.Errorf(
				"ServiceType doesn't match. Want: %d, Got: %d",
				ServiceTypeFindServersRequest,
				c.structured.ServiceType(),
			)
		}
	}
}
