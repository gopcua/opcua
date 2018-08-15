// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/hex"
	"testing"

	"github.com/wmnsk/gopcua/datatypes"
)

var testResponseHeaderBytes = [][]byte{
	{
		// Timestamp
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
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
	case r.Timestamp != 0x00000001:
		t.Errorf("Timestamp doesn't match. Want: %d, Got: %d", 0x00000001, r.Timestamp)
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
		0x00000001,
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
