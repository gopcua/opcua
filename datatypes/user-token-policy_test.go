// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

var userTokenPolicyCases = []struct {
	description string
	structured  *UserTokenPolicy
	serialized  []byte
}{
	{
		"normal",
		NewUserTokenPolicy(
			"1", UserTokenAnonymous,
			"issued-token", "issuer-uri", "sec-uri",
		),
		[]byte{
			// PolicyID
			0x01, 0x00, 0x00, 0x00, 0x31,
			// TokenType
			0x00, 0x00, 0x00, 0x00,
			// IssuedTokenType
			0x0c, 0x00, 0x00, 0x00, 0x69, 0x73, 0x73, 0x75, 0x65, 0x64, 0x2d, 0x74, 0x6f, 0x6b, 0x65, 0x6e,
			// IssuerEndpointURI
			0x0a, 0x00, 0x00, 0x00, 0x69, 0x73, 0x73, 0x75, 0x65, 0x72, 0x2d, 0x75, 0x72, 0x69,
			// SecurityPolicyURI
			0x07, 0x00, 0x00, 0x00, 0x73, 0x65, 0x63, 0x2d, 0x75, 0x72, 0x69,
		},
	},
}

func TestDecodeUserTokenPolicy(t *testing.T) {
	for _, c := range userTokenPolicyCases {
		got, err := DecodeUserTokenPolicy(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeUserTokenPolicy(t *testing.T) {
	for _, c := range userTokenPolicyCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestUserTokenPolicyLen(t *testing.T) {
	for _, c := range userTokenPolicyCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

var userTokenPolicyArrayCases = []struct {
	description string
	structured  *UserTokenPolicyArray
	serialized  []byte
}{
	{
		"empty",
		NewUserTokenPolicyArray(nil),
		[]byte{
			0x00, 0x00, 0x00, 0x00,
		},
	},
	{
		"normal",
		NewUserTokenPolicyArray(
			[]*UserTokenPolicy{
				NewUserTokenPolicy(
					"1", UserTokenAnonymous,
					"issued-token", "issuer-uri", "sec-uri",
				),
				NewUserTokenPolicy(
					"1", UserTokenAnonymous,
					"issued-token", "issuer-uri", "sec-uri",
				),
			},
		),
		[]byte{
			// ArraySize
			0x02, 0x00, 0x00, 0x00,
			// PolicyID
			0x01, 0x00, 0x00, 0x00, 0x31,
			// TokenType
			0x00, 0x00, 0x00, 0x00,
			// IssuedTokenType
			0x0c, 0x00, 0x00, 0x00, 0x69, 0x73, 0x73, 0x75, 0x65, 0x64, 0x2d, 0x74, 0x6f, 0x6b, 0x65, 0x6e,
			// IssuerEndpointURI
			0x0a, 0x00, 0x00, 0x00, 0x69, 0x73, 0x73, 0x75, 0x65, 0x72, 0x2d, 0x75, 0x72, 0x69,
			// SecurityPolicyURI
			0x07, 0x00, 0x00, 0x00, 0x73, 0x65, 0x63, 0x2d, 0x75, 0x72, 0x69,
			// PolicyID
			0x01, 0x00, 0x00, 0x00, 0x31,
			// TokenType
			0x00, 0x00, 0x00, 0x00,
			// IssuedTokenType
			0x0c, 0x00, 0x00, 0x00, 0x69, 0x73, 0x73, 0x75, 0x65, 0x64, 0x2d, 0x74, 0x6f, 0x6b, 0x65, 0x6e,
			// IssuerEndpointURI
			0x0a, 0x00, 0x00, 0x00, 0x69, 0x73, 0x73, 0x75, 0x65, 0x72, 0x2d, 0x75, 0x72, 0x69,
			// SecurityPolicyURI
			0x07, 0x00, 0x00, 0x00, 0x73, 0x65, 0x63, 0x2d, 0x75, 0x72, 0x69,
		},
	},
}

func TestDecodeUserTokenPolicyArray(t *testing.T) {
	for _, c := range userTokenPolicyArrayCases {
		got, err := DecodeUserTokenPolicyArray(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeUserTokenPolicyArray(t *testing.T) {
	for _, c := range userTokenPolicyArrayCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestUserTokenPolicyArrayLen(t *testing.T) {
	for _, c := range userTokenPolicyArrayCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
