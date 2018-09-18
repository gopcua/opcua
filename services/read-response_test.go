// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/id"
)

var readResponseTests = []struct {
	description string
	bytes       []byte
	r           *ReadResponse
	length      int
}{
	{
		description: "read response with single float value",
		bytes: []byte{
			0x01, 0x00, 0x7a, 0x02, 0x90, 0x18, 0xe3, 0x05,
			0x3f, 0x4f, 0xd4, 0x01, 0x01, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff,
			0xff, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
			0x01, 0x0a, 0x8e, 0x02, 0x20, 0x40, 0xff, 0xff,
			0xff, 0xff,
		},
		r: &ReadResponse{
			TypeID: &datatypes.ExpandedNodeID{
				NodeID: &datatypes.FourByteNodeID{
					EncodingMask: 0x01,
					Namespace:    0,
					Identifier:   id.ReadResponse_Encoding_DefaultBinary,
				},
			},
			ResponseHeader: &ResponseHeader{
				Timestamp:     time.Date(2018, time.September, 18, 11, 02, 00, 89000000, time.UTC),
				RequestHandle: 1,
				ServiceResult: 0x00000000,
				ServiceDiagnostics: &DiagnosticInfo{
					EncodingMask: 0x00,
				},
				StringTable: &datatypes.StringArray{
					ArraySize: -1,
				},
				AdditionalHeader: &AdditionalHeader{
					TypeID: &datatypes.ExpandedNodeID{
						NodeID: &datatypes.TwoByteNodeID{
							EncodingMask: 0x00,
							Identifier:   0,
						},
					},
					EncodingMask: 0x00,
				},
				Payload: []byte{
					0x01, 0x00, 0x00, 0x00, 0x01, 0x0a, 0x8e, 0x02,
					0x20, 0x40, 0xff, 0xff, 0xff, 0xff,
				},
			},
			Results: &datatypes.DataValueArray{
				ArraySize: 1,
				DataValues: []*datatypes.DataValue{
					{
						EncodingMask: 0x01,
						Value: datatypes.NewVariant(
							datatypes.NewFloat(2.5001559257507324),
						),
					},
				},
			},
			DiagnosticInfos: &DiagnosticInfoArray{
				ArraySize: -1,
			},
		},
		length: 42,
	},
}

func TestDecodeReadResponse(t *testing.T) {
	for _, test := range readResponseTests {
		t.Run(test.description, func(t *testing.T) {
			v, err := DecodeReadResponse(test.bytes)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(v, test.r); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestReadResponseSerialize(t *testing.T) {
	for _, test := range readResponseTests {
		t.Run(test.description, func(t *testing.T) {

			// need to clear Payload here.
			test.r.ResponseHeader.Payload = nil

			b, err := test.r.Serialize()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(b, test.bytes); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestReadResponseLen(t *testing.T) {
	for _, test := range readResponseTests {
		t.Run(test.description, func(t *testing.T) {
			if test.r.Len() != test.length {
				t.Errorf("Len doesn't match. Want: %d, Got: %d", test.length, test.r.Len())
			}
		})
	}
}

func TestReadResponseServiceType(t *testing.T) {
	for _, test := range readResponseTests {
		if test.r.ServiceType() != id.ReadResponse_Encoding_DefaultBinary {
			t.Errorf(
				"ServiceType doesn't match. Want: %d, Got: %d",
				id.ReadResponse_Encoding_DefaultBinary,
				test.r.ServiceType(),
			)
		}
	}
}
