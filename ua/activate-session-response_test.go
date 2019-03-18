// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"testing"
)

func TestActivateSessionResponse(t *testing.T) {
	cases := []CodecTestCase{
		{ // Without dummy nonce, results nor diags
			Name: "nothing",
			Struct: &ActivateSessionResponse{
				ResponseHeader: NewNullResponseHeader(),
			},
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
			Struct: &ActivateSessionResponse{
				ResponseHeader: NewNullResponseHeader(),
				ServerNonce:    []byte{0xde, 0xad, 0xbe, 0xef},
			},
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
	RunCodecTest(t, cases)
}
