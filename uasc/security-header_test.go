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
		// SenderCertificateLength
		0x0f, 0x00, 0x00, 0x00,
		// SenderCertificate
		0x73, 0x6f, 0x6d, 0x65, 0x63, 0x65, 0x72, 0x74,
		0x69, 0x66, 0x69, 0x63, 0x61, 0x74, 0x65,
		// ReceiverCertificateThumbprintLength
		0x0e, 0x00, 0x00, 0x00,
		// ReceiverCertificateThumbprint
		0x73, 0x6f, 0x6d, 0x65, 0x74, 0x68, 0x75, 0x6d,
		0x62, 0x70, 0x72, 0x69, 0x6e, 0x74,
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

	dummyStr := hex.EncodeToString(a.Payload)
	switch {
	case a.SecurityPolicyURILength != 46:
		t.Errorf("SecurityPolicyURILength doesn't match. Want: %d, Got: %d", 46, a.SecurityPolicyURILength)
	case a.SecurityPolicyURIValue() != "http://gopcua.example/OPCUA/SecurityPolicy#Foo":
		t.Errorf("SecurityPolicyURI doesn't match. Want: %s, Got: %s", "http://gopcua.example/OPCUA/SecurityPolicy#Foo", a.SecurityPolicyURIValue())
	case a.SenderCertificateLength != 15:
		t.Errorf("SenderCertificateLength doesn't match. Want: %d, Got: %d", 15, a.SenderCertificateLength)
	case a.SenderCertificateValue() != "somecertificate":
		t.Errorf("SenderCertificate doesn't match. Want: %s, Got: %s", "somecertificate", a.SenderCertificateValue())
	case a.ReceiverCertificateThumbprintLength != 14:
		t.Errorf("ReceiverCertificateThumbprintLength doesn't match. Want: %d, Got: %d", 14, a.ReceiverCertificateThumbprintLength)
	case a.ReceiverCertificateThumbprintValue() != "somethumbprint":
		t.Errorf("ReceiverCertificateThumbprint doesn't match. Want: %s, Got: %s", "somethumbprint", a.ReceiverCertificateThumbprintValue())
	case dummyStr != "deadbeef":
		t.Errorf("Payload doesn't match. Want: %s, Got: %s", "deadbeef", dummyStr)
	}
}

func TestSerializeAsymmetricSecurityHeader(t *testing.T) {
	a := NewAsymmetricSecurityHeader(
		"http://gopcua.example/OPCUA/SecurityPolicy#Foo",
		"somecertificate",
		"somethumbprint",
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
