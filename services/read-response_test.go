// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"log"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/wmnsk/gopcua/datatypes"
)

var readResponseCases = []struct {
	description string
	structured  *ReadResponse
	serialized  []byte
}{
	{
		"read response with single float value",
		NewReadResponse(
			NewResponseHeader(
				time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
				1, 0, NewNullDiagnosticInfo(), []string{}, NewNullAdditionalHeader(), nil,
			),
			[]*datatypes.DataValue{
				datatypes.NewDataValue(
					true, false, false, false, false, false,
					datatypes.NewVariant(
						datatypes.NewFloat(2.5001559257507324),
					), 0, time.Time{}, 0, time.Time{}, 0,
				),
			},
			[]*DiagnosticInfo{
				NewNullDiagnosticInfo(),
			},
		),
		[]byte{
			// TypeID
			0x01, 0x00, 0x7a, 0x02,
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
			// ArraySize
			0x01, 0x00, 0x00, 0x00,
			// EncodingMask
			0x01,
			// Value
			0x0a, 0x8e, 0x02, 0x20, 0x40, 0x01, 0x00, 0x00, 0x00, 0x00,
		},
	},
}

func TestDecodeReadResponse(t *testing.T) {
	for _, c := range readResponseCases {
		got, err := DecodeReadResponse(c.serialized)
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

func TestSerializeReadResponse(t *testing.T) {
	for _, c := range readResponseCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}
		log.Printf("%#x", got)
		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestReadResponseLen(t *testing.T) {
	for _, c := range readResponseCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestReadResponseServiceType(t *testing.T) {
	for _, c := range readResponseCases {
		if c.structured.ServiceType() != ServiceTypeReadResponse {
			t.Errorf(
				"ServiceType doesn't match. Want: %d, Got: %d",
				ServiceTypeReadResponse,
				c.structured.ServiceType(),
			)
		}
	}
}
