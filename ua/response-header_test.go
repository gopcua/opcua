// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"testing"
	"time"
)

func NewNullResponseHeader() *ResponseHeader {
	return &ResponseHeader{
		Timestamp:          time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		ServiceDiagnostics: &DiagnosticInfo{},
		AdditionalHeader:   NewExtensionObject(nil),
	}
}

var nullResponseHeaderBytes = []byte{
	// Timestamp
	0x0, 0x80, 0x3e, 0xd5, 0xde, 0xb1, 0x9d, 0x1,
	// RequestHandle
	0x0, 0x0, 0x0, 0x0,
	// ServiceResult
	0x0, 0x0, 0x0, 0x0,
	// ServiceDiagnostics
	0x0,
	// StringTable
	0xff, 0xff, 0xff, 0xff,
	// AdditionalHeader
	0x00, 0x00, 0x00,
}

func TestResponseHeader(t *testing.T) {
	cases := []CodecTestCase{
		{
			Struct: NewNullResponseHeader(),
			Bytes:  nullResponseHeaderBytes,
		},
		{
			Struct: &ResponseHeader{
				Timestamp:          time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
				RequestHandle:      1,
				ServiceDiagnostics: &DiagnosticInfo{},
				StringTable:        []string{"foo", "bar"},
				AdditionalHeader: &ExtensionObject{
					TypeID: NewTwoByteExpandedNodeID(255),
				},
			},
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
			},
		},
	}
	RunCodecTest(t, cases)
}
