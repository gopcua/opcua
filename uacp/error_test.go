// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestError(t *testing.T) {
	cases := []codectest.Case{
		{
			Struct: NewError(
				BadSecureChannelClosed, // Error
				"foobar",
			),
			Bytes: []byte{
				// Error: BadSecureChannelClosed
				0x00, 0x00, 0x86, 0x80,
				// Reason: dummy
				0x06, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72,
			},
		},
	}
	codectest.Run(t, cases)
}
