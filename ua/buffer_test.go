// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"encoding/binary"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// OPC 10000-6 §5.2.2.5 defines the range of a date/time as 1601-01-01 to
// 9999-12-31 11:59:59PM, carried in 100 nanosecond intervals as a signed
// Int64. A tick count outside that range saturates instead of wrapping
// through the conversion arithmetic: on decode, a value at or below 0 - a
// time before 1601-01-01 - decodes as time.Time{}, the value tick 0
// decodes to, which gopcua returns as its null DateTime (see the IsZero
// callers), and a value naming a time after 9999-12-31T23:59:59Z,
// including the maximum Int64, decodes as that instant. Encoding clamps
// symmetrically: a time at or before 1601-01-01 encodes as 0 and a time at
// or after 9999-12-31T23:59:59Z as the maximum Int64.
func TestDateTimeOutOfRangeSaturates(t *testing.T) {
	// 9999-12-31T23:59:59Z in 100 nanosecond ticks since 1601-01-01.
	const maxTicks = int64(2650467743990000000)
	earliest := time.Time{}
	latest := time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC)

	t.Run("decode", func(t *testing.T) {
		cases := []struct {
			name  string
			ticks uint64
			want  time.Time
		}{
			{"tick 0", 0, earliest},
			{"MaxUint64, i.e. -1 as an Int64", math.MaxUint64, earliest},
			{"the Int64 minimum", 1 << 63, earliest},
			{"one tick into the range", 1, time.Date(1601, time.January, 1, 0, 0, 0, 100, time.UTC)},
			{"the latest tick in the range", uint64(maxTicks), latest},
			{"one tick past the range", uint64(maxTicks) + 1, latest},
			{"the Int64 maximum", math.MaxInt64, latest},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				d := make([]byte, 8)
				binary.LittleEndian.PutUint64(d, c.ticks)
				got := NewBuffer(d).ReadTime()
				require.Equal(t, c.want, got)
			})
		}
	})

	t.Run("encode", func(t *testing.T) {
		cases := []struct {
			name string
			v    time.Time
			want uint64
		}{
			{"the zero time", time.Time{}, 0},
			{"a time far before 1601", time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC), 0},
			{"a time before 1601", time.Date(1600, time.December, 31, 23, 59, 59, 0, time.UTC), 0},
			{"1601-01-01 itself", time.Date(1601, time.January, 1, 0, 0, 0, 0, time.UTC), 0},
			{"one tick into the range", time.Date(1601, time.January, 1, 0, 0, 0, 100, time.UTC), 1},
			{"one tick before the end of the range", latest.Add(-100 * time.Nanosecond), uint64(maxTicks) - 1},
			{"9999-12-31T23:59:59Z", latest, math.MaxInt64},
			{"one tick past the end of the range", latest.Add(100 * time.Nanosecond), math.MaxInt64},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				b := NewBuffer(nil)
				b.WriteTime(c.v)
				require.NoError(t, b.Error())
				require.Equal(t, c.want, binary.LittleEndian.Uint64(b.Bytes()))
			})
		}
	})
}
