// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import "testing"

var testUserTokenPolicyBytes = [][]byte{
	{ // Single
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
	{ // Multiple
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
	{},
	{},
	{},
	{},
}

func TestDecodeUserTokenPolicy(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		t.Parallel()

		u, err := DecodeUserTokenPolicy(testUserTokenPolicyBytes[0])
		if err != nil {
			t.Fatalf("Failed to decode UserTokenPolicy: %s", err)
		}

		switch {
		case u.PolicyID.Get() != "1":
			t.Errorf("PolicyID doesn't match. Want: %s, Got: %s", "1", u.PolicyID.Get())
		case u.TokenType != UserTokenAnonymous:
			t.Errorf("IssuedTokenType doesn't match. Want: %d, Got: %d", UserTokenAnonymous, u.TokenType)
		case u.IssuedTokenType.Get() != "issued-token":
			t.Errorf("IssuedTokenType doesn't match. Want: %s, Got: %s", "issued-token", u.IssuedTokenType.Get())
		case u.IssuerEndpointURI.Get() != "issuer-uri":
			t.Errorf("IssuerEndpointURI doesn't match. Want: %s, Got: %s", "issuer-uri", u.IssuerEndpointURI.Get())
		case u.SecurityPolicyURI.Get() != "sec-uri":
			t.Errorf("SecurityPolicyURI doesn't match. Want: %s, Got: %s", "sec-uri", u.SecurityPolicyURI.Get())
		}
		t.Log(u.String())
	})
	t.Run("multiple", func(t *testing.T) {
		t.Parallel()

		u, err := DecodeUserTokenPolicyArray(testUserTokenPolicyBytes[1])
		if err != nil {
			t.Fatalf("Failed to decode UserTokenPolicy: %s", err)
		}

		for _, ut := range u.UserTokenPolicies {
			switch {
			case ut.PolicyID.Get() != "1":
				t.Errorf("PolicyID doesn't match. Want: %s, Got: %s", "1", ut.PolicyID.Get())
			case ut.TokenType != UserTokenAnonymous:
				t.Errorf("IssuedTokenType doesn't match. Want: %d, Got: %d", UserTokenAnonymous, ut.TokenType)
			case ut.IssuedTokenType.Get() != "issued-token":
				t.Errorf("IssuedTokenType doesn't match. Want: %s, Got: %s", "issued-token", ut.IssuedTokenType.Get())
			case ut.IssuerEndpointURI.Get() != "issuer-uri":
				t.Errorf("IssuerEndpointURI doesn't match. Want: %s, Got: %s", "issuer-uri", ut.IssuerEndpointURI.Get())
			case ut.SecurityPolicyURI.Get() != "sec-uri":
				t.Errorf("SecurityPolicyURI doesn't match. Want: %s, Got: %s", "sec-uri", ut.SecurityPolicyURI.Get())
			}
			t.Log(ut)
		}
	})
}

func TestSerializeUserTokenPolicy(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		t.Parallel()

		u := NewUserTokenPolicy(
			"1", UserTokenAnonymous,
			"issued-token", "issuer-uri", "sec-uri",
		)

		serialized, err := u.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize Service: %s", err)
		}

		for i, s := range serialized {
			x := testUserTokenPolicyBytes[0][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("multiple", func(t *testing.T) {
		t.Parallel()

		u := NewUserTokenPolicyArray(
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
		)

		serialized, err := u.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize Service: %s", err)
		}

		for i, s := range serialized {
			x := testUserTokenPolicyBytes[1][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
}
