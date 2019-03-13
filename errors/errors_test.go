// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// XXX - Implement!
package errors

import (
	"testing"

	"github.com/pascaldekloe/goe/verify"
)

var dummy string

func TestNewErrors(t *testing.T) {
	t.Run("ErrTooShortToDecode", func(t *testing.T) {
		got := NewErrTooShortToDecode(dummy, "should be XXX.")
		want := &ErrTooShortToDecode{
			Type:    dummy,
			Message: "should be XXX.",
		}
		verify.Values(t, "", got, want)
	})
	t.Run("ErrInvalidLength", func(t *testing.T) {
		got := NewErrInvalidLength(dummy, "should be XXX.")
		want := &ErrInvalidLength{
			Type:    dummy,
			Message: "should be XXX.",
		}
		verify.Values(t, "", got, want)
	})
	t.Run("ErrUnsupported", func(t *testing.T) {
		got := NewErrUnsupported(dummy, "XXX is not supported yet.")
		want := &ErrUnsupported{
			Type:    dummy,
			Message: "XXX is not supported yet.",
		}
		verify.Values(t, "", got, want)
	})
	t.Run("ErrInvalidType", func(t *testing.T) {
		got := NewErrInvalidType(dummy, "decode", "something's wrong.")
		want := &ErrInvalidType{
			Type:    dummy,
			Action:  "decode",
			Message: "something's wrong.",
		}
		verify.Values(t, "", got, want)
	})
	t.Run("ErrReceiverNil", func(t *testing.T) {
		got := NewErrReceiverNil(dummy)
		want := &ErrReceiverNil{
			Type: dummy,
		}
		verify.Values(t, "", got, want)
	})
}

func TestError(t *testing.T) {
	t.Run("ErrTooShortToDecode", func(t *testing.T) {
		got := NewErrTooShortToDecode(dummy, "should be XXX.").Error()
		want := "too short to decode as string: should be XXX."
		verify.Values(t, "", got, want)
	})
	t.Run("ErrInvalidLength", func(t *testing.T) {
		got := NewErrInvalidLength(dummy, "should be XXX.").Error()
		want := "got invalid Length in string: should be XXX."
		verify.Values(t, "", got, want)
	})
	t.Run("ErrUnsupported", func(t *testing.T) {
		got := NewErrUnsupported(dummy, "XXX is not supported yet.").Error()
		want := "unsupported string: XXX is not supported yet."
		verify.Values(t, "", got, want)
	})
	t.Run("ErrInvalidType", func(t *testing.T) {
		got := NewErrInvalidType(dummy, "decode", "something's wrong.").Error()
		want := "cannot decode as string: something's wrong."
		verify.Values(t, "", got, want)
	})
	t.Run("ErrReceiverNil", func(t *testing.T) {
		got := NewErrReceiverNil(dummy).Error()
		want := "Receiver string is nil."
		verify.Values(t, "", got, want)
	})
}
