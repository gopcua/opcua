// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestReadValueID(t *testing.T) {
	cases := []codectest.Case{
		{
			Name: "Normal",
			Struct: NewReadValueID(
				NewFourByteNodeID(0, 2256),
				IntegerIDValue,
				"", 0, "",
			),
			Bytes: []byte{
				// NodeID
				0x01,
				0x00,
				0xd0, 0x08,
				// AttributeID
				0x0d, 0x00, 0x00, 0x00,
				// Index Range
				0xff, 0xff, 0xff, 0xff,
				// qualified name
				0x00, 0x00,
				0xff, 0xff, 0xff, 0xff,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeReadValueID(b)
	})
}

func TestReadValueIDArray(t *testing.T) {
	cases := []codectest.Case{
		{
			Name: "Normal",
			Struct: NewReadValueIDArray(
				[]*ReadValueID{
					{
						NodeID:       NewStringNodeID(1, "Temperature"),
						AttributeID:  IntegerIDNodeClass,
						IndexRange:   NewString(""),
						DataEncoding: NewQualifiedName(0, ""),
					},
					{
						NodeID:       NewStringNodeID(1, "Temperature"),
						AttributeID:  IntegerIDBrowseName,
						IndexRange:   NewString(""),
						DataEncoding: NewQualifiedName(0, ""),
					},
					{
						NodeID:       NewStringNodeID(1, "Temperature"),
						AttributeID:  IntegerIDDisplayName,
						IndexRange:   NewString(""),
						DataEncoding: NewQualifiedName(0, ""),
					},
				}),
			Bytes: []byte{
				// Length
				0x03, 0x00, 0x00, 0x00,

				// NodeID
				0x03,
				0x01, 0x00,
				0x0b, 0x00, 0x00, 0x00,
				0x54, 0x65, 0x6d, 0x70, 0x65, 0x72, 0x61, 0x74, 0x75, 0x72, 0x65,
				// AttributeID
				0x02, 0x00, 0x00, 0x00,
				// IndexRange
				0xff, 0xff, 0xff, 0xff,
				// QualifiedName
				0x00, 0x00,
				0xff, 0xff, 0xff, 0xff,

				// NodeID
				0x03,
				0x01, 0x00,
				0x0b, 0x00, 0x00, 0x00,
				0x54, 0x65, 0x6d, 0x70, 0x65, 0x72, 0x61, 0x74, 0x75, 0x72, 0x65,
				// AttributeID
				0x03, 0x00, 0x00, 0x00,
				// IndexRange
				0xff, 0xff, 0xff, 0xff,
				// QualifiedName
				0x00, 0x00,
				0xff, 0xff, 0xff, 0xff,

				// NodeID
				0x03,
				0x01, 0x00,
				0x0b, 0x00, 0x00, 0x00,
				0x54, 0x65, 0x6d, 0x70, 0x65, 0x72, 0x61, 0x74, 0x75, 0x72, 0x65,
				// AttributeID
				0x04, 0x00, 0x00, 0x00,
				// IndexRange
				0xff, 0xff, 0xff, 0xff,
				// QualifiedName
				0x00, 0x00,
				0xff, 0xff, 0xff, 0xff,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeReadValueIDArray(b)
	})
}
