// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"log"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	testBooleanTrue  = []byte{0x01}
	testBooleanFalse = []byte{0x00}
)

func TestDecodeBoolean(t *testing.T) {
	t.Run("TRUE", func(t *testing.T) {
		bo, err := DecodeBoolean(testBooleanTrue)
		if err != nil {
			t.Fatal(err)
		}

		expected := &Boolean{Value: 0x01}
		log.Printf("%d, %d", bo, expected)
		if diff := cmp.Diff(bo, expected); diff != "" {
			t.Error(bo)
		}
	})
	t.Run("FALSE", func(t *testing.T) {
		bo, err := DecodeBoolean(testBooleanFalse)
		if err != nil {
			t.Fatal(err)
		}

		expected := &Boolean{Value: 0x00}
		log.Printf("%d, %d", bo, expected)
		if diff := cmp.Diff(bo, expected); diff != "" {
			t.Error(bo)
		}
	})
}

func TestBooleanDecodeFromBytes(t *testing.T) {
	t.Run("TRUE", func(t *testing.T) {
		bo := &Boolean{}
		if err := bo.DecodeFromBytes(testBooleanTrue); err != nil {
			t.Fatal(err)
		}

		expected := &Boolean{Value: 0x01}
		log.Printf("%d, %d", bo, expected)
		if diff := cmp.Diff(bo, expected); diff != "" {
			t.Error(bo)
		}
	})
	t.Run("FALSE", func(t *testing.T) {
		bo := &Boolean{}
		if err := bo.DecodeFromBytes(testBooleanFalse); err != nil {
			t.Fatal(err)
		}

		expected := &Boolean{Value: 0x00}
		log.Printf("%d, %d", bo, expected)
		if diff := cmp.Diff(bo, expected); diff != "" {
			t.Error(bo)
		}
	})
}

func TestBooleanSerialize(t *testing.T) {
	t.Run("TRUE", func(t *testing.T) {
		bo := &Boolean{Value: 0x01}
		b, err := bo.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(b, testBooleanTrue); diff != "" {
			t.Error(diff)
		}
	})
	t.Run("FALSE", func(t *testing.T) {
		bo := &Boolean{Value: 0x00}
		b, err := bo.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(b, testBooleanFalse); diff != "" {
			t.Error(diff)
		}
	})
}

func TestBooleanSerializeTo(t *testing.T) {
	t.Run("TRUE", func(t *testing.T) {
		bo := &Boolean{Value: 0x01}
		b := make([]byte, bo.Len())
		if err := bo.SerializeTo(b); err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(b, testBooleanTrue); diff != "" {
			t.Error(diff)
		}
	})
	t.Run("FALSE", func(t *testing.T) {
		bo := &Boolean{Value: 0x00}
		b := make([]byte, bo.Len())
		if err := bo.SerializeTo(b); err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(b, testBooleanFalse); diff != "" {
			t.Error(diff)
		}
	})
}

func TestBooleanLen(t *testing.T) {
	bo := &Boolean{Value: 0x01}
	if bo.Len() != 1 {
		t.Errorf("Len doesn't match. Want: %d, Got: %d", 1, bo.Len())
	}
}

func TestBooleanString(t *testing.T) {
	t.Run("TRUE", func(t *testing.T) {
		bo := &Boolean{Value: 0x01}
		if bo.String() != "TRUE" {
			t.Errorf("Len doesn't match. Want: %s, Got: %s", "TRUE", bo.String())
		}
	})
	t.Run("FALSE", func(t *testing.T) {
		bo := &Boolean{Value: 0x00}
		if bo.String() != "FALSE" {
			t.Errorf("Len doesn't match. Want: %s, Got: %s", "FALSE", bo.String())
		}
	})
}
