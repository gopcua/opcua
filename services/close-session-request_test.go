// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/wmnsk/gopcua/datatypes"
)

var closeSessionRequestCases = []struct {
	description string
	structured  *CloseSessionRequest
	serialized  []byte
}{
	{
		"normal",
		NewCloseSessionRequest(
			time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
			datatypes.NewOpaqueNodeID(0x00, []byte{
				0x08, 0x22, 0x87, 0x62, 0xba, 0x81, 0xe1, 0x11,
				0xa6, 0x43, 0xf8, 0x77, 0x7b, 0xc6, 0x2f, 0xc8,
			}), 1, 0, 0, "", true,
		),
		[]byte{ // CloseSessionRequest
			// TypeID
			0x01, 0x00, 0xd9, 0x01,
			// RequestHeader
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
			// DeleteSubscription
			0x01,
		},
	},
}

// option to regard []T{} and []T{nil} as equal
// https://godoc.org/github.com/google/go-cmp/cmp#example-Option--EqualEmpty
var decodeCmpOpt = cmp.FilterValues(func(x, y interface{}) bool {
	vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
	return (vx.IsValid() && vy.IsValid() && vx.Type() == vy.Type()) &&
		(vx.Kind() == reflect.Slice) && (vx.Len() == 0 && vy.Len() == 0)
}, cmp.Comparer(func(_, _ interface{}) bool { return true }))

func TestDecodeCloseSessionRequest(t *testing.T) {
	for _, c := range closeSessionRequestCases {
		got, err := DecodeCloseSessionRequest(c.serialized)
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

func TestSerializeCloseSessionRequest(t *testing.T) {
	for _, c := range closeSessionRequestCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestCloseSessionRequestLen(t *testing.T) {
	for _, c := range closeSessionRequestCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestCloseSessionRequestServiceType(t *testing.T) {
	for _, c := range closeSessionRequestCases {
		if c.structured.ServiceType() != ServiceTypeCloseSessionRequest {
			t.Errorf(
				"ServiceType doesn't match. Want: %d, Got: %d",
				ServiceTypeCloseSessionRequest,
				c.structured.ServiceType(),
			)
		}
	}
}
