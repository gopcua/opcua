// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import "testing"

var testChannelSecurityTokenBytes = [][]byte{
	{
		// ChannelID
		0x01, 0x00, 0x00, 0x00,
		// TokenID
		0x02, 0x00, 0x00, 0x00,
		// CreatedAt
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// RevisedLifetime
		0x80, 0x8d, 0x5b, 0x00,
	},
	{},
	{},
}

func TestDecodeChannelSecurityToken(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		t.Parallel()
		c, err := DecodeChannelSecurityToken(testChannelSecurityTokenBytes[0])
		if err != nil {
			t.Fatalf("Failed to decode ChannelSecurityToken: %s", err)
		}

		switch {
		case c.ChannelID != 1:
			t.Errorf("ChannelID doesn't match. Want: %d, Got: %d", 1, c.ChannelID)
		case c.TokenID != 2:
			t.Errorf("TokenID doesn't match. Want: %d, Got: %d", 2, c.TokenID)
		case c.CreatedAt != 1:
			t.Errorf("CreatedAt doesn't match. Want: %d, Got: %d", 1, c.CreatedAt)
		case c.RevisedLifetime != 6000000:
			t.Errorf("RevisedLifetime doesn't match. Want: %d, Got: %d", 6000000, c.RevisedLifetime)
		}
	})
}

func TestSerializeChannelSecurityToken(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		t.Parallel()
		c := NewChannelSecurityToken(1, 2, 1, 6000000)

		serialized, err := c.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize DiagnosticInfo: %s", err)
		}

		for i, s := range serialized {
			x := testChannelSecurityTokenBytes[0][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
}
