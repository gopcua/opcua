// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// XXX - Implement!
package errors

import "testing"

func raiseErrTooShortToDecode() error {
	return &ErrTooShortToDecode{
		Type: "",
		Msg:  "too short to decode",
	}
}

func TestErrors(t *testing.T) {
	err := raiseErrTooShortToDecode()

	if _, ok := err.(*ErrTooShortToDecode); !ok {
		t.Fatalf("Failed to assert type: %T", err)
	}
}
