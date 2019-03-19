// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"testing"
)

func TestActivateSessionRequest(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name: "normal",
			Struct: &ActivateSessionRequest{
				RequestHeader:              NewNullRequestHeader(),
				ClientSignature:            &SignatureData{},
				ClientSoftwareCertificates: nil,
				LocaleIDs:                  nil,
				UserIdentityToken:          NewExtensionObject(&AnonymousIdentityToken{PolicyID: "anonymous"}),
				UserTokenSignature:         &SignatureData{},
			},
			Bytes: flatten(
				nullRequestHeaderBytes,
				[]byte{
					// ClientSignature
					0xff, 0xff, 0xff, 0xff,
					// ClientSoftwareCertificates
					0xff, 0xff, 0xff, 0xff,
					// Algorithm
					0xff, 0xff, 0xff, 0xff,
					// Signature
					0xff, 0xff, 0xff, 0xff,
					// UserIdentityToken
					// TypeID
					0x01, 0x00, 0x41, 0x01,
					// EncodingMask
					0x01,
					// Length
					0x0d, 0x00, 0x00, 0x00,
					// AnonymousIdentityToken
					0x09, 0x00, 0x00, 0x00, 0x61, 0x6e, 0x6f, 0x6e, 0x79, 0x6d, 0x6f, 0x75, 0x73,
					// UserTokenSignature
					0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				}),
		},
	}
	RunCodecTest(t, cases)
}
