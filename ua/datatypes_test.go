// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"testing"
	"time"
)

func TestDataValue(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name: "value only",
			Struct: &DataValue{
				EncodingMask: 0x01,
				Value:        MustVariant(float32(2.50025)),
			},
			Bytes: []byte{
				// EncodingMask
				0x01,
				// Value
				0x0a,                   // type
				0x19, 0x04, 0x20, 0x40, // value
			},
		},
		{
			Name: "value, source timestamp, server timestamp",
			Struct: &DataValue{
				EncodingMask:    0x0d,
				Value:           MustVariant(float32(2.50017)),
				SourceTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
				ServerTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
			},
			Bytes: []byte{
				// EncodingMask
				0x0d,
				// Value
				0x0a,                   // type
				0xc9, 0x02, 0x20, 0x40, // value
				// SourceTimestamp
				0x80, 0x3b, 0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
				// SeverTimestamp
				0x80, 0x3b, 0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
			},
		},
	}
	RunCodecTest(t, cases)
}

func TestDataValueArray(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name: "value only and value, source timestamp, server timestamp",
			Struct: []*DataValue{
				{
					EncodingMask: 0x01,
					Value:        MustVariant(float32(2.50025)),
				},
				{
					EncodingMask:    0x0d,
					Value:           MustVariant(float32(2.50017)),
					SourceTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
					ServerTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
				},
			},
			Bytes: []byte{
				// length
				0x02, 0x00, 0x00, 0x00,

				// EncodingMask
				0x01,
				// Value
				0x0a,
				0x19, 0x04, 0x20, 0x40,

				// EncodingMask
				0x0d,
				// Value
				0x0a,
				0xc9, 0x02, 0x20, 0x40,
				// SourceTimestamp
				0x80, 0x3b, 0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
				// ServerTimestamp
				0x80, 0x3b, 0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
			},
		},
	}
	RunCodecTest(t, cases)
}

func TestGUID(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name:   "ok",
			Struct: NewGUID("AAAABBBB-CCDD-EEFF-0102-0123456789AB"),
			Bytes: []byte{
				// data1 (inverse order)
				0xbb, 0xbb, 0xaa, 0xaa,
				// data2 (inverse order)
				0xdd, 0xcc,
				// data3 (inverse order)
				0xff, 0xee,
				// data4 (same order)
				0x01, 0x02, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab,
			},
		},
		{
			Name:   "spec",
			Struct: NewGUID("72962B91-FA75-4AE6-8D28-B404DC7DAF63"),
			Bytes: []byte{
				// data1 (inverse order)
				0x91, 0x2b, 0x96, 0x72,
				// data2 (inverse order)
				0x75, 0xfa,
				// data3 (inverse order)
				0xe6, 0x4a,
				// data4 (same order)
				0x8d, 0x28, 0xb4, 0x04, 0xdc, 0x7d, 0xaf, 0x63,
			},
		},
	}
	RunCodecTest(t, cases)
}

func TestLocalizedText(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name:   "nothing",
			Struct: NewLocalizedText(""),
			Bytes:  []byte{0x00},
		},
		{
			Name:   "has-locale",
			Struct: NewLocalizedTextWithLocale("", "foo"),
			Bytes: []byte{
				0x01,
				0x03, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
			},
		},
		{
			Name:   "has-text",
			Struct: NewLocalizedText("bar"),
			Bytes: []byte{
				0x02,
				0x03, 0x00, 0x00, 0x00, 0x62, 0x61, 0x72,
			},
		},
		{
			Name:   "has-both",
			Struct: NewLocalizedTextWithLocale("bar", "foo"),
			Bytes: []byte{
				0x03,
				0x03, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
				// second String: "bar"
				0x03, 0x00, 0x00, 0x00, 0x62, 0x61, 0x72,
			},
		},
	}
	RunCodecTest(t, cases)
}
