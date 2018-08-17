// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package utils

import (
	"testing"
	"time"
)

var testTimestampBytes = [][]byte{
	{ // 2018-08-10 23:00:00 +0000 UTC, 1533942000000000000
		0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30, 0xd4, 0x01,
	},
}

func TestDecodeTime(t *testing.T) {
	ts := DecodeTimestamp(testTimestampBytes[0])

	if ts.UnixNano() != 1533942000000000000 {
		t.Errorf("Timestamp doesn't match. Want: %d, Got: %d", 1533942000000000000, ts.UnixNano())
	}
}

func TestEncodeTime(t *testing.T) {
	ts := time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC)

	serialized := make([]byte, 8)
	EncodeTimestamp(serialized, ts)

	for i, s := range serialized {
		x := testTimestampBytes[0][i]
		if s != x {
			t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
		}
	}
	t.Logf("%x", serialized)
}
