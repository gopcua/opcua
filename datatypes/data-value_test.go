// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"
	"time"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestDataValue(t *testing.T) {
	cases := []codectest.Case{
		{
			Name: "value only",
			Struct: &DataValue{
				EncodingMask: 0x01,
				Value:        NewVariant(NewFloat(2.50025)),
			},
			Bytes: []byte{0x01, 0x0a, 0x19, 0x04, 0x20, 0x40},
		},
		{
			Name: "value, source timestamp, server timestamp",
			Struct: &DataValue{
				EncodingMask:    0x0d,
				Value:           NewVariant(NewFloat(2.50017)),
				SourceTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
				ServerTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
			},
			Bytes: []byte{
				0x0d, 0x0a, 0xc9, 0x02, 0x20, 0x40, 0x80, 0x3b,
				0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01, 0x80, 0x3b,
				0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeDataValue(b)
	})
}

func TestDataValueArray(t *testing.T) {
	cases := []codectest.Case{
		{
			Name: "value only and value, source timestamp, server timestamp",
			Struct: NewDataValueArray([]*DataValue{
				&DataValue{
					EncodingMask: 0x01,
					Value:        NewVariant(NewFloat(2.50025)),
				},
				&DataValue{
					EncodingMask:    0x0d,
					Value:           NewVariant(NewFloat(2.50017)),
					SourceTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
					ServerTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
				},
			}),
			Bytes: []byte{
				0x02, 0x00, 0x00, 0x00, 0x01, 0x0a, 0x19, 0x04,
				0x20, 0x40, 0x0d, 0x0a, 0xc9, 0x02, 0x20, 0x40, 0x80, 0x3b,
				0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01, 0x80, 0x3b,
				0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeDataValueArray(b)
	})
}
