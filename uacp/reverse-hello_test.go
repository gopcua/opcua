// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"testing"
)

var testReverseHelloBytes = [][]byte{
	{ // Undefined type of message
		// MessageType: RHE
		0x52, 0x48, 0x45,
		// Chunk Type: F
		0x46,
		// MessageSize: 12
		0x5c, 0x00, 0x00, 0x00,
		// ServerURI
		0x26, 0x00, 0x00, 0x00, 0x6f, 0x70, 0x63, 0x2e,
		0x74, 0x63, 0x70, 0x3a, 0x2f, 0x2f, 0x77, 0x6f,
		0x77, 0x2e, 0x69, 0x74, 0x73, 0x2e, 0x65, 0x61,
		0x73, 0x79, 0x3a, 0x31, 0x31, 0x31, 0x31, 0x31,
		0x2f, 0x55, 0x41, 0x2f, 0x53, 0x65, 0x72, 0x76,
		0x65, 0x72,
		// EndPointURL
		0x26, 0x00, 0x00, 0x00, 0x6f, 0x70, 0x63, 0x2e,
		0x74, 0x63, 0x70, 0x3a, 0x2f, 0x2f, 0x77, 0x6f,
		0x77, 0x2e, 0x69, 0x74, 0x73, 0x2e, 0x65, 0x61,
		0x73, 0x79, 0x3a, 0x31, 0x31, 0x31, 0x31, 0x31,
		0x2f, 0x55, 0x41, 0x2f, 0x53, 0x65, 0x72, 0x76,
		0x65, 0x72,
	},
	{},
	{},
}

func TestDecodeReverseHello(t *testing.T) {
	r, err := DecodeReverseHello(testReverseHelloBytes[0])
	if err != nil {
		t.Fatalf("Failed to decode ReverseHello: %s", err)
	}

	switch {
	case r.MessageTypeValue() != MessageTypeReverseHello:
		t.Errorf("MessageType doesn't match. Want: %s, Got: %s", MessageTypeReverseHello, r.MessageTypeValue())
	case r.ChunkTypeValue() != ChunkTypeFinal:
		t.Errorf("ChunkType doesn't match. Want: %s, Got: %s", ChunkTypeFinal, r.ChunkTypeValue())
	case r.MessageSize != 92:
		t.Errorf("MessageSize doesn't match. Want: %d, Got: %d", 92, r.MessageSize)
	case r.ServerURI.Get() != "opc.tcp://wow.its.easy:11111/UA/Server":
		t.Errorf("ServerURI doesn't match. Want: %s, Got: %s", "opc.tcp://wow.its.easy:11111/UA/Server", r.ServerURI.Get())
	case r.EndPointURL.Get() != "opc.tcp://wow.its.easy:11111/UA/Server":
		t.Errorf("EndPointURL doesn't match. Want: %s, Got: %s", "opc.tcp://wow.its.easy:11111/UA/Server", r.EndPointURL.Get())
	}
}

func TestSerializeReverseHello(t *testing.T) {
	r := NewReverseHello(
		"opc.tcp://wow.its.easy:11111/UA/Server", // ServerURI
		"opc.tcp://wow.its.easy:11111/UA/Server", // EndPointURL
	)

	serialized, err := r.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize ReverseHello: %s", err)
	}

	for i, s := range serialized {
		x := testReverseHelloBytes[0][i]
		if s != x {
			t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
		}
	}
	t.Logf("%x", serialized)
}
