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

var createSubscriptionRequestCases = []struct {
	description string
	structured  *CreateSubscriptionRequest
	serialized  []byte
}{
	{
		"normal",
		&CreateSubscriptionRequest{
			TypeID: &datatypes.ExpandedNodeID{
				NodeID: &datatypes.FourByteNodeID{
					EncodingMask: 0x01,
					Namespace:    0,
					Identifier:   id.CreateSubscriptionRequest_Encoding_DefaultBinary,
				},
			},
			RequestHeader: &RequestHeader{
				AuthenticationToken: datatypes.NewOpaqueNodeID(0, []byte{
					0xfe, 0x8d, 0x87, 0x79, 0xf7, 0x03, 0x27, 0x77,
					0xc5, 0x03, 0xa1, 0x09, 0x50, 0x29, 0x27, 0x60,
				}),
				AuditEntryID:  datatypes.NewString(""),
				RequestHandle: 1003429,
				TimeoutHint:   10000,
				AdditionalHeader: &AdditionalHeader{
					TypeID: &datatypes.ExpandedNodeID{
						NodeID: datatypes.NewTwoByteNodeID(0),
					},
					EncodingMask: 0x00,
				},
				Timestamp: time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
			},
			RequestedPublishingInterval: 500,
			RequestedLifetimeCount:      2400,
			RequestedMaxKeepAliveCount:  10,
			MaxNotificationsPerPublish:  65536,
			PublishingEnabled:           datatypes.NewBoolean(true),
			Priority:                    0,
		},
		[]byte{
			0x01, 0x00, 0x13, 0x03, 0x05, 0x00, 0x00, 0x10,
			0x00, 0x00, 0x00, 0xfe, 0x8d, 0x87, 0x79, 0xf7,
			0x03, 0x27, 0x77, 0xc5, 0x03, 0xa1, 0x09, 0x50,
			0x29, 0x27, 0x60, 0x00, 0x98, 0x67, 0xdd, 0xfd,
			0x30, 0xd4, 0x01, 0xa5, 0x4f, 0x0f, 0x00, 0x00,
			0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x10,
			0x27, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x40, 0x7f, 0x40, 0x60, 0x09,
			0x00, 0x00, 0x0a, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x01, 0x00, 0x01, 0x00,
		},
	},
}

func TestDecodeCreateSubscriptionRequest(t *testing.T) {
	for _, c := range createSubscriptionRequestCases {
		got, err := DecodeCreateSubscriptionRequest(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		// need to clear Payload here.
		got.Payload = nil

		if diff := cmp.Diff(got, c.structured); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeCreateSubscriptionRequest(t *testing.T) {
	for _, c := range createSubscriptionRequestCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestCreateSubscriptionRequestLen(t *testing.T) {
	for _, c := range createSubscriptionRequestCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestCreateSubscriptionRequestServiceType(t *testing.T) {
	for _, c := range createSubscriptionRequestCases {
		if c.structured.ServiceType() != id.CreateSubscriptionRequest_Encoding_DefaultBinary {
			t.Errorf(
				"ServiceType doesn't match. Want: %d, Got: %d",
				id.CreateSubscriptionRequest_Encoding_DefaultBinary,
				c.structured.ServiceType(),
			)
		}
	}
}
