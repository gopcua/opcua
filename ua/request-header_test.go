// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"testing"
	"time"
)

func NewNullRequestHeader() *RequestHeader {
	return &RequestHeader{
		AuthenticationToken: NewTwoByteNodeID(0),
		Timestamp:           time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		AdditionalHeader:    NewExtensionObject(nil),
	}
}

var nullRequestHeaderBytes = []byte{
	// AuthenticationToken
	0x0, 0x0,
	// Timestamp
	0x00, 0x80, 0x3e, 0xd5, 0xde, 0xb1, 0x9d, 0x01,
	// RequestHandle
	0x0, 0x0, 0x0, 0x0,
	// ReturnDiagnostics
	0x0, 0x0, 0x0, 0x0,
	// AuditEntryID
	0xff, 0xff, 0xff, 0xff,
	// TimeeoutHint
	0x0, 0x0, 0x0, 0x0,
	// AdditionalHeader
	0x0, 0x0, 0x0,
}

func TestRequestHeader(t *testing.T) {
	cases := []CodecTestCase{
		{
			Struct: NewNullRequestHeader(),
			Bytes:  nullRequestHeaderBytes,
		},
		{
			Struct: func() *RequestHeader {
				r := &RequestHeader{
					AuthenticationToken: NewFourByteNodeID(0, 33008),
					Timestamp:           time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
					RequestHandle:       1,
					ReturnDiagnostics:   ReturnDiagnosticsAll,
					AuditEntryID:        "foobar",
					AdditionalHeader: &ExtensionObject{
						TypeID: NewTwoByteExpandedNodeID(255),
					},
				}
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
			},
		},
	}
	RunCodecTest(t, cases)
}
