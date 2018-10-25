// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestGUID(t *testing.T) {
	cases := []codectest.Case{
		{
			Name:   "ok",
			Struct: NewGUID("AAAABBBB-CCDD-EEFF-0101-0123456789AB"),
			Bytes: []byte{
				0xbb, 0xbb, 0xaa, 0xaa, 0xdd, 0xcc, 0xff, 0xee,
				0xab, 0x89, 0x67, 0x45, 0x23, 0x01, 0x01, 0x01,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeGUID(b)
	})
}
