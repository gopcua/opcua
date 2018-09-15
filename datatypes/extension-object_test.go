// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewExtensionObject(t *testing.T) {
	e := NewExtensionObject(
		NewExpandedNodeID(
			false, false, NewFourByteNodeID(0, 321), "", 0,
		),
		0x01,
		[]byte("0"),
	)
	expected := &ExtensionObject{
		TypeID: &ExpandedNodeID{
			NodeID: &FourByteNodeID{
				EncodingMask: 0x01,
				Namespace:    0,
				Identifier:   321,
			},
		},
		EncodingMask: 0x01,
		Length:       5,
		Body:         NewByteString([]byte("0")),
	}

	if diff := cmp.Diff(e, expected); diff != "" {
		t.Error(diff)
	}
}

func TestDecodeExtensionObject(t *testing.T) {
	b := []byte{
		0x01, 0x00, 0x41, 0x01, 0x01, 0x05, 0x00, 0x00,
		0x00, 0x01, 0x00, 0x00, 0x00, 0x30,
	}
	e, err := DecodeExtensionObject(b)
	if err != nil {
		t.Fatal(err)
	}
	expected := &ExtensionObject{
		TypeID: &ExpandedNodeID{
			NodeID: &FourByteNodeID{
				EncodingMask: 0x01,
				Namespace:    0,
				Identifier:   321,
			},
		},
		EncodingMask: 0x01,
		Length:       5,
		Body:         NewByteString([]byte("0")),
	}
	if diff := cmp.Diff(e, expected); diff != "" {
		t.Error(diff)
	}
}

func TestExtensionObjectDecodeFromBytes(t *testing.T) {
	b := []byte{
		0x01, 0x00, 0x41, 0x01, 0x01, 0x05, 0x00, 0x00,
		0x00, 0x01, 0x00, 0x00, 0x00, 0x30,
	}
	e := &ExtensionObject{}
	if err := e.DecodeFromBytes(b); err != nil {
		t.Fatal(err)
	}
	expected := &ExtensionObject{
		TypeID: &ExpandedNodeID{
			NodeID: &FourByteNodeID{
				EncodingMask: 0x01,
				Namespace:    0,
				Identifier:   321,
			},
		},
		EncodingMask: 0x01,
		Length:       5,
		Body:         NewByteString([]byte("0")),
	}
	if diff := cmp.Diff(e, expected); diff != "" {
		t.Error(diff)
	}
}

func TestExtensionObjectSerialize(t *testing.T) {
	e := &ExtensionObject{
		TypeID: &ExpandedNodeID{
			NodeID: &FourByteNodeID{
				EncodingMask: 0x01,
				Namespace:    0,
				Identifier:   321,
			},
		},
		EncodingMask: 0x01,
		Length:       5,
		Body:         NewByteString([]byte("0")),
	}
	b, err := e.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte{
		0x01, 0x00, 0x41, 0x01, 0x01, 0x05, 0x00, 0x00,
		0x00, 0x01, 0x00, 0x00, 0x00, 0x30,
	}
	if diff := cmp.Diff(b, expected); diff != "" {
		t.Error(diff)
	}
}

func TestExtensionObjectSerializeTo(t *testing.T) {
	e := &ExtensionObject{
		TypeID: &ExpandedNodeID{
			NodeID: &FourByteNodeID{
				EncodingMask: 0x01,
				Namespace:    0,
				Identifier:   321,
			},
		},
		EncodingMask: 0x01,
		Length:       5,
		Body:         NewByteString([]byte("0")),
	}
	b := make([]byte, e.Len())
	if err := e.SerializeTo(b); err != nil {
		t.Fatal(err)
	}
	expected := []byte{
		0x01, 0x00, 0x41, 0x01, 0x01, 0x05, 0x00, 0x00,
		0x00, 0x01, 0x00, 0x00, 0x00, 0x30,
	}
	if diff := cmp.Diff(b, expected); diff != "" {
		t.Error(diff)
	}
}

func TestExtensionObjectLen(t *testing.T) {
	e := &ExtensionObject{
		TypeID: &ExpandedNodeID{
			NodeID: &FourByteNodeID{
				EncodingMask: 0x01,
				Namespace:    0,
				Identifier:   321,
			},
		},
		EncodingMask: 0x01,
		Length:       5,
		Body:         NewByteString([]byte("0")),
	}
	if e.Len() != 14 {
		t.Errorf("Len doesn't match. Want: %d, Got: %d", 14, e.Len())
	}
}
