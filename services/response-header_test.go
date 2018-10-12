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

var testResponseHeaderBytes = [][]byte{
	{
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
	{},
	{},
}

func TestDecodeResponseHeader(t *testing.T) {
	r, err := DecodeResponseHeader(testResponseHeaderBytes[0])
	if err != nil {
		t.Fatalf("Failed to decode ResponseHeader: %s", err)
	}

	dummyStr := hex.EncodeToString(r.Payload)
	switch {
	case r.Timestamp.UnixNano() != 1533942000000000000:
		t.Errorf("Timestamp doesn't match. Want: %d, Got: %d", 1533942000000000000, r.Timestamp.UnixNano())
	case r.RequestHandle != 1:
		t.Errorf("RequestHandle doesn't match. Want: %d, Got: %d", 1, r.RequestHandle)
	case r.ServiceResult != 0x00000000:
		t.Errorf("ServiceResult doesn't match. Want: %d, Got: %d", 0x00000000, r.ServiceResult)
	case r.ServiceDiagnostics.EncodingMask != 0x00:
		t.Errorf("ServiceDiagnostics.EncodingMask doesn't match. Want: %d, Got: %d", 0x00, r.ServiceDiagnostics.EncodingMask)
	case r.StringTable.ArraySize != 2:
		t.Errorf("StringTable.ArraySize doesn't match. Want: %d, Got: %d", 2, r.StringTable.ArraySize)
	case r.AdditionalHeader.TypeID.NodeID.EncodingMaskValue() != 0x00:
		t.Errorf("AdditionalHeader.TypeID.NodeID.EncodingMaskValue() doesn't match. Want: %d, Got: %d", 0x00, r.AdditionalHeader.TypeID.NodeID.EncodingMaskValue())
	case dummyStr != "deadbeef":
		t.Errorf("dummyStr doesn't match. Want: %s, Got: %s", "deadbeef", dummyStr)
	}
	t.Log(r.String())
}

func TestSerializeResponseHeader(t *testing.T) {
	r := NewResponseHeader(
		time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
		1,
		0x00000000,
		datatypes.NewDiagnosticInfo(
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
	)

	serialized, err := r.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize ResponseHeader: %s", err)
	}

	for i, s := range serialized {
		x := testResponseHeaderBytes[0][i]
		if s != x {
			t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
		}
	}
	t.Logf("%x", serialized)
}
