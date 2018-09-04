package services

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/wmnsk/gopcua/datatypes"
)

func TestDecodeActivateSessionRequest(t *testing.T) {
	b := []byte{
		0x01, 0x00, 0xd3, 0x01, 0x02, 0x03, 0x00, 0xc7,
		0xd7, 0x68, 0xb5, 0x00, 0x98, 0x67, 0xdd, 0xfd,
		0x30, 0xd4, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
		0x41, 0x01, 0x01, 0x05, 0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x30, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff,
	}
	a, err := DecodeActivateSessionRequest(b)
	if err != nil {
		t.Error(err)
	}
	expected := &ActivateSessionRequest{
		TypeID: &datatypes.ExpandedNodeID{
			NodeID: datatypes.NewFourByteNodeID(0, ServiceTypeActivateSessionRequest),
		},
		RequestHeader: &RequestHeader{
			AuthenticationToken: datatypes.NewNumericNodeID(3, 3043547079),
			AuditEntryID:        datatypes.NewString(""),
			RequestHandle:       1,
			TimeoutHint:         0,
			AdditionalHeader: &AdditionalHeader{
				TypeID: &datatypes.ExpandedNodeID{
					NodeID: datatypes.NewTwoByteNodeID(0),
				},
				EncodingMask: 0x00,
			},
			Timestamp: time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
			Payload: []byte{
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x01, 0x00, 0x41, 0x01, 0x01, 0x05, 0x00, 0x00,
				0x00, 0x01, 0x00, 0x00, 0x00, 0x30, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
		ClientSignature:            NewSignatureData("", nil),
		ClientSoftwareCertificates: NewSignedSoftwareCertificateArray(nil),
		LocaleIDs:                  datatypes.NewStringArray(nil),
		UserIdentityToken: &datatypes.ExtensionObject{
			TypeID: &datatypes.ExpandedNodeID{
				NodeID: datatypes.NewFourByteNodeID(
					0,
					ServiceTypeAnonymousIdentityToken,
				),
			},
			Length:       5,
			EncodingMask: 0x01,
			Body:         datatypes.NewByteString([]byte("0")),
		},
		UserTokenSignature: NewSignatureData("", nil),
	}
	if diff := cmp.Diff(a, expected); diff != "" {
		t.Error(diff)
	}
}

func TestActivateSessionRequestDecodeFromBytes(t *testing.T) {
	b := []byte{
		0x01, 0x00, 0xd3, 0x01, 0x02, 0x03, 0x00, 0xc7,
		0xd7, 0x68, 0xb5, 0x00, 0x98, 0x67, 0xdd, 0xfd,
		0x30, 0xd4, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
		0x41, 0x01, 0x01, 0x05, 0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x30, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff,
	}
	a := &ActivateSessionRequest{}
	if err := a.DecodeFromBytes(b); err != nil {
		t.Error(err)
	}
	expected := &ActivateSessionRequest{
		TypeID: &datatypes.ExpandedNodeID{
			NodeID: datatypes.NewFourByteNodeID(0, ServiceTypeActivateSessionRequest),
		},
		RequestHeader: &RequestHeader{
			AuthenticationToken: datatypes.NewNumericNodeID(3, 3043547079),
			AuditEntryID:        datatypes.NewString(""),
			RequestHandle:       1,
			TimeoutHint:         0,
			AdditionalHeader: &AdditionalHeader{
				TypeID: &datatypes.ExpandedNodeID{
					NodeID: datatypes.NewTwoByteNodeID(0),
				},
				EncodingMask: 0x00,
			},
			Timestamp: time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
			Payload: []byte{
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x01, 0x00, 0x41, 0x01, 0x01, 0x05, 0x00, 0x00,
				0x00, 0x01, 0x00, 0x00, 0x00, 0x30, 0xff, 0xff,
				0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			},
		},
		ClientSignature:            NewSignatureData("", nil),
		ClientSoftwareCertificates: NewSignedSoftwareCertificateArray(nil),
		LocaleIDs:                  datatypes.NewStringArray(nil),
		UserIdentityToken: &datatypes.ExtensionObject{
			TypeID: &datatypes.ExpandedNodeID{
				NodeID: datatypes.NewFourByteNodeID(
					0,
					ServiceTypeAnonymousIdentityToken,
				),
			},
			Length:       5,
			EncodingMask: 0x01,
			Body:         datatypes.NewByteString([]byte("0")),
		},
		UserTokenSignature: NewSignatureData("", nil),
	}
	if diff := cmp.Diff(a, expected); diff != "" {
		t.Error(diff)
	}
}

func TestActivateSessionRequestSerialize(t *testing.T) {
	a := &ActivateSessionRequest{
		TypeID: &datatypes.ExpandedNodeID{
			NodeID: datatypes.NewFourByteNodeID(0, ServiceTypeActivateSessionRequest),
		},
		RequestHeader: &RequestHeader{
			AuthenticationToken: datatypes.NewNumericNodeID(3, 3043547079),
			AuditEntryID:        datatypes.NewString(""),
			RequestHandle:       1,
			TimeoutHint:         0,
			AdditionalHeader: &AdditionalHeader{
				TypeID: &datatypes.ExpandedNodeID{
					NodeID: datatypes.NewTwoByteNodeID(0),
				},
				EncodingMask: 0x00,
			},
			Timestamp: time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
		},
		ClientSignature:            NewSignatureData("", nil),
		ClientSoftwareCertificates: NewSignedSoftwareCertificateArray(nil),
		LocaleIDs:                  datatypes.NewStringArray(nil),
		UserIdentityToken: &datatypes.ExtensionObject{
			TypeID: &datatypes.ExpandedNodeID{
				NodeID: datatypes.NewFourByteNodeID(
					0,
					ServiceTypeAnonymousIdentityToken,
				),
			},
			Length:       5,
			EncodingMask: 0x01,
			Body:         datatypes.NewByteString([]byte("0")),
		},
		UserTokenSignature: NewSignatureData("", nil),
	}
	b, err := a.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte{
		0x01, 0x00, 0xd3, 0x01, 0x02, 0x03, 0x00, 0xc7,
		0xd7, 0x68, 0xb5, 0x00, 0x98, 0x67, 0xdd, 0xfd,
		0x30, 0xd4, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
		0x41, 0x01, 0x01, 0x05, 0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x30, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff,
	}
	if diff := cmp.Diff(b, expected); diff != "" {
		t.Error(diff)
	}
}

func TestActivateSessionRequestSerializeTo(t *testing.T) {
	a := &ActivateSessionRequest{
		TypeID: &datatypes.ExpandedNodeID{
			NodeID: datatypes.NewFourByteNodeID(0, ServiceTypeActivateSessionRequest),
		},
		RequestHeader: &RequestHeader{
			AuthenticationToken: datatypes.NewNumericNodeID(3, 3043547079),
			AuditEntryID:        datatypes.NewString(""),
			RequestHandle:       1,
			TimeoutHint:         0,
			AdditionalHeader: &AdditionalHeader{
				TypeID: &datatypes.ExpandedNodeID{
					NodeID: datatypes.NewTwoByteNodeID(0),
				},
				EncodingMask: 0x00,
			},
			Timestamp: time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
		},
		ClientSignature:            NewSignatureData("", nil),
		ClientSoftwareCertificates: NewSignedSoftwareCertificateArray(nil),
		LocaleIDs:                  datatypes.NewStringArray(nil),
		UserIdentityToken: &datatypes.ExtensionObject{
			TypeID: &datatypes.ExpandedNodeID{
				NodeID: datatypes.NewFourByteNodeID(
					0,
					ServiceTypeAnonymousIdentityToken,
				),
			},
			Length:       5,
			EncodingMask: 0x01,
			Body:         datatypes.NewByteString([]byte("0")),
		},
		UserTokenSignature: NewSignatureData("", nil),
	}
	b := make([]byte, a.Len())
	if err := a.SerializeTo(b); err != nil {
		t.Fatal(err)
	}
	expected := []byte{
		0x01, 0x00, 0xd3, 0x01, 0x02, 0x03, 0x00, 0xc7,
		0xd7, 0x68, 0xb5, 0x00, 0x98, 0x67, 0xdd, 0xfd,
		0x30, 0xd4, 0x01, 0x01, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
		0x41, 0x01, 0x01, 0x05, 0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x30, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff,
	}
	if diff := cmp.Diff(b, expected); diff != "" {
		t.Error(diff)
	}
}

func TestActivateSessionRequestLen(t *testing.T) {
	a := &ActivateSessionRequest{
		TypeID: &datatypes.ExpandedNodeID{
			NodeID: datatypes.NewFourByteNodeID(0, ServiceTypeActivateSessionRequest),
		},
		RequestHeader: &RequestHeader{
			AuthenticationToken: datatypes.NewNumericNodeID(3, 3043547079),
			AuditEntryID:        datatypes.NewString(""),
			RequestHandle:       1,
			TimeoutHint:         0,
			AdditionalHeader: &AdditionalHeader{
				TypeID: &datatypes.ExpandedNodeID{
					NodeID: datatypes.NewTwoByteNodeID(0),
				},
				EncodingMask: 0x00,
			},
			Timestamp: time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
		},
		ClientSignature:            NewSignatureData("", nil),
		ClientSoftwareCertificates: NewSignedSoftwareCertificateArray(nil),
		LocaleIDs:                  datatypes.NewStringArray(nil),
		UserIdentityToken: &datatypes.ExtensionObject{
			TypeID: &datatypes.ExpandedNodeID{
				NodeID: datatypes.NewFourByteNodeID(
					0,
					ServiceTypeAnonymousIdentityToken,
				),
			},
			Length:       5,
			EncodingMask: 0x01,
			Body:         datatypes.NewByteString([]byte("0")),
		},
		UserTokenSignature: NewSignatureData("", nil),
	}
	if a.Len() != 76 {
		t.Errorf("Len doesn't match. Want: %d, Got: %d", 76, a.Len())
	}
}

func TestActivateSessionRequestServiceType(t *testing.T) {
	a := &ActivateSessionRequest{}
	if a.ServiceType() != ServiceTypeActivateSessionRequest {
		t.Errorf(
			"ServiceType doesn't match. Want: %d, Got: %d",
			ServiceTypeActivateSessionRequest,
			a.ServiceType(),
		)
	}
}
