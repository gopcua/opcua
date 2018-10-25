// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestBoolean(t *testing.T) {
	cases := []codectest.Case{
		{
			Name:   "true",
			Struct: NewBoolean(true),
			Bytes:  []byte{0x01},
		},
		{
			Name:   "false",
			Struct: NewBoolean(false),
			Bytes:  []byte{0x00},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeBoolean(b)
	})
}
