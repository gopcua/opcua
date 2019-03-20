// Copyright 2018-2019 opcua authors. All rights reserved.
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
				&DataValue{
					EncodingMask: 0x01,
					Value:        MustVariant(float32(2.50025)),
				},
				&DataValue{
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
			Struct: NewGUID("AAAABBBB-CCDD-EEFF-0101-0123456789AB"),
			Bytes: []byte{
				0xbb, 0xbb, 0xaa, 0xaa, 0xdd, 0xcc, 0xff, 0xee,
				0xab, 0x89, 0x67, 0x45, 0x23, 0x01, 0x01, 0x01,
			},
		},
	}
	RunCodecTest(t, cases)
}

func TestLocalizedText(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name:   "nothing",
			Struct: &LocalizedText{},
			Bytes:  []byte{0x00},
		},
		{
			Name: "has-locale",
			Struct: &LocalizedText{
				EncodingMask: LocalizedTextLocale,
				Locale:       "foo",
			},
			Bytes: []byte{
				0x01,
				0x03, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f,
			},
		},
		{
			Name: "has-text",
			Struct: &LocalizedText{
				EncodingMask: LocalizedTextText,
				Text:         "bar",
			},
			Bytes: []byte{
				0x02,
				0x03, 0x00, 0x00, 0x00, 0x62, 0x61, 0x72,
			},
		},
		{
			Name: "has-both",
			Struct: &LocalizedText{
				EncodingMask: LocalizedTextLocale | LocalizedTextText,
				Locale:       "foo",
				Text:         "bar",
			},
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
