// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"
	"time"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestWriteValue(t *testing.T) {
	cases := []codectest.Case{
		{
			Name: "normal",
			Struct: NewWriteValue(
				NewFourByteNodeID(0, 2256),
				IntegerIDValue,
				"",
				NewDataValue(
					true, false, true, false, true, false,
					NewVariant(NewFloat(2.50017)),
					0,
					time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
					0,
					time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
					0,
				),
			),
			Bytes: []byte{
				// NodeID
				0x01, 0x00, 0xd0, 0x08,
				// AttributeID
				0x0d, 0x00, 0x00, 0x00,
				// IndexRange
				0xff, 0xff, 0xff, 0xff,
				// Value
				0x0d, 0x0a, 0xc9, 0x02, 0x20, 0x40, 0x80, 0x3b,
				0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01, 0x80, 0x3b,
				0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
			},
		},
	}

	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeWriteValue(b)
	})
}

func TestWriteValueArray(t *testing.T) {
	cases := []codectest.Case{
		{
			Name: "normal",
			Struct: NewWriteValueArray(
				[]*WriteValue{
					NewWriteValue(
						NewFourByteNodeID(0, 2256),
						IntegerIDValue,
						"",
						NewDataValue(
							true, false, true, false, true, false,
							NewVariant(NewFloat(2.50017)),
							0,
							time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
							0,
							time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
							0,
						),
					),
					NewWriteValue(
						NewFourByteNodeID(0, 2256),
						IntegerIDValue,
						"",
						NewDataValue(
							true, false, true, false, true, false,
							NewVariant(NewFloat(2.50017)),
							0,
							time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
							0,
							time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
							0,
						),
					),
				},
			),
			Bytes: []byte{
				// ArraySize
				0x02, 0x00, 0x00, 0x00,
				// NodeID
				0x01, 0x00, 0xd0, 0x08,
				// AttributeID
				0x0d, 0x00, 0x00, 0x00,
				// IndexRange
				0xff, 0xff, 0xff, 0xff,
				// Value
				0x0d, 0x0a, 0xc9, 0x02, 0x20, 0x40, 0x80, 0x3b,
				0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01, 0x80, 0x3b,
				0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
				// NodeID
				0x01, 0x00, 0xd0, 0x08,
				// AttributeID
				0x0d, 0x00, 0x00, 0x00,
				// IndexRange
				0xff, 0xff, 0xff, 0xff,
				// Value
				0x0d, 0x0a, 0xc9, 0x02, 0x20, 0x40, 0x80, 0x3b,
				0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01, 0x80, 0x3b,
				0xe8, 0xb3, 0x92, 0x4e, 0xd4, 0x01,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeWriteValueArray(b)
	})
}
