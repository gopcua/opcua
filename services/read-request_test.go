// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/wmnsk/gopcua/datatypes"
)

func TestNewReadRequest(t *testing.T) {
	r := NewReadRequest(
		time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC), 1033572, 0, 10000, "",
		0, TimestampsToReturnBoth,
		[]*datatypes.ReadValueID{
			datatypes.NewReadValueID(
				datatypes.NewFourByteNodeID(0, 2256),
				datatypes.IntegerIDValue,
				"", 0, "",
			),
		},
	)
	expected := &ReadRequest{
		TypeID: &datatypes.ExpandedNodeID{
			NodeID:       datatypes.NewFourByteNodeID(0, ServiceTypeReadRequest),
			NamespaceURI: datatypes.NewString(""),
		},
		RequestHeader: &RequestHeader{
			AuthenticationToken: datatypes.NewTwoByteNodeID(0x00),
			AuditEntryID:        datatypes.NewString(""),
			RequestHandle:       1033572,
			TimeoutHint:         10000,
			AdditionalHeader: &AdditionalHeader{
				TypeID: &datatypes.ExpandedNodeID{
					NodeID:       datatypes.NewTwoByteNodeID(0),
					NamespaceURI: datatypes.NewString(""),
				},
				EncodingMask: 0x00,
			},
			Timestamp: time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
		},
		MaxAge:             0,
		TimestampsToReturn: TimestampsToReturnBoth,
		NodesToRead: &datatypes.ReadValueIDArray{
			ArraySize: 1,
			ReadValueIDs: []*datatypes.ReadValueID{
				{
					NodeID:       datatypes.NewFourByteNodeID(0, 2256),
					AttributeID:  datatypes.IntegerIDValue,
					IndexRange:   datatypes.NewString(""),
					DataEncoding: datatypes.NewQualifiedName(0, ""),
				},
			},
		},
	}

	if diff := cmp.Diff(r, expected); diff != "" {
		t.Error(diff)
	}
}
func TestDecodeReadRequest(t *testing.T) {
	b := []byte{
		0x01, 0x00, 0x77, 0x02, 0x05, 0x00, 0x00, 0x10,
		0x00, 0x00, 0x00, 0x08, 0x22, 0x87, 0x62, 0xba,
		0x81, 0xe1, 0x11, 0xa6, 0x43, 0xf8, 0x77, 0x7b,
		0xc6, 0x2f, 0xc8, 0x00, 0x98, 0x67, 0xdd, 0xfd,
		0x30, 0xd4, 0x01, 0x64, 0xc5, 0x0f, 0x00, 0x00,
		0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x10,
		0x27, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00,
		0xd0, 0x08, 0x0d, 0x00, 0x00, 0x00, 0xff, 0xff,
		0xff, 0xff, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff,
	}
	r, err := DecodeReadRequest(b)
	if err != nil {
		t.Error(err)
	}
	expected := &ReadRequest{
		TypeID: &datatypes.ExpandedNodeID{
			NodeID: datatypes.NewFourByteNodeID(0, ServiceTypeReadRequest),
		},
		RequestHeader: &RequestHeader{
			AuthenticationToken: datatypes.NewOpaqueNodeID(0, []byte{
				0x08, 0x22, 0x87, 0x62, 0xba, 0x81, 0xe1, 0x11,
				0xa6, 0x43, 0xf8, 0x77, 0x7b, 0xc6, 0x2f, 0xc8,
			}),
			AuditEntryID:  datatypes.NewString(""),
			RequestHandle: 1033572,
			TimeoutHint:   10000,
			AdditionalHeader: &AdditionalHeader{
				TypeID: &datatypes.ExpandedNodeID{
					NodeID: datatypes.NewTwoByteNodeID(0),
				},
				EncodingMask: 0x00,
			},
			Timestamp: time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
			Payload: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
				0x01, 0x00, 0xd0, 0x08, 0x0d, 0x00, 0x00, 0x00,
				0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff,
				0xff, 0xff,
			},
		},
		MaxAge:             0,
		TimestampsToReturn: TimestampsToReturnBoth,
		NodesToRead: &datatypes.ReadValueIDArray{
			ArraySize: 1,
			ReadValueIDs: []*datatypes.ReadValueID{
				{
					NodeID:       datatypes.NewFourByteNodeID(0, 2256),
					AttributeID:  datatypes.IntegerIDValue,
					IndexRange:   datatypes.NewString(""),
					DataEncoding: datatypes.NewQualifiedName(0, ""),
				},
			},
		},
	}
	if diff := cmp.Diff(r, expected); diff != "" {
		t.Error(diff)
	}
}
func TestReadRequestDecodeFromBytes(t *testing.T) {
	b := []byte{
		0x01, 0x00, 0x77, 0x02, 0x05, 0x00, 0x00, 0x10,
		0x00, 0x00, 0x00, 0x08, 0x22, 0x87, 0x62, 0xba,
		0x81, 0xe1, 0x11, 0xa6, 0x43, 0xf8, 0x77, 0x7b,
		0xc6, 0x2f, 0xc8, 0x00, 0x98, 0x67, 0xdd, 0xfd,
		0x30, 0xd4, 0x01, 0x64, 0xc5, 0x0f, 0x00, 0x00,
		0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x10,
		0x27, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00,
		0xd0, 0x08, 0x0d, 0x00, 0x00, 0x00, 0xff, 0xff,
		0xff, 0xff, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff,
	}
	r := &ReadRequest{}
	if err := r.DecodeFromBytes(b); err != nil {
		t.Error(err)
	}
	expected := &ReadRequest{
		TypeID: &datatypes.ExpandedNodeID{
			NodeID: datatypes.NewFourByteNodeID(0, ServiceTypeReadRequest),
		},
		RequestHeader: &RequestHeader{
			AuthenticationToken: datatypes.NewOpaqueNodeID(0, []byte{
				0x08, 0x22, 0x87, 0x62, 0xba, 0x81, 0xe1, 0x11,
				0xa6, 0x43, 0xf8, 0x77, 0x7b, 0xc6, 0x2f, 0xc8,
			}),
			AuditEntryID:  datatypes.NewString(""),
			RequestHandle: 1033572,
			TimeoutHint:   10000,
			AdditionalHeader: &AdditionalHeader{
				TypeID: &datatypes.ExpandedNodeID{
					NodeID: datatypes.NewTwoByteNodeID(0),
				},
				EncodingMask: 0x00,
			},
			Timestamp: time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
			Payload: []byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
				0x01, 0x00, 0xd0, 0x08, 0x0d, 0x00, 0x00, 0x00,
				0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff,
				0xff, 0xff,
			},
		},
		MaxAge:             0,
		TimestampsToReturn: TimestampsToReturnBoth,
		NodesToRead: &datatypes.ReadValueIDArray{
			ArraySize: 1,
			ReadValueIDs: []*datatypes.ReadValueID{
				{
					NodeID:       datatypes.NewFourByteNodeID(0, 2256),
					AttributeID:  datatypes.IntegerIDValue,
					IndexRange:   datatypes.NewString(""),
					DataEncoding: datatypes.NewQualifiedName(0, ""),
				},
			},
		},
	}
	if diff := cmp.Diff(r, expected); diff != "" {
		t.Error(diff)
	}
}

