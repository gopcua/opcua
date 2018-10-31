// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestQualifiedName(t *testing.T) {
	cases := []codectest.Case{
		{
			Name:   "normal",
			Struct: NewQualifiedName(1, "foobar"),
			Bytes: []byte{
				0x01,
				0x00,
				0x06, 0x00, 0x00, 0x00,
				0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72,
			},
		},
		{
			Name:   "empty",
			Struct: NewQualifiedName(1, ""),
			Bytes: []byte{
				0x01,
				0x00,
				0xff, 0xff, 0xff, 0xff,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeQualifiedName(b)
	})
}
