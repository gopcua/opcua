// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"testing"
)

func TestDecode(t *testing.T) {
	t.Run("HEL", func(t *testing.T) {
		t.Parallel()
		u, err := Decode(testHelloBytes[0])
		if err != nil {
			t.Fatalf("Failed to decode UACP: %s", err)
		}

		if _, ok := u.(*Hello); !ok {
			t.Errorf("Decoded as wrong type. Want: %s, Got: %T", "*Hello", u)
		}

		var (
			msgType   = u.MessageTypeValue()
			chunkType = u.ChunkTypeValue()
		)
		switch {
		case msgType != MessageTypeHello:
			t.Errorf("MessageType doesn't match. Want: %s, Got: %s", MessageTypeHello, msgType)
		case chunkType != "F":
			t.Errorf("ChunkType doesn't match. Want: %s, Got: %s", "F", chunkType)
		}
	})
	t.Run("ACK", func(t *testing.T) {
		t.Parallel()
		u, err := Decode(testAcknowledgeBytes[0])
		if err != nil {
			t.Fatalf("Failed to decode UACP: %s", err)
		}

		if _, ok := u.(*Acknowledge); !ok {
			t.Errorf("Decoded as wrong type. Want: %s, Got: %T", "*Acknowledge", u)
		}

		var (
			msgType   = u.MessageTypeValue()
			chunkType = u.ChunkTypeValue()
		)
		switch {
		case msgType != MessageTypeAcknowledge:
			t.Errorf("MessageType doesn't match. Want: %s, Got: %s", MessageTypeAcknowledge, msgType)
		case chunkType != "F":
			t.Errorf("ChunkType doesn't match. Want: %s, Got: %s", "F", chunkType)
		}
	})
	t.Run("RHE", func(t *testing.T) {
		t.Parallel()
		u, err := Decode(testReverseHelloBytes[0])
		if err != nil {
			t.Fatalf("Failed to decode UACP: %s", err)
		}

		if _, ok := u.(*ReverseHello); !ok {
			t.Errorf("Decoded as wrong type. Want: %s, Got: %T", "*ReverseHello", u)
		}

		var (
			msgType   = u.MessageTypeValue()
			chunkType = u.ChunkTypeValue()
		)
		switch {
		case msgType != MessageTypeReverseHello:
			t.Errorf("MessageType doesn't match. Want: %s, Got: %s", MessageTypeReverseHello, msgType)
		case chunkType != "F":
			t.Errorf("ChunkType doesn't match. Want: %s, Got: %s", "F", chunkType)
		}
	})
	t.Run("XXX", func(t *testing.T) {
		t.Parallel()
		u, err := Decode(testGenericBytes[0])
		if err != nil {
			t.Fatalf("Failed to decode UACP: %s", err)
		}

		if _, ok := u.(*Generic); !ok {
			t.Errorf("Decoded as wrong type. Want: %s, Got: %T", "*Generic", u)
		}

		var (
			msgType   = u.MessageTypeValue()
			chunkType = u.ChunkTypeValue()
		)
		switch {
		case msgType != "XXX":
			t.Errorf("MessageType doesn't match. Want: %s, Got: %s", "XXX", msgType)
		case chunkType != "X":
			t.Errorf("ChunkType doesn't match. Want: %s, Got: %s", "X", chunkType)
		}
	})
}
