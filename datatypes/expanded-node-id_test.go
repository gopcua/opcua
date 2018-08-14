// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"
)

var testExpandedNodeIDBytes = [][]byte{
	{ // Without optional fields
		0x00, 0xff,
	},
	{ // With NamespaceURI
		0x80, 0xff, 0x06, 0x00, 0x00, 0x00, 0x66, 0x6f,
		0x6f, 0x62, 0x61, 0x72,
	},
	{ // With ServerIndex
		0x40, 0xff, 0x00, 0x80, 0x00, 0x00,
	},
	{ // With NamespaceURI and ServerIndex
		0xc0, 0xff, 0x06, 0x00, 0x00, 0x00, 0x66, 0x6f,
		0x6f, 0x62, 0x61, 0x72, 0x00, 0x80, 0x00, 0x00,
	},
}

func TestDecodeExpandedNodeID(t *testing.T) {
	t.Run("no-options", func(t *testing.T) {
		t.Parallel()
		e, err := DecodeExpandedNodeID(testExpandedNodeIDBytes[0])
		if err != nil {
			t.Fatalf("Failed to decode ExpandedNodeID: %s", err)
		}

		switch {
		case e.HasNamespaceURI():
			t.Errorf("URI Flag doesn't match. Want: %v, Got: %v", false, e.HasNamespaceURI())
		case e.HasServerIndex():
			t.Errorf("Index Flag doesn't match. Want: %v, Got: %v", false, e.HasServerIndex())
		}
		t.Log(e.String())
	})
	t.Run("has-uri", func(t *testing.T) {
		t.Parallel()
		e, err := DecodeExpandedNodeID(testExpandedNodeIDBytes[1])
		if err != nil {
			t.Fatalf("Failed to decode ExpandedNodeID: %s", err)
		}

		switch {
		case !e.HasNamespaceURI():
			t.Errorf("URI Flag doesn't match. Want: %v, Got: %v", true, e.HasNamespaceURI())
		case e.HasServerIndex():
			t.Errorf("Index Flag doesn't match. Want: %v, Got: %v", false, e.HasServerIndex())
		case e.NamespaceURI.Get() != "foobar":
			t.Errorf("NamespaceURI doesn't match. Want: %s, Got: %s", "foobar", e.NamespaceURI.Get())
		}
		t.Log(e.String())
	})
	t.Run("has-index", func(t *testing.T) {
		t.Parallel()
		e, err := DecodeExpandedNodeID(testExpandedNodeIDBytes[2])
		if err != nil {
			t.Fatalf("Failed to decode ExpandedNodeID: %s", err)
		}

		switch {
		case e.HasNamespaceURI():
			t.Errorf("URI Flag doesn't match. Want: %v, Got: %v", false, e.HasNamespaceURI())
		case !e.HasServerIndex():
			t.Errorf("Index Flag doesn't match. Want: %v, Got: %v", true, e.HasServerIndex())
		case e.ServerIndex != 32768:
			t.Errorf("ServerIndex doesn't match. Want: %d, Got: %d", 32768, e.ServerIndex)
		}
		t.Log(e.String())
	})
	t.Run("has-both", func(t *testing.T) {
		t.Parallel()
		e, err := DecodeExpandedNodeID(testExpandedNodeIDBytes[3])
		if err != nil {
			t.Fatalf("Failed to decode ExpandedNodeID: %s", err)
		}

		switch {
		case !e.HasNamespaceURI():
			t.Errorf("URI Flag doesn't match. Want: %v, Got: %v", true, e.HasNamespaceURI())
		case !e.HasServerIndex():
			t.Errorf("Index Flag doesn't match. Want: %v, Got: %v", true, e.HasServerIndex())
		case e.NamespaceURI.Get() != "foobar":
			t.Errorf("NamespaceURI doesn't match. Want: %s, Got: %s", "foobar", e.NamespaceURI.Get())
		case e.ServerIndex != 32768:
			t.Errorf("ServerIndex doesn't match. Want: %d, Got: %d", 32768, e.ServerIndex)
		}
		t.Log(e.String())
	})
}

func TestSerializeExpandedNodeID(t *testing.T) {
	t.Run("no-options", func(t *testing.T) {
		t.Parallel()
		e := NewExpandedNodeID(
			false, false,
			NewTwoByteNodeID(0xff),
			"", 0,
		)

		serialized, err := e.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize ExpandedNodeID: %s", err)
		}

		for i, s := range serialized {
			x := testExpandedNodeIDBytes[0][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("has-uri", func(t *testing.T) {
		t.Parallel()
		e := NewExpandedNodeID(
			true, false,
			NewTwoByteNodeID(0xff),
			"foobar", 0,
		)

		serialized, err := e.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize ExpandedNodeID: %s", err)
		}

		for i, s := range serialized {
			x := testExpandedNodeIDBytes[1][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("has-index", func(t *testing.T) {
		t.Parallel()
		e := NewExpandedNodeID(
			false, true,
			NewTwoByteNodeID(0xff),
			"", 32768,
		)

		serialized, err := e.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize ExpandedNodeID: %s", err)
		}

		for i, s := range serialized {
			x := testExpandedNodeIDBytes[2][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("has-both", func(t *testing.T) {
		t.Parallel()
		e := NewExpandedNodeID(
			true, true,
			NewTwoByteNodeID(0xff),
			"foobar", 32768,
		)

		serialized, err := e.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize ExpandedNodeID: %s", err)
		}

		for i, s := range serialized {
			x := testExpandedNodeIDBytes[3][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
}
