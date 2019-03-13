// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package utils

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var thumbprintCases = []struct {
	description string
	cert        string
	hash        []byte
}{
	{
		"pass",
		"testfiles/example-cert.der",
		[]byte{
			0x9f, 0x59, 0x73, 0xbd, 0x92, 0x32, 0xe7, 0x5e,
			0xf9, 0x01, 0x7b, 0xd9, 0x2d, 0x95, 0xf3, 0x43,
			0x79, 0x54, 0xd6, 0x8c,
		},
	},
}

func TestNewThumbprintFromCert(t *testing.T) {
	for _, c := range thumbprintCases {
		got, err := NewThumbprintFromCert(c.cert)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.hash); diff != "" {
			t.Error(diff)
		}
	}
}
