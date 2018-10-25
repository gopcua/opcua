// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// XXX - Implement!
package errors

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var dummy string

func TestNewErrors(t *testing.T) {
	t.Run("ErrTooShortToDecode", func(t *testing.T) {
		e := NewErrTooShortToDecode(dummy, "should be XXX.")
		expected := &ErrTooShortToDecode{
			Type:    dummy,
			Message: "should be XXX.",
		}

		if diff := cmp.Diff(e, expected); diff != "" {
			t.Error(diff)
		}
	})
	t.Run("ErrInvalidLength", func(t *testing.T) {
		e := NewErrInvalidLength(dummy, "should be XXX.")
		expected := &ErrInvalidLength{
			Type:    dummy,
			Message: "should be XXX.",
		}

		if diff := cmp.Diff(e, expected); diff != "" {
			t.Error(diff)
		}
	})
	t.Run("ErrUnsupported", func(t *testing.T) {
		e := NewErrUnsupported(dummy, "XXX is not supported yet.")
		expected := &ErrUnsupported{
			Type:    dummy,
			Message: "XXX is not supported yet.",
		}

		if diff := cmp.Diff(e, expected); diff != "" {
			t.Error(diff)
		}
	})
	t.Run("ErrInvalidType", func(t *testing.T) {
		e := NewErrInvalidType(dummy, "decode", "something's wrong.")
		expected := &ErrInvalidType{
			Type:    dummy,
			Action:  "decode",
			Message: "something's wrong.",
		}

		if diff := cmp.Diff(e, expected); diff != "" {
			t.Error(diff)
		}
	})
	t.Run("ErrReceiverNil", func(t *testing.T) {
		e := NewErrReceiverNil(dummy)
		expected := &ErrReceiverNil{
			Type: dummy,
		}

		if diff := cmp.Diff(e, expected); diff != "" {
			t.Error(diff)
		}
	})
}

func TestError(t *testing.T) {
	t.Run("ErrTooShortToDecode", func(t *testing.T) {
		e := NewErrTooShortToDecode(dummy, "should be XXX.").Error()
		expected := "too short to decode as string: should be XXX."

		if diff := cmp.Diff(e, expected); diff != "" {
			t.Error(diff)
		}
	})
	t.Run("ErrInvalidLength", func(t *testing.T) {
		e := NewErrInvalidLength(dummy, "should be XXX.").Error()
		expected := "got invalid Length in string: should be XXX."

		if diff := cmp.Diff(e, expected); diff != "" {
			t.Error(diff)
		}
	})
	t.Run("ErrUnsupported", func(t *testing.T) {
		e := NewErrUnsupported(dummy, "XXX is not supported yet.").Error()
		expected := "unsupported string: XXX is not supported yet."

		if diff := cmp.Diff(e, expected); diff != "" {
			t.Error(diff)
		}
	})
	t.Run("ErrInvalidType", func(t *testing.T) {
		e := NewErrInvalidType(dummy, "decode", "something's wrong.").Error()
		expected := "cannot decode as string: something's wrong."

		if diff := cmp.Diff(e, expected); diff != "" {
			t.Error(diff)
		}
	})
	t.Run("ErrReceiverNil", func(t *testing.T) {
		e := NewErrReceiverNil(dummy).Error()
		expected := "Receiver string is nil."

		if diff := cmp.Diff(e, expected); diff != "" {
			t.Error(diff)
		}
	})
}
