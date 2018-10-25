// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestByteString(t *testing.T) {
	cases := []codectest.Case{
		{
			Struct: NewByteString([]byte{0xde, 0xad, 0xbe, 0xef}),
			Bytes: []byte{
				// length
				0x04, 0x00, 0x00, 0x00,
				// value
				0xde, 0xad, 0xbe, 0xef,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeByteString(b)
	})
}
