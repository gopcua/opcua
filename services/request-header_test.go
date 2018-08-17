// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/wmnsk/gopcua/datatypes"
)

var testRequestHeaderBytes = [][]byte{
	{
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
	{},
	{},
}

func TestDecodeRequestHeader(t *testing.T) {
	r, err := DecodeRequestHeader(testRequestHeaderBytes[0])
	if err != nil {
		t.Fatalf("Failed to decode RequestHeader: %s", err)
	}

	dummyStr := hex.EncodeToString(r.Payload)
	switch {
	case r.AuthenticationToken.EncodingMaskValue() != datatypes.TypeFourByte:
		t.Errorf("AuthenticationToken doesn't match. Want: %x, Got: %x", datatypes.TypeFourByte, r.AuthenticationToken.EncodingMaskValue())
	case r.Timestamp.UnixNano() != 1533942000000000000:
		t.Errorf("Timestamp doesn't match. Want: %d, Got: %d", 1533942000000000000, r.Timestamp.UnixNano())
	case r.RequestHandle != 0x01:
		t.Errorf("RequestHandle doesn't match. Want: %d, Got: %d", 0x01, r.RequestHandle)
	case r.ReturnDiagnostics != 0x03ff:
		t.Errorf("ReturnDiagnostics doesn't match. Want: %d, Got: %d", 0x03ff, r.ReturnDiagnostics)
	case r.AuditEntryID.Get() != "foobar":
		t.Errorf("AuditEntryID doesn't match. Want: %s, Got: %s", "foobar", r.AuditEntryID.Get())
	case r.TimeoutHint != 0x00:
		t.Errorf("TimeoutHint doesn't match. Want: %d, Got: %d", 0x00, r.TimeoutHint)
	case r.AdditionalHeader.EncodingMask != 0x00:
		t.Errorf("AdditionalHeader doesn't match. Want: %d, Got: %d", 0x00, r.AdditionalHeader.EncodingMask)
	case dummyStr != "deadbeef":
		t.Errorf("Payload doesn't match. Want: %s, Got: %s", "deadbeef", dummyStr)
	}
	t.Log(r.String())
}

func TestSerializeRequestHeader(t *testing.T) {
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

	serialized, err := r.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize RequestHeader: %s", err)
	}

	for i, s := range serialized {
		x := testRequestHeaderBytes[0][i]
		if s != x {
			t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
		}
	}
	t.Logf("%x", serialized)
}