func TestReadRequestSerialize(t *testing.T) {
	r := &ReadRequest{
		TypeID: &datatypes.ExpandedNodeID{
			NodeID: datatypes.NewFourByteNodeID(0, ServiceTypeReadRequest),
		},
		RequestHeader: &RequestHeader{
			AuthenticationToken: datatypes.NewOpaqueNodeID(0, []byte{
				0x08, 0x22, 0x87, 0x62, 0xba, 0x81, 0xe1, 0x11,
				0xa6, 0x43, 0xf8, 0x77, 0x7b, 0xc6, 0x2f, 0xc8,
			}),
			AuditEntryID:  datatypes.NewString(""),
			RequestHandle: 1033572,
			TimeoutHint:   10000,
			AdditionalHeader: &AdditionalHeader{
				TypeID: &datatypes.ExpandedNodeID{
					NodeID: datatypes.NewTwoByteNodeID(0),
				},
				EncodingMask: 0x00,
			},
			Timestamp: time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
		},
		MaxAge:             0,
		TimestampsToReturn: TimestampsToReturnBoth,
		NodesToRead: &datatypes.ReadValueIDArray{
			ArraySize: 1,
			ReadValueIDs: []*datatypes.ReadValueID{
				{
					NodeID:       datatypes.NewFourByteNodeID(0, 2256),
					AttributeID:  datatypes.IntegerIDValue,
					IndexRange:   datatypes.NewString(""),
					DataEncoding: datatypes.NewQualifiedName(0, ""),
				},
			},
		},
	}
	b, err := r.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte{
		0x01, 0x00, 0x77, 0x02, 0x05, 0x00, 0x00, 0x10,
		0x00, 0x00, 0x00, 0x08, 0x22, 0x87, 0x62, 0xba,
		0x81, 0xe1, 0x11, 0xa6, 0x43, 0xf8, 0x77, 0x7b,
		0xc6, 0x2f, 0xc8, 0x00, 0x98, 0x67, 0xdd, 0xfd,
		0x30, 0xd4, 0x01, 0x64, 0xc5, 0x0f, 0x00, 0x00,
		0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x10,
		0x27, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00,
		0xd0, 0x08, 0x0d, 0x00, 0x00, 0x00, 0xff, 0xff,
		0xff, 0xff, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff,
	}
	if diff := cmp.Diff(b, expected); diff != "" {
		t.Error(diff)
	}
}

