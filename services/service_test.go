// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"
	"time"
)

var testServiceBytes = [][]byte{
	{ // OpenSecureChannelRequest
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
	{ // OpenSecureChannelResponse
		// TypeID
		0x01, 0x00, 0xc1, 0x01,
		// ResponseHeader
		0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30, 0xd4, 0x01,
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x02, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00,
		0x00, 0x66, 0x6f, 0x6f, 0x03, 0x00, 0x00, 0x00,
		0x62, 0x61, 0x72, 0x00, 0x00, 0x00,
		// ServerProtocolVersion
		0x00, 0x00, 0x00, 0x00,
		// SecurityToken
		0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x80, 0x8d, 0x5b, 0x00,
		// ServerNonce
		0x01, 0x00, 0x00, 0x00, 0xff,
	},
	{ // GetEndpointRequest
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
		// LocalIDs
		0xff, 0xff, 0xff, 0xff,
		// ProfileURIs
		0xff, 0xff, 0xff, 0xff,
	},
	{ // GetEndpointResponse
		// TypeID
		0x01, 0x00, 0xaf, 0x01,
		// ResponseHeader
		0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30, 0xd4, 0x01,
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Endpoints
		// ArraySize: 2
		0x02, 0x00, 0x00, 0x00,
		// EndpointURI
		0x06, 0x00, 0x00, 0x00, 0x65, 0x70, 0x2d, 0x75, 0x72, 0x6c,
		// Server (ApplicationDescription)
		// ApplicationURI
		0x07, 0x00, 0x00, 0x00, 0x61, 0x70, 0x70, 0x2d, 0x75, 0x72, 0x69,
		// ProductURI
		0x08, 0x00, 0x00, 0x00, 0x70, 0x72, 0x6f, 0x64, 0x2d, 0x75, 0x72, 0x69,
		// ApplicationName
		0x02, 0x08, 0x00, 0x00, 0x00, 0x61, 0x70, 0x70, 0x2d,
		0x6e, 0x61, 0x6d, 0x65,
		// ApplicationType
		0x00, 0x00, 0x00, 0x00,
		// GatewayServerURI
		0x06, 0x00, 0x00, 0x00, 0x67, 0x77, 0x2d, 0x75, 0x72, 0x69,
		// DiscoveryProfileURI
		0x08, 0x00, 0x00, 0x00, 0x70, 0x72, 0x6f, 0x64, 0x2d, 0x75, 0x72, 0x69,
		// DiscoveryURIs
		0x02, 0x00, 0x00, 0x00,
		0x0c, 0x00, 0x00, 0x00, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x2d, 0x75, 0x72, 0x69, 0x2d, 0x31,
		0x0c, 0x00, 0x00, 0x00, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x2d, 0x75, 0x72, 0x69, 0x2d, 0x32,
		// ServerCertificate
		0xff, 0xff, 0xff, 0xff,
		// MessageSecurityMode
		0x01, 0x00, 0x00, 0x00,
		// SecurityPolicyURI
		0x07, 0x00, 0x00, 0x00, 0x73, 0x65, 0x63, 0x2d, 0x75, 0x72, 0x69,
		// UserIdentityTokens
		// ArraySize
		0x02, 0x00, 0x00, 0x00,
		// PolicyID
		0x01, 0x00, 0x00, 0x00, 0x31,
		// TokenType
		0x00, 0x00, 0x00, 0x00,
		// IssuedTokenType
		0x0c, 0x00, 0x00, 0x00, 0x69, 0x73, 0x73, 0x75, 0x65, 0x64, 0x2d, 0x74, 0x6f, 0x6b, 0x65, 0x6e,
		// IssuerEndpointURI
		0x0a, 0x00, 0x00, 0x00, 0x69, 0x73, 0x73, 0x75, 0x65, 0x72, 0x2d, 0x75, 0x72, 0x69,
		// SecurityPolicyURI
		0x07, 0x00, 0x00, 0x00, 0x73, 0x65, 0x63, 0x2d, 0x75, 0x72, 0x69,
		// PolicyID
		0x01, 0x00, 0x00, 0x00, 0x31,
		// TokenType
		0x00, 0x00, 0x00, 0x00,
		// IssuedTokenType
		0x0c, 0x00, 0x00, 0x00, 0x69, 0x73, 0x73, 0x75, 0x65, 0x64, 0x2d, 0x74, 0x6f, 0x6b, 0x65, 0x6e,
		// IssuerEndpointURI
		0x0a, 0x00, 0x00, 0x00, 0x69, 0x73, 0x73, 0x75, 0x65, 0x72, 0x2d, 0x75, 0x72, 0x69,
		// SecurityPolicyURI
		0x07, 0x00, 0x00, 0x00, 0x73, 0x65, 0x63, 0x2d, 0x75, 0x72, 0x69,
		// TransportProfileURI
		0x09, 0x00, 0x00, 0x00, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x2d, 0x75, 0x72, 0x69,
		// SecurityLevel
		0x00,
		// EndpointURI
		0x06, 0x00, 0x00, 0x00, 0x65, 0x70, 0x2d, 0x75, 0x72, 0x6c,
		// Server (ApplicationDescription)
		// ApplicationURI
		0x07, 0x00, 0x00, 0x00, 0x61, 0x70, 0x70, 0x2d, 0x75, 0x72, 0x69,
		// ProductURI
		0x08, 0x00, 0x00, 0x00, 0x70, 0x72, 0x6f, 0x64, 0x2d, 0x75, 0x72, 0x69,
		// ApplicationName
		0x02, 0x08, 0x00, 0x00, 0x00, 0x61, 0x70, 0x70, 0x2d,
		0x6e, 0x61, 0x6d, 0x65,
		// ApplicationType
		0x00, 0x00, 0x00, 0x00,
		// GatewayServerURI
		0x06, 0x00, 0x00, 0x00, 0x67, 0x77, 0x2d, 0x75, 0x72, 0x69,
		// DiscoveryProfileURI
		0x08, 0x00, 0x00, 0x00, 0x70, 0x72, 0x6f, 0x64, 0x2d, 0x75, 0x72, 0x69,
		// DiscoveryURIs
		0x02, 0x00, 0x00, 0x00,
		0x0c, 0x00, 0x00, 0x00, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x2d, 0x75, 0x72, 0x69, 0x2d, 0x31,
		0x0c, 0x00, 0x00, 0x00, 0x64, 0x69, 0x73, 0x63, 0x6f, 0x76, 0x2d, 0x75, 0x72, 0x69, 0x2d, 0x32,
		// ServerCertificate
		0xff, 0xff, 0xff, 0xff,
		// MessageSecurityMode
		0x01, 0x00, 0x00, 0x00,
		// SecurityPolicyURI
		0x07, 0x00, 0x00, 0x00, 0x73, 0x65, 0x63, 0x2d, 0x75, 0x72, 0x69,
		// UserIdentityTokens
		// ArraySize
		0x02, 0x00, 0x00, 0x00,
		// PolicyID
		0x01, 0x00, 0x00, 0x00, 0x31,
		// TokenType
		0x00, 0x00, 0x00, 0x00,
		// IssuedTokenType
		0x0c, 0x00, 0x00, 0x00, 0x69, 0x73, 0x73, 0x75, 0x65, 0x64, 0x2d, 0x74, 0x6f, 0x6b, 0x65, 0x6e,
		// IssuerEndpointURI
		0x0a, 0x00, 0x00, 0x00, 0x69, 0x73, 0x73, 0x75, 0x65, 0x72, 0x2d, 0x75, 0x72, 0x69,
		// SecurityPolicyURI
		0x07, 0x00, 0x00, 0x00, 0x73, 0x65, 0x63, 0x2d, 0x75, 0x72, 0x69,
		// PolicyID
		0x01, 0x00, 0x00, 0x00, 0x31,
		// TokenType
		0x00, 0x00, 0x00, 0x00,
		// IssuedTokenType
		0x0c, 0x00, 0x00, 0x00, 0x69, 0x73, 0x73, 0x75, 0x65, 0x64, 0x2d, 0x74, 0x6f, 0x6b, 0x65, 0x6e,
		// IssuerEndpointURI
		0x0a, 0x00, 0x00, 0x00, 0x69, 0x73, 0x73, 0x75, 0x65, 0x72, 0x2d, 0x75, 0x72, 0x69,
		// SecurityPolicyURI
		0x07, 0x00, 0x00, 0x00, 0x73, 0x65, 0x63, 0x2d, 0x75, 0x72, 0x69,
		// TransportProfileURI
		0x09, 0x00, 0x00, 0x00, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x2d, 0x75, 0x72, 0x69,
		// SecurityLevel
		0x00,
	},
	{},
	{},
}

func TestDecode(t *testing.T) {
	t.Run("open-sec-chan-req", func(t *testing.T) {
		t.Parallel()
		o, err := Decode(testServiceBytes[0])
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
		o, err := Decode(testServiceBytes[1])
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
	t.Run("get-endpoint-req", func(t *testing.T) {
		t.Parallel()
		g, err := Decode(testServiceBytes[2])
		if err != nil {
			t.Fatalf("Failed to decode Service: %s", err)
		}

		gep, ok := g.(*GetEndpointRequest)
		if !ok {
			t.Fatalf("Failed to assert type.")
		}

		switch {
		case g.ServiceType() != ServiceTypeGetEndpointRequest:
			t.Errorf("ServiceType doesn't Match. Want: %d, Got: %d", ServiceTypeGetEndpointRequest, g.ServiceType())
		case gep.EndpointURL.Get() != "opc.tcp://wow.its.easy:11111/UA/Server":
			t.Errorf("EndpointURL doesn't Match. Want: %s, Got: %s", "opc.tcp://wow.its.easy:11111/UA/Server", gep.EndpointURL.Get())
		case gep.LocalIDs.ArraySize != -1:
			t.Errorf("LocalIDs.ArraySize doesn't Match. Want: %d, Got: %d", -1, gep.LocalIDs.ArraySize)
		case gep.ProfileURIs.ArraySize != -1:
			t.Errorf("ProfileURIs.ArraySize doesn't Match. Want: %d, Got: %d", -1, gep.ProfileURIs.ArraySize)
		}
		t.Log(g.String())
	})
	t.Run("get-endpoint-res", func(t *testing.T) {
		t.Parallel()
		g, err := Decode(testServiceBytes[3])
		if err != nil {
			t.Fatalf("Failed to decode Service: %s", err)
		}

		gep, ok := g.(*GetEndpointResponse)
		if !ok {
			t.Fatalf("Failed to assert type.")
		}

		if g.ServiceType() != ServiceTypeGetEndpointResponse {
			t.Errorf("ServiceType doesn't Match. Want: %d, Got: %d", ServiceTypeGetEndpointResponse, g.ServiceType())
		}

		for _, ep := range gep.Endpoints.EndpointDescriptions {
			switch {
			case ep.EndpointURL.Get() != "ep-url":
				t.Errorf("EndpointURL doesn't match. Want: %s, Got: %s", "ep-url", ep.EndpointURL.Get())
			case ep.ServerCertificate.Get() != nil:
				t.Errorf("ServerCertificate doesn't match. Want: %v, Got: %v", nil, ep.ServerCertificate.Get())
			case ep.MessageSecurityMode != SecModeNone:
				t.Errorf("MessageSecurityMode doesn't match. Want: %d, Got: %d", SecModeNone, ep.MessageSecurityMode)
			case ep.SecurityPolicyURI.Get() != "sec-uri":
				t.Errorf("SecurityPolicyURI doesn't match. Want: %s, Got: %s", "sec-uri", ep.SecurityPolicyURI.Get())
			case ep.TransportProfileURI.Get() != "trans-uri":
				t.Errorf("TransportProfileURI doesn't match. Want: %s, Got: %s", "trans-uri", ep.TransportProfileURI.Get())
			case ep.SecurityLevel != 0:
				t.Errorf("SecurityLevel doesn't match. Want: %d, Got: %d", 0, ep.SecurityLevel)
			}
			t.Log(ep.String())
		}

		t.Log(gep.String())
	})
}

func TestSerializeServices(t *testing.T) {
	t.Run("open-sec-chan-req", func(t *testing.T) {
		t.Parallel()
		o := NewOpenSecureChannelRequest(
			time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
			0, 1, 0, 0, "",
			0, ReqTypeIssue, SecModeNone, 6000000, nil,
		)
		o.SetDiagAll()

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
			time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
			1,
			0x00000000,
			NewNullDiagnosticInfo(),
			[]string{"foo", "bar"},
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
		t.Logf("%x", serialized)
	})
	t.Run("get-endpoint-req", func(t *testing.T) {
		t.Parallel()
		g := NewGetEndpointRequest(
			time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
			1, 0, 0, "",
			"opc.tcp://wow.its.easy:11111/UA/Server",
			nil, nil,
		)
		g.SetDiagAll()

		serialized, err := g.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize Service: %s", err)
		}

		for i, s := range serialized {
			x := testServiceBytes[2][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
	t.Run("get-endpoint-res", func(t *testing.T) {
		t.Parallel()
		g := NewGetEndpointResponse(
			time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
			1, 0x00000000,
			NewNullDiagnosticInfo(),
			[]string{},
			NewEndpointDesctiption(
				"ep-url",
				NewApplicationDescription(
					"app-uri", "prod-uri", "app-name", AppTypeServer,
					"gw-uri", "prof-uri", []string{"discov-uri-1", "discov-uri-2"},
				),
				[]byte{},
				SecModeNone,
				"sec-uri",
				NewUserTokenPolicyArray(
					[]*UserTokenPolicy{
						NewUserTokenPolicy(
							"1", UserTokenAnonymous,
							"issued-token", "issuer-uri", "sec-uri",
						),
						NewUserTokenPolicy(
							"1", UserTokenAnonymous,
							"issued-token", "issuer-uri", "sec-uri",
						),
					},
				),
				"trans-uri",
				0,
			),
			NewEndpointDesctiption(
				"ep-url",
				NewApplicationDescription(
					"app-uri", "prod-uri", "app-name", AppTypeServer,
					"gw-uri", "prof-uri", []string{"discov-uri-1", "discov-uri-2"},
				),
				[]byte{},
				SecModeNone,
				"sec-uri",
				NewUserTokenPolicyArray(
					[]*UserTokenPolicy{
						NewUserTokenPolicy(
							"1", UserTokenAnonymous,
							"issued-token", "issuer-uri", "sec-uri",
						),
						NewUserTokenPolicy(
							"1", UserTokenAnonymous,
							"issued-token", "issuer-uri", "sec-uri",
						),
					},
				),
				"trans-uri",
				0,
			),
		)

		serialized, err := g.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize Service: %s", err)
		}

		for i, s := range serialized {
			x := testServiceBytes[3][i]
			if s != x {
				t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
			}
		}
		t.Logf("%x", serialized)
	})
}
