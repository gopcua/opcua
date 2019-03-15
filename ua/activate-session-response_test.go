// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"testing"

	"github.com/gopcua/opcua/utils/codectest"
)

func TestActivateSessionResponse(t *testing.T) {
	cases := []codectest.Case{
		{ // Without dummy nonce, results nor diags
			Name: "nothing",
			Struct: NewActivateSessionResponse(
				NewNullResponseHeader(),
				nil,
				nil,
				nil,
			),
			Bytes: flatten(
				nullResponseHeaderBytes,
				[]byte{
					// ServerNonce
					0xff, 0xff, 0xff, 0xff,
					// Results
					0xff, 0xff, 0xff, 0xff,
					// DiagnosticInfos
					0xff, 0xff, 0xff, 0xff,
				}),
		}, { // With dummy nonce, no results and diags
			Name: "with-nonce",
			Struct: NewActivateSessionResponse(
				NewNullResponseHeader(),
				[]byte{0xde, 0xad, 0xbe, 0xef},
				nil,
				nil,
			),
			Bytes: flatten(
				nullResponseHeaderBytes,
				[]byte{
					// ServerNonce
					0x04, 0x00, 0x00, 0x00,
					0xde, 0xad, 0xbe, 0xef,
					// Results
					0xff, 0xff, 0xff, 0xff,
					// DiagnosticInfos
					0xff, 0xff, 0xff, 0xff,
				}),
		},
	}
	codectest.Run(t, cases)
}
