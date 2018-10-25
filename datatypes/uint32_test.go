// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestUint32Array(t *testing.T) {
	cases := []codectest.Case{
		{
			Name:   "No contents",
			Struct: NewUint32Array(nil),
			Bytes: []byte{
				// todo(fs): this should be 0xffffffff
				0x00, 0x00, 0x00, 0x00,
			},
		},
		{
			Name:   "1 value",
			Struct: NewUint32Array([]uint32{1}),
			Bytes: []byte{
				// length
				0x01, 0x00, 0x00, 0x00,
				// val[0]
				0x01, 0x00, 0x00, 0x00,
			},
		},
		{
			Name:   "4 values",
			Struct: NewUint32Array([]uint32{1, 2, 3, 4}),
			Bytes: []byte{
				// length
				0x04, 0x00, 0x00, 0x00,
				// val[0]
				0x01, 0x00, 0x00, 0x00,
				// val[1]
				0x02, 0x00, 0x00, 0x00,
				// val[2]
				0x03, 0x00, 0x00, 0x00,
				// val[3]
				0x04, 0x00, 0x00, 0x00,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeUint32Array(b)
	})
}
