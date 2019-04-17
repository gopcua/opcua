// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"testing"
	"time"

	"github.com/gopcua/opcua/id"

	"github.com/gopcua/opcua/ua"
)

func TestMessage(t *testing.T) {
	cases := []CodecTestCase{
		{
			Name: "OPN",
			Struct: func() interface{} {
				m := NewMessage(
					&ua.OpenSecureChannelRequest{
						RequestHeader: &ua.RequestHeader{
							AuthenticationToken: ua.NewTwoByteNodeID(0),
							Timestamp:           time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
							RequestHandle:       1,
							ReturnDiagnostics:   0x03ff,
							AdditionalHeader:    ua.NewExtensionObject(nil),
						},
						ClientProtocolVersion: 0,
						RequestType:           ua.SecurityTokenRequestTypeIssue,
						SecurityMode:          ua.MessageSecurityModeNone,
						RequestedLifetime:     6000000,
					},
					id.OpenSecureChannelRequest_Encoding_DefaultBinary,
					&Config{
						SecureChannelID:   0,
						SecurityPolicyURI: "http://gopcua.example/OPCUA/SecurityPolicy#Foo",
						RequestID:         1,
						SequenceNumber:    1,
						SecurityTokenID:   0,
					},
				)

				// set message size manually, since it is computed in Encode
				// otherwise, the decode tests failed.
				m.Header.MessageSize = 131

				return m
			}(),
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
				// - AuthenticationToken
				0x00, 0x00,
				// - Timestamp
				0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30, 0xd4, 0x01,
				// - RequestHandle
				0x01, 0x00, 0x00, 0x00,
				// - ReturnDiagnostics
				0xff, 0x03, 0x00, 0x00,
				// - AuditEntry
				0xff, 0xff, 0xff, 0xff,
				// - TimeoutHint
				0x00, 0x00, 0x00, 0x00,
				// - AdditionalHeader
				//   - TypeID
				0x00, 0x00,
				//   - EncodingMask
				0x00,
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
			Struct: func() interface{} {
				m := NewMessage(
					&ua.GetEndpointsRequest{
						RequestHeader: &ua.RequestHeader{
							AuthenticationToken: ua.NewTwoByteNodeID(0),
							Timestamp:           time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
							RequestHandle:       1,
							ReturnDiagnostics:   0x03ff,
							AdditionalHeader:    ua.NewExtensionObject(nil),
						},
						EndpointURL: "opc.tcp://wow.its.easy:11111/UA/Server",
					},
					id.GetEndpointsRequest_Encoding_DefaultBinary,
					&Config{
						SecureChannelID:   0,
						SecurityPolicyURI: "http://gopcua.example/OPCUA/SecurityPolicy#Foo",
						RequestID:         1,
						SequenceNumber:    1,
						SecurityTokenID:   0,
					},
				)

				// set message size manually, since it is computed in Encode
				// otherwise, the decode tests failed.
				m.Header.MessageSize = 107

				return m
			}(),
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
				0xff, 0xff, 0xff, 0xff,
				// ProfileURIs
				0xff, 0xff, 0xff, 0xff,
			},
		}, {
			Name: "CLO",
			Struct: func() interface{} {
				m := NewMessage(
					&ua.CloseSecureChannelRequest{
						RequestHeader: &ua.RequestHeader{
							AuthenticationToken: ua.NewTwoByteNodeID(0),
							Timestamp:           time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
							RequestHandle:       1,
							ReturnDiagnostics:   0x03ff,
							AdditionalHeader:    ua.NewExtensionObject(nil),
						},
					},
					id.CloseSecureChannelRequest_Encoding_DefaultBinary,
					&Config{
						SecureChannelID:   0,
						SecurityPolicyURI: "http://gopcua.example/OPCUA/SecurityPolicy#Foo",
						RequestID:         1,
						SequenceNumber:    1,
						SecurityTokenID:   0,
					},
				)

				// set message size manually, since it is computed in Encode
				// otherwise, the decode tests failed.
				m.Header.MessageSize = 57

				return m
			}(),
			Bytes: []byte{ // OpenSecureChannelRequest
				// Message Header
				// MessageType: CLO
				0x43, 0x4c, 0x4f,
				// Chunk Type: Final
				0x46,
				// MessageSize: 57
				0x39, 0x00, 0x00, 0x00,
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
			},
		},
	}
	RunCodecTest(t, cases)
}
