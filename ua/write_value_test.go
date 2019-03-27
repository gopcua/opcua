// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"testing"
	"time"
)

func TestWriteValue(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name: "normal",
			Struct: &WriteValue{
				NodeID:      NewFourByteNodeID(0, 2256),
				AttributeID: AttributeIDValue,
				Value: &DataValue{
					EncodingMask:    DataValueValue | DataValueSourceTimestamp | DataValueServerTimestamp,
					Value:           MustVariant(float32(2.50017)),
					SourceTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
					ServerTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
				},
			},
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

	RunCodecTest(t, cases)
}

func TestWriteValueArray(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name: "normal",
			Struct: []*WriteValue{
				&WriteValue{
					NodeID:      NewFourByteNodeID(0, 2256),
					AttributeID: AttributeIDValue,
					Value: &DataValue{
						EncodingMask:    DataValueValue | DataValueSourceTimestamp | DataValueServerTimestamp,
						Value:           MustVariant(float32(2.50017)),
						SourceTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
						ServerTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
					},
				},
				&WriteValue{
					NodeID:      NewFourByteNodeID(0, 2256),
					AttributeID: AttributeIDValue,
					Value: &DataValue{
						EncodingMask:    DataValueValue | DataValueSourceTimestamp | DataValueServerTimestamp,
						Value:           MustVariant(float32(2.50017)),
						SourceTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
						ServerTimestamp: time.Date(2018, time.September, 17, 14, 28, 29, 112000000, time.UTC),
					},
				},
			},
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
	RunCodecTest(t, cases)
}
