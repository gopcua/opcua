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

func TestResponseHeader(t *testing.T) {
	cases := []codectest.Case{
		{
			Struct: NewResponseHeader(
				time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
				1,
				0x00000000,
				NewDiagnosticInfo(
					false, false, false, false, false, false, false,
					0, 0, 0, 0, nil, 0, nil,
				),
				[]string{"foo", "bar"},
				NewAdditionalHeader(
					datatypes.NewExpandedNodeID(
						false, false,
						datatypes.NewTwoByteNodeID(255),
						"", 0,
					),
					0x00,
				),
				[]byte{0xde, 0xad, 0xbe, 0xef},
			),
			Bytes: []byte{
				// Timestamp
				0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30, 0xd4, 0x01,
				// RequestHandle
				0x01, 0x00, 0x00, 0x00,
				// ServiceResult
				0x00, 0x00, 0x00, 0x00,
				// ServiceDiagnostics
				0x00,
				// StringTable: "foo", "bar"
				0x02, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
				0x66, 0x6f, 0x6f, 0x03, 0x00, 0x00, 0x00, 0x62,
				0x61, 0x72,
				// AdditionalHeader
				0x00, 0xff, 0x00,
				// dummy Payload
				0xde, 0xad, 0xbe, 0xef,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeResponseHeader(b)
	})
}
