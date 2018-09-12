// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestDecodeChannelSecurityToken(t *testing.T) {
	var cases = []struct {
		input []byte
		want  *ChannelSecurityToken
	}{
		{
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
			NewChannelSecurityToken(
				1, 2, time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC), 6000000,
			),
		},
	}

	for i, c := range cases {
		got, err := DecodeChannelSecurityToken(c.input)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}

func TestSerializeChannelSecurityToken(t *testing.T) {
	var cases = []struct {
		input *ChannelSecurityToken
		want  []byte
	}{
		{
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

	for i, c := range cases {
		got, err := c.input.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}

func TestChannelSecurityTokenLen(t *testing.T) {
	var cases = []struct {
		input *ChannelSecurityToken
		want  int
	}{
		{
			NewChannelSecurityToken(
				1, 2, time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC), 6000000,
			),
			20,
		},
	}

	for i, c := range cases {
		got := c.input.Len()
		if diff := cmp.Diff(got, c.want); diff != "" {
			t.Errorf("case #%d failed\n%s", i, diff)
		}
	}
}
