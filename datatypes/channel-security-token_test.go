// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

var channelSecurityTokenCases = []struct {
	description string
	structured  *ChannelSecurityToken
	serialized  []byte
}{
	{
		"normal",
		NewChannelSecurityToken(
			1, 2, time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC), 6000000,
		),
		[]byte{
			// ChannelID
			0x01, 0x00, 0x00, 0x00,
			// TokenID
			0x02, 0x00, 0x00, 0x00,
			// CreatedAt
			0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30, 0xd4, 0x01,
			// RevisedLifetime
			0x80, 0x8d, 0x5b, 0x00,
		},
	},
}

func TestDecodeChannelSecurityToken(t *testing.T) {
	for _, c := range channelSecurityTokenCases {
		got, err := DecodeChannelSecurityToken(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeChannelSecurityToken(t *testing.T) {
	for _, c := range channelSecurityTokenCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestChannelSecurityTokenLen(t *testing.T) {
	for _, c := range channelSecurityTokenCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