func TestReadRequestSerializeTo(t *testing.T) {
	r := &ReadRequest{
		TypeID: &datatypes.ExpandedNodeID{
			NodeID: datatypes.NewFourByteNodeID(0, ServiceTypeReadRequest),
		},
		RequestHeader: &RequestHeader{
			AuthenticationToken: datatypes.NewOpaqueNodeID(0, []byte{
				0x08, 0x22, 0x87, 0x62, 0xba, 0x81, 0xe1, 0x11,
				0xa6, 0x43, 0xf8, 0x77, 0x7b, 0xc6, 0x2f, 0xc8,
			}),
			AuditEntryID:  datatypes.NewString(""),
			RequestHandle: 1033572,
			TimeoutHint:   10000,
			AdditionalHeader: &AdditionalHeader{
				TypeID: &datatypes.ExpandedNodeID{
					NodeID: datatypes.NewTwoByteNodeID(0),
				},
				EncodingMask: 0x00,
			},
			Timestamp: time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
		},
		MaxAge:             0,
		TimestampsToReturn: TimestampsToReturnBoth,
		NodesToRead: &datatypes.ReadValueIDArray{
			ArraySize: 1,
			ReadValueIDs: []*datatypes.ReadValueID{
				{
					NodeID:       datatypes.NewFourByteNodeID(0, 2256),
					AttributeID:  datatypes.IntegerIDValue,
					IndexRange:   datatypes.NewString(""),
					DataEncoding: datatypes.NewQualifiedName(0, ""),
				},
			},
		},
	}
	b := make([]byte, r.Len())
	if err := r.SerializeTo(b); err != nil {
		t.Fatal(err)
	}
	expected := []byte{
		0x01, 0x00, 0x77, 0x02, 0x05, 0x00, 0x00, 0x10,
		0x00, 0x00, 0x00, 0x08, 0x22, 0x87, 0x62, 0xba,
		0x81, 0xe1, 0x11, 0xa6, 0x43, 0xf8, 0x77, 0x7b,
		0xc6, 0x2f, 0xc8, 0x00, 0x98, 0x67, 0xdd, 0xfd,
		0x30, 0xd4, 0x01, 0x64, 0xc5, 0x0f, 0x00, 0x00,
		0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x10,
		0x27, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00,
		0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00,
		0xd0, 0x08, 0x0d, 0x00, 0x00, 0x00, 0xff, 0xff,
		0xff, 0xff, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff,
	}
	if diff := cmp.Diff(b, expected); diff != "" {
		t.Error(diff)
	}
}

func TestReadRequestLen(t *testing.T) {
	r := &ReadRequest{
		TypeID: &datatypes.ExpandedNodeID{
			NodeID: datatypes.NewFourByteNodeID(0, ServiceTypeReadRequest),
		},
		RequestHeader: &RequestHeader{
			AuthenticationToken: datatypes.NewOpaqueNodeID(0, []byte{
				0x08, 0x22, 0x87, 0x62, 0xba, 0x81, 0xe1, 0x11,
				0xa6, 0x43, 0xf8, 0x77, 0x7b, 0xc6, 0x2f, 0xc8,
			}),
			AuditEntryID:  datatypes.NewString(""),
			RequestHandle: 1033572,
			TimeoutHint:   10000,
			AdditionalHeader: &AdditionalHeader{
				TypeID: &datatypes.ExpandedNodeID{
					NodeID: datatypes.NewTwoByteNodeID(0),
				},
				EncodingMask: 0x00,
			},
			Timestamp: time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
		},
		MaxAge:             0,
		TimestampsToReturn: TimestampsToReturnBoth,
		NodesToRead: &datatypes.ReadValueIDArray{
			ArraySize: 1,
			ReadValueIDs: []*datatypes.ReadValueID{
				{
					NodeID:       datatypes.NewFourByteNodeID(0, 2256),
					AttributeID:  datatypes.IntegerIDValue,
					IndexRange:   datatypes.NewString(""),
					DataEncoding: datatypes.NewQualifiedName(0, ""),
				},
			},
		},
	}
	if r.Len() != 88 {
		t.Errorf("Len doesn't match. Want: %d, Got: %d", 88, r.Len())
	}
}

func TestReadRequestServiceType(t *testing.T) {
	r := &ReadRequest{}
	if r.ServiceType() != ServiceTypeReadRequest {
		t.Errorf(
			"ServiceType doesn't match. Want: %d, Got: %d",
			ServiceTypeActivateSessionRequest,
			r.ServiceType(),
		)
	}
}
