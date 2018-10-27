// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"reflect"
	"testing"
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/utils/codectest"

	"github.com/google/go-cmp/cmp"
	"github.com/wmnsk/gopcua/services"
)

func TestMessage(t *testing.T) {
	t.Skip("test fails in payload comparison")
	cases := []codectest.Case{
		{
			Name: "OPN",
			Struct: New(
				services.NewOpenSecureChannelRequest(
					services.NewRequestHeader(
						datatypes.NewTwoByteNodeID(0),
						time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
						1, 0x03ff, 0, "", services.NewNullAdditionalHeader(), nil,
					),
					0, services.ReqTypeIssue,
					services.SecModeNone, 6000000, nil,
				),
				&Config{
					SecureChannelID:   0,
					SecurityPolicyURI: "http://gopcua.example/OPCUA/SecurityPolicy#Foo",
					RequestID:         1,
					SequenceNumber:    1,
					SecurityTokenID:   0,
				},
			),
			Bytes: []byte{ // OpenSecureChannelRequest
				// Message Header
				// MessageType: OPN
				0x4f, 0x50, 0x4e,
				// Chunk Type: Final
				0x46,
				// MessageSize: 131
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
		}, {
			Name: "MSG",
			Struct: New(
				services.NewGetEndpointsRequest(
					services.NewRequestHeader(
						datatypes.NewTwoByteNodeID(0),
						time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
						1, 0x03ff, 0, "", services.NewNullAdditionalHeader(), nil,
					),
					"opc.tcp://wow.its.easy:11111/UA/Server",
					nil, nil,
				),
				&Config{
					SecureChannelID:   0,
					SecurityPolicyURI: "http://gopcua.example/OPCUA/SecurityPolicy#Foo",
					RequestID:         1,
					SequenceNumber:    1,
					SecurityTokenID:   0,
				},
			),
			Bytes: []byte{ // GetEndpointsRequest
				// Message Header
				// MessageType: MSG
				0x4d, 0x53, 0x47,
				// Chunk Type: Final
				0x46,
				// MessageSize: 107
				0x6b, 0x00, 0x00, 0x00,
				// SecureChannelID: 0
				0x00, 0x00, 0x00, 0x00,
				// SymmetricSecurityHeader
				// TokenID
				0x00, 0x00, 0x00, 0x00,
				// Sequence Header
				// SequenceNumber
				0x01, 0x00, 0x00, 0x00,
				// RequestID
				0x01, 0x00, 0x00, 0x00,
				// TypeID
				0x01, 0x00, 0xac, 0x01,
				// RequestHeader
				0x00, 0x00, 0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30,
				0xd4, 0x01, 0x01, 0x00, 0x00, 0x00, 0xff, 0x03,
				0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00,
				// ClientProtocolVersion
				0x26, 0x00, 0x00, 0x00, 0x6f, 0x70, 0x63, 0x2e,
				0x74, 0x63, 0x70, 0x3a, 0x2f, 0x2f, 0x77, 0x6f,
				0x77, 0x2e, 0x69, 0x74, 0x73, 0x2e, 0x65, 0x61,
				0x73, 0x79, 0x3a, 0x31, 0x31, 0x31, 0x31, 0x31,
				0x2f, 0x55, 0x41, 0x2f, 0x53, 0x65, 0x72, 0x76,
				0x65, 0x72,
				// LocaleIDs
				0x00, 0x00, 0x00, 0x00,
				// ProfileURIs
				0x00, 0x00, 0x00, 0x00,
			},
		}, {
			Name: "CLO",
			Struct: New(
				services.NewCloseSecureChannelRequest(
					services.NewRequestHeader(
						datatypes.NewTwoByteNodeID(0),
						time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
						1, 0x03ff, 0, "", services.NewNullAdditionalHeader(), nil,
					),
					1,
				),
				&Config{
					SecureChannelID:   0,
					SecurityPolicyURI: "http://gopcua.example/OPCUA/SecurityPolicy#Foo",
					RequestID:         1,
					SequenceNumber:    1,
					SecurityTokenID:   0,
				},
			),
			Bytes: []byte{ // OpenSecureChannelRequest
				// Message Header
				// MessageType: CLO
				0x43, 0x4c, 0x4f,
				// Chunk Type: Final
				0x46,
				// MessageSize: 61
				0x3d, 0x00, 0x00, 0x00,
				// SecureChannelID: 0
				0x00, 0x00, 0x00, 0x00,
				// SymmetricSecurityHeader
				// TokenID
				0x00, 0x00, 0x00, 0x00,
				// Sequence Header
				// SequenceNumber
				0x01, 0x00, 0x00, 0x00,
				// RequestID
				0x01, 0x00, 0x00, 0x00,
				// TypeID
				0x01, 0x00, 0xc4, 0x01,
				// RequestHeader
				0x00, 0x00, 0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30,
				0xd4, 0x01, 0x01, 0x00, 0x00, 0x00, 0xff, 0x03,
				0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00,
				// SecureChannelID
				0x01, 0x00, 0x00, 0x00,
			},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		v, err := Decode(b)
		if err != nil {
			return nil, err
		}
		// need to clear Payload in each headers, and the Service should be ignored.
		v.Header.Payload = nil
		if v.AsymmetricSecurityHeader != nil {
			v.AsymmetricSecurityHeader.Payload = nil
		}
		if v.SymmetricSecurityHeader != nil {
			v.SymmetricSecurityHeader.Payload = nil
		}
		v.SequenceHeader.Payload = nil
		return v, nil
	})
}

var msgCases = []struct {
	description string
	structured  *Message
	serialized  []byte
}{
	{
		"OPN",
		New(
			services.NewOpenSecureChannelRequest(
				services.NewRequestHeader(
					datatypes.NewTwoByteNodeID(0),
					time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
					1, 0x03ff, 0, "", services.NewNullAdditionalHeader(), nil,
				),
				0, services.ReqTypeIssue,
				services.SecModeNone, 6000000, nil,
			),
			&Config{
				SecureChannelID:   0,
				SecurityPolicyURI: "http://gopcua.example/OPCUA/SecurityPolicy#Foo",
				RequestID:         1,
				SequenceNumber:    1,
				SecurityTokenID:   0,
			},
		),
		[]byte{ // OpenSecureChannelRequest
			// Message Header
			// MessageType: OPN
			0x4f, 0x50, 0x4e,
			// Chunk Type: Final
			0x46,
			// MessageSize: 131
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
	}, {
		"MSG",
		New(
			services.NewGetEndpointsRequest(
				services.NewRequestHeader(
					datatypes.NewTwoByteNodeID(0),
					time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
					1, 0x03ff, 0, "", services.NewNullAdditionalHeader(), nil,
				),
				"opc.tcp://wow.its.easy:11111/UA/Server",
				nil, nil,
			),
			&Config{
				SecureChannelID:   0,
				SecurityPolicyURI: "http://gopcua.example/OPCUA/SecurityPolicy#Foo",
				RequestID:         1,
				SequenceNumber:    1,
				SecurityTokenID:   0,
			},
		),
		[]byte{ // GetEndpointsRequest
			// Message Header
			// MessageType: MSG
			0x4d, 0x53, 0x47,
			// Chunk Type: Final
			0x46,
			// MessageSize: 107
			0x6b, 0x00, 0x00, 0x00,
			// SecureChannelID: 0
			0x00, 0x00, 0x00, 0x00,
			// SymmetricSecurityHeader
			// TokenID
			0x00, 0x00, 0x00, 0x00,
			// Sequence Header
			// SequenceNumber
			0x01, 0x00, 0x00, 0x00,
			// RequestID
			0x01, 0x00, 0x00, 0x00,
			// TypeID
			0x01, 0x00, 0xac, 0x01,
			// RequestHeader
			0x00, 0x00, 0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30,
			0xd4, 0x01, 0x01, 0x00, 0x00, 0x00, 0xff, 0x03,
			0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00,
			// ClientProtocolVersion
			0x26, 0x00, 0x00, 0x00, 0x6f, 0x70, 0x63, 0x2e,
			0x74, 0x63, 0x70, 0x3a, 0x2f, 0x2f, 0x77, 0x6f,
			0x77, 0x2e, 0x69, 0x74, 0x73, 0x2e, 0x65, 0x61,
			0x73, 0x79, 0x3a, 0x31, 0x31, 0x31, 0x31, 0x31,
			0x2f, 0x55, 0x41, 0x2f, 0x53, 0x65, 0x72, 0x76,
			0x65, 0x72,
			// LocaleIDs
			0x00, 0x00, 0x00, 0x00,
			// ProfileURIs
			0x00, 0x00, 0x00, 0x00,
		},
	}, {
		"CLO",
		New(
			services.NewCloseSecureChannelRequest(
				services.NewRequestHeader(
					datatypes.NewTwoByteNodeID(0),
					time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
					1, 0x03ff, 0, "", services.NewNullAdditionalHeader(), nil,
				),
				1,
			),
			&Config{
				SecureChannelID:   0,
				SecurityPolicyURI: "http://gopcua.example/OPCUA/SecurityPolicy#Foo",
				RequestID:         1,
				SequenceNumber:    1,
				SecurityTokenID:   0,
			},
		),
		[]byte{ // OpenSecureChannelRequest
			// Message Header
			// MessageType: CLO
			0x43, 0x4c, 0x4f,
			// Chunk Type: Final
			0x46,
			// MessageSize: 61
			0x3d, 0x00, 0x00, 0x00,
			// SecureChannelID: 0
			0x00, 0x00, 0x00, 0x00,
			// SymmetricSecurityHeader
			// TokenID
			0x00, 0x00, 0x00, 0x00,
			// Sequence Header
			// SequenceNumber
			0x01, 0x00, 0x00, 0x00,
			// RequestID
			0x01, 0x00, 0x00, 0x00,
			// TypeID
			0x01, 0x00, 0xc4, 0x01,
			// RequestHeader
			0x00, 0x00, 0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30,
			0xd4, 0x01, 0x01, 0x00, 0x00, 0x00, 0xff, 0x03,
			0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00,
			// SecureChannelID
			0x01, 0x00, 0x00, 0x00,
		},
	},
}

func TestDecodeMessage(t *testing.T) {
	// option to regard []T{} and []T{nil} as equal
	// https://godoc.org/github.com/google/go-cmp/cmp#example-Option--EqualEmpty
	alwaysEqual := cmp.Comparer(func(_, _ interface{}) bool { return true })
	opt := cmp.FilterValues(func(x, y interface{}) bool {
		vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
		return (vx.IsValid() && vy.IsValid() && vx.Type() == vy.Type()) &&
			(vx.Kind() == reflect.Slice) && (vx.Len() == 0 && vy.Len() == 0)
	}, alwaysEqual)

	for _, c := range msgCases {
		got, err := Decode(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		// need to clear Payload in each headers, and the Service should be ignored.
		got.Header.Payload = nil
		if got.AsymmetricSecurityHeader != nil {
			got.AsymmetricSecurityHeader.Payload = nil
		}
		if got.SymmetricSecurityHeader != nil {
			got.SymmetricSecurityHeader.Payload = nil
		}
		got.SequenceHeader.Payload = nil

		if diff := cmp.Diff(got.Header, c.structured.Header, opt); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
		if diff := cmp.Diff(got.AsymmetricSecurityHeader, c.structured.AsymmetricSecurityHeader, opt); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
		if diff := cmp.Diff(got.SymmetricSecurityHeader, c.structured.SymmetricSecurityHeader, opt); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
		if diff := cmp.Diff(got.SequenceHeader, c.structured.SequenceHeader, opt); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeMessage(t *testing.T) {
	for _, c := range msgCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestMessageLen(t *testing.T) {
	for _, c := range msgCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
