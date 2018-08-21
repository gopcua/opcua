// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"testing"
	"time"

	"github.com/wmnsk/gopcua/services"
)

var testUASCBytes = [][]byte{
	{ // OpenSecureChannelRequest
		// Message Header
		// MessageType: MSG
		0x4f, 0x50, 0x4e,
		// Chunk Type: Final
		0x46,
		// MessageSize: 16
		0x83, 0x00, 0x00, 0x00,
		// SecureChannelID: 0
		0x00, 0x00, 0x00, 0x00,
		// AsymmetricSecurityHeader
		// SecurityPolicyURILength
		0x2e, 0x00, 0x00, 0x00,
		// SecurityPolicyURI
		0x68, 0x74, 0x74, 0x70, 0x3a, 0x2f, 0x2f, 0x67,
		0x6f, 0x70, 0x63, 0x75, 0x61, 0x2e, 0x65, 0x78,
		0x61, 0x6d, 0x70, 0x6c, 0x65, 0x2f, 0x4f, 0x50,
		0x43, 0x55, 0x41, 0x2f, 0x53, 0x65, 0x63, 0x75,
		0x72, 0x69, 0x74, 0x79, 0x50, 0x6f, 0x6c, 0x69,
		0x63, 0x79, 0x23, 0x46, 0x6f, 0x6f,
		// SenderCertificate
		0xff, 0xff, 0xff, 0xff,
		// ReceiverCertificateThumbprint
		0xff, 0xff, 0xff, 0xff,
		// Sequence Header
		// SequenceNumber
		0x01, 0x00, 0x00, 0x00,
		// RequestID
		0x01, 0x00, 0x00, 0x00,
		// TypeID
		0x01, 0x00, 0xbe, 0x01,
		// RequestHeader
		0x00, 0x00, 0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30,
		0xd4, 0x01, 0x01, 0x00, 0x00, 0x00, 0xff, 0x03,
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
	{},
	{},
	{},
	{},
}

func TestDecodeMessage(t *testing.T) {
	t.Run("open", func(t *testing.T) {
		t.Parallel()
		o, err := Decode(testUASCBytes[0])
		if err != nil {
			t.Fatalf("Failed to decode UASC Message: %s", err)
		}

		osc, ok := o.Service.(*services.OpenSecureChannelRequest)
		if !ok {
			t.Errorf("Failed to assert type: %T", o)
		}

		t.Log(osc.String())
	})
}

func TestSerializeMessage(t *testing.T) {
	t.Run("open", func(t *testing.T) {
		t.Parallel()

		opn := services.NewOpenSecureChannelRequest(
			time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
			0, 1, 0, 0, "", 0, services.ReqTypeIssue,
			services.SecModeNone, 6000000, nil,
		)
		opn.SetDiagAll()

		o := New(
			opn,
			&Config{
				SecureChannelID:   0,
				SecurityPolicyURI: "http://gopcua.example/OPCUA/SecurityPolicy#Foo",
				RequestID:         1,
				SequenceNumber:    1,
				SecurityTokenID:   0,
			},
		)

		serialized, err := o.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize Service: %s", err)
		}

		for i, s := range serialized {
			x := testUASCBytes[0][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
}
