// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"encoding/hex"
	"testing"
)

var testAsymmetricSecurityHeaderBytes = [][]byte{
	{
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
		0x02, 0x00, 0x00, 0x00, 0xde, 0xad,
		// ReceiverCertificateThumbprint
		0x02, 0x00, 0x00, 0x00, 0xbe, 0xef,
		// dummy Payload
		0xde, 0xad, 0xbe, 0xef,
	},
	{},
	{},
}

func TestDecodeAsymmetricSecurityHeader(t *testing.T) {
	a, err := DecodeAsymmetricSecurityHeader(testAsymmetricSecurityHeaderBytes[0])
	if err != nil {
		t.Fatalf("Failed to decode AsymmetricSecurityHeader: %s", err)
	}

	cert := hex.EncodeToString(a.SenderCertificate.Get())
	thumb := hex.EncodeToString(a.ReceiverCertificateThumbprint.Get())
	dummyStr := hex.EncodeToString(a.Payload)
	switch {
	case a.SecurityPolicyURI.Length != 46:
		t.Errorf("SecurityPolicyURILength doesn't match. Want: %d, Got: %d", 46, a.SecurityPolicyURI.Length)
	case a.SecurityPolicyURI.Get() != "http://gopcua.example/OPCUA/SecurityPolicy#Foo":
		t.Errorf("SecurityPolicyURI doesn't match. Want: %s, Got: %s", "http://gopcua.example/OPCUA/SecurityPolicy#Foo", a.SecurityPolicyURI.Get())
	case a.SenderCertificate.Length != 2:
		t.Errorf("SenderCertificateLength doesn't match. Want: %d, Got: %d", 2, a.SenderCertificate.Length)
	case cert != "dead":
		t.Errorf("SenderCertificate doesn't match. Want: %s, Got: %s", "dead", cert)
	case a.ReceiverCertificateThumbprint.Length != 2:
		t.Errorf("ReceiverCertificateThumbprintLength doesn't match. Want: %d, Got: %d", 2, a.ReceiverCertificateThumbprint.Length)
	case thumb != "beef":
		t.Errorf("ReceiverCertificateThumbprint doesn't match. Want: %s, Got: %s", "beef", thumb)
	case dummyStr != "deadbeef":
		t.Errorf("Payload doesn't match. Want: %s, Got: %s", "deadbeef", dummyStr)
	}
	t.Log(a.String())
}

func TestSerializeAsymmetricSecurityHeader(t *testing.T) {
	a := NewAsymmetricSecurityHeader(
		"http://gopcua.example/OPCUA/SecurityPolicy#Foo",
		[]byte{0xde, 0xad},
		[]byte{0xbe, 0xef},
		[]byte{0xde, 0xad, 0xbe, 0xef},
	)

	serialized, err := a.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize Header: %s", err)
	}

	for i, s := range serialized {
		x := testAsymmetricSecurityHeaderBytes[0][i]
		if s != x {
			t.Errorf("Bytes doesn't match. Want: %#x, Got: %#x at %dth", x, s, i)
		}
	}
	t.Logf("%x", serialized)
}
