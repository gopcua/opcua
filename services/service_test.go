// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"

	"github.com/wmnsk/gopcua/datatypes"
)

var testServiceBytes = [][]byte{
	{ // OpenSecureChannelRequest
		// TypeID
		0x01, 0x00, 0xbe, 0x01,
		// RequestHeader
		0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0xff, 0x03,
		0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00,
		// ClientProtocolVersion
		0x00, 0x00, 0x00, 0x00,
		// SecurityTokenRequestType
		0x00, 0x00, 0x00, 0x00,
		// MessageSecurityMode
		0x01, 0x00, 0x00, 0x00,
		// ClientNonce
		0xff, 0xff, 0xff, 0xff,
		// RequestedLifetime
		0x80, 0x8d, 0x5b, 0x00,
	},
	{ // OpenSecureChannelResponse
		// TypeID
		0x01, 0x00, 0xc1, 0x01,
		// ResponseHeader
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x02, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00,
		0x00, 0x66, 0x6f, 0x6f, 0x03, 0x00, 0x00, 0x00,
		0x62, 0x61, 0x72, 0x00, 0xff, 0x00,
		// ServerProtocolVersion
		0x00, 0x00, 0x00, 0x00,
		// SecurityToken
		0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x80, 0x8d, 0x5b, 0x00,
		// ServerNonce
		0x01, 0x00, 0x00, 0x00, 0xff,
	},
	{},
	{},
	{},
}

func TestDecodeServices(t *testing.T) {
	t.Run("open-sec-chan-req", func(t *testing.T) {
		t.Parallel()
		o, err := DecodeService(testServiceBytes[0])
		if err != nil {
			t.Fatalf("Failed to decode Service: %s", err)
		}

		osc, ok := o.(*OpenSecureChannelRequest)
		if !ok {
			t.Fatalf("Failed to assert type.")
		}

		switch {
		case o.ServiceType() != ServiceTypeOpenSecureChannelRequest:
			t.Errorf("ServiceType doesn't Match. Want: %d, Got: %d", ServiceTypeOpenSecureChannelRequest, o.ServiceType())
		case osc.ClientProtocolVersion != 0:
			t.Errorf("ClientProtocolVersion doesn't Match. Want: %d, Got: %d", 0, osc.ClientProtocolVersion)
		case osc.SecurityTokenRequestType != 0:
			t.Errorf("SecurityTokenRequestType doesn't Match. Want: %d, Got: %d", 0, osc.SecurityTokenRequestType)
		case osc.MessageSecurityMode != 1:
			t.Errorf("MessageSecurityMode doesn't Match. Want: %d, Got: %d", 1, osc.MessageSecurityMode)
		case osc.ClientNonce.Get() != nil:
			t.Errorf("ClientNonce doesn't Match. Want: %v, Got: %v", nil, osc.ClientNonce.Get())
		case osc.RequestedLifetime != 6000000:
			t.Errorf("RequestedLifetime doesn't Match. Want: %d, Got: %d", 6000000, osc.RequestedLifetime)
		}
		t.Log(o.String())
	})
	t.Run("open-sec-chan-res", func(t *testing.T) {
		t.Parallel()
		o, err := DecodeService(testServiceBytes[1])
		if err != nil {
			t.Fatalf("Failed to decode Service: %s", err)
		}

		osc, ok := o.(*OpenSecureChannelResponse)
		if !ok {
			t.Fatalf("Failed to assert type.")
		}

		switch {
		case o.ServiceType() != ServiceTypeOpenSecureChannelResponse:
			t.Errorf("ServiceType doesn't Match. Want: %d, Got: %d", ServiceTypeOpenSecureChannelResponse, o.ServiceType())
		case osc.ServerProtocolVersion != 0:
			t.Errorf("ServerProtocolVersion doesn't Match. Want: %d, Got: %d", 0, osc.ServerProtocolVersion)
		case osc.SecurityToken.ChannelID != 1:
			t.Errorf("SecurityToken.ChannelID doesn't Match. Want: %d, Got: %d", 1, osc.SecurityToken.ChannelID)
		case osc.SecurityToken.TokenID != 2:
			t.Errorf("SecurityToken.TokenID doesn't Match. Want: %d, Got: %d", 2, osc.SecurityToken.TokenID)
		case osc.SecurityToken.CreatedAt != 1:
			t.Errorf("SecurityToken.CreatedAt doesn't Match. Want: %d, Got: %d", 1, osc.SecurityToken.CreatedAt)
		case osc.SecurityToken.RevisedLifetime != 6000000:
			t.Errorf("SecurityToken.RevisedLifetime doesn't Match. Want: %d, Got: %d", 6000000, osc.SecurityToken.RevisedLifetime)
		case osc.ServerNonce.Get()[0] != 255:
			t.Errorf("ServerNonce doesn't Match. Want: %v, Got: %v", 255, osc.ServerNonce.Get()[0])
		}
		t.Log(o.String())
	})
}

func TestSerializeServices(t *testing.T) {
	t.Run("open-sec-chan-req", func(t *testing.T) {
		t.Parallel()
		o := NewOpenSecureChannelRequest(
			NewRequestHeader(
				datatypes.NewTwoByteNodeID(0),
				1,
				1,
				0x000003ff,
				0,
				"",
				NewAdditionalHeader(
					datatypes.NewExpandedNodeID(
						false, false,
						datatypes.NewTwoByteNodeID(0),
						"", 0,
					),
					0x00,
				),
				nil,
			),
			0, 0, 1, 6000000, nil,
		)

		serialized, err := o.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize Service: %s", err)
		}

		for i, s := range serialized {
			x := testServiceBytes[0][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("open-sec-chan-res", func(t *testing.T) {
		t.Parallel()
		o := NewOpenSecureChannelResponse(
			NewResponseHeader(
				1,
				1,
				0x00000000,
				datatypes.NewDiagnosticInfo(
					false, false, false, false, false, false, false,
					0, 0, 0, 0, nil, 0, nil,
				),
				[]string{"foo", "bar"},
				NewAdditionalHeader(
					datatypes.NewExpandedNodeID(
						false, false,
						datatypes.NewTwoByteNodeID(0xff),
						"", 0,
					),
					0x00,
				),
				nil,
			),
			0,
			NewChannelSecurityToken(
				1, 2, 1, 6000000,
			),
			[]byte{0xff},
		)

		serialized, err := o.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize Service: %s", err)
		}

		for i, s := range serialized {
			x := testServiceBytes[1][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", testServiceBytes[1])
		t.Logf("%x", serialized)
	})
}
