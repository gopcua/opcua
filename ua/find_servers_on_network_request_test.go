// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"testing"
	"time"
)

func TestFindServersOnNetworkRequest(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name: "normal",
			Struct: &FindServersOnNetworkRequest{
				RequestHeader: &RequestHeader{
					AuthenticationToken: NewByteStringNodeID(0x00, []byte{
						0x08, 0x22, 0x87, 0x62, 0xba, 0x81, 0xe1, 0x11,
						0xa6, 0x43, 0xf8, 0x77, 0x7b, 0xc6, 0x2f, 0xc8,
					}),
					Timestamp:        time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
					RequestHandle:    1,
					AdditionalHeader: NewExtensionObject(nil),
				},
				StartingRecordID:       1000,
				MaxRecordsToReturn:     0,
				ServerCapabilityFilter: nil,
			},
			//Struct: NewFindServersOnNetworkRequest(
			//	NewRequestHeader(
			//		NewByteStringNodeID(0x00, []byte{
			//			0x08, 0x22, 0x87, 0x62, 0xba, 0x81, 0xe1, 0x11,
			//			0xa6, 0x43, 0xf8, 0x77, 0x7b, 0xc6, 0x2f, 0xc8,
			//		}),
			//		time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
			//		1, 0, 0, "", NewExtensionObject(nil),
			//	),
			//	1000,
			//	0,
			//	// todo(fs): adding an empty string here is a bug
			//	// todo(fs): since this creates an array of size 1
			//	// todo(fs): and not an empty array. Also, using
			//	// todo(fs): ...string always generates an empty
			//	// todo(fs): but never 'nil'.
			//	//"",
			//	nil,
			//),
			Bytes: []byte{
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
				// StartingRecordID
				0xe8, 0x03, 0x00, 0x00,
				// MaxRecordsToReturn
				0x00, 0x00, 0x00, 0x00,
				// ServerCapabilityFilter
				0xff, 0xff, 0xff, 0xff,
			},
		},
	}
	RunCodecTest(t, cases)
}
