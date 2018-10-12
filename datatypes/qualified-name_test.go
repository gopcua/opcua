// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var qualifiedNameCases = []struct {
	description string
	structured  *QualifiedName
	serialized  []byte
}{
	{
		"normal",
		NewQualifiedName(1, "foobar"),
		[]byte{
			// NamespaceIndex
			0x01, 0x00,
			// String
			0x06, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72,
		},
	},
}

func TestDecodeQualifiedName(t *testing.T) {
	for _, c := range qualifiedNameCases {
		got, err := DecodeQualifiedName(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeQualifiedName(t *testing.T) {
	for _, c := range qualifiedNameCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestQualifiedNameLen(t *testing.T) {
	for _, c := range qualifiedNameCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
