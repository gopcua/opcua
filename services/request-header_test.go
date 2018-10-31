// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestRequestHeader(t *testing.T) {
	cases := []codectest.Case{
		{
			Struct: func() *RequestHeader {
				r := NewRequestHeader(
					datatypes.NewFourByteNodeID(0, 33008),
					time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
					1,
					0,
					0,
					"foobar",
					NewAdditionalHeader(
						datatypes.NewExpandedNodeID(
							false, false,
							datatypes.NewTwoByteNodeID(255),
							"", 0,
						),
						0x00,
					),
					[]byte{0xde, 0xad, 0xbe, 0xef},
				)
				r.SetDiagAll()
				return r
			}(),
			Bytes: []byte{
				// AuthenticationToken
				0x01, 0x00, 0xf0, 0x80,
				// Timestamp
				0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30, 0xd4, 0x01,
				// RequestHandle
				0x01, 0x00, 0x00, 0x00,
				// ReturnDiagnostics
				0xff, 0x03, 0x00, 0x00,
				// AuditEntryID
				0x06, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f, 0x62,
				0x61, 0x72,
				// TimeoutHint
				0x00, 0x00, 0x00, 0x00,
				// AdditionalHeader
				0x00, 0xff, 0x00,
				// dummy Payload
				0xde, 0xad, 0xbe, 0xef,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeRequestHeader(b)
	})
}
