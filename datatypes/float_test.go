// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewFloat(t *testing.T) {
	f := NewFloat(5.00078)
	expected := &Float{
		Value: 5.00078,
	}
	if diff := cmp.Diff(f, expected); diff != "" {
		t.Error(diff)
	}
}

func TestDecodeFloat(t *testing.T) {
	b := []byte{0x64, 0x06, 0xa0, 0x40}
	f, err := DecodeFloat(b)
	if err != nil {
		t.Fatal(err)
	}
	expected := &Float{
		Value: 5.00078,
	}
	if diff := cmp.Diff(f, expected); diff != "" {
		t.Error(diff)
	}
}

func TestFloatDecodeFromBytes(t *testing.T) {
	f := &Float{}
	b := []byte{0x64, 0x06, 0xa0, 0x40}
	if err := f.DecodeFromBytes(b); err != nil {
		t.Fatal(err)
	}
	expected := &Float{
		Value: 5.00078,
	}
	if diff := cmp.Diff(f, expected); diff != "" {
		t.Error(diff)
	}
}

func TestFloatSerialize(t *testing.T) {
	f := &Float{
		Value: 5.00078,
	}
	b, err := f.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte{0x64, 0x06, 0xa0, 0x40}
	if diff := cmp.Diff(b, expected); diff != "" {
		t.Error(diff)
	}
}

func TestFloatSerializeTo(t *testing.T) {
	f := &Float{
		Value: 5.00078,
	}
	b := make([]byte, f.Len())
	if err := f.SerializeTo(b); err != nil {
		t.Fatal(err)
	}
	expected := []byte{0x64, 0x06, 0xa0, 0x40}
	if diff := cmp.Diff(b, expected); diff != "" {
		t.Error(diff)
	}
}

func TestFloatLen(t *testing.T) {
	f := &Float{}
	if f.Len() != 4 {
		t.Errorf("Len doesn't match. Want: %d, Got: %d", 4, f.Len())
	}
}
