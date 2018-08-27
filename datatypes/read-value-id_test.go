package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDecodeReadValueID(t *testing.T) {
	// sample qualified name from wireshark
	b := []byte{
		0x01, 0x00, 0xd0, 0x08, 0x0d, 0x00, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff,
		0xff, 0xff,
	}

	r, err := DecodeReadValueID(b)
	if err != nil {
		t.Error(err)
	}
	expected := &ReadValueID{
		NodeID:       NewFourByteNodeID(0, 2256),
		AttributeID:  IntegerIDValue,
		IndexRange:   NewString(""),
		DataEncoding: NewQualifiedName(0, ""),
	}
	if diff := cmp.Diff(r, expected); diff != "" {
		t.Error(diff)
	}
}

func TestReadValueIDDecodeFromBytes(t *testing.T) {
	r := &ReadValueID{}
	// sample qualified name from wireshark
	b := []byte{
		0x01, 0x00, 0xd0, 0x08, 0x0d, 0x00, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff,
		0xff, 0xff,
	}
	if err := r.DecodeFromBytes(b); err != nil {
		t.Error(err)
	}
	expected := &ReadValueID{
		NodeID:       NewFourByteNodeID(0, 2256),
		AttributeID:  IntegerIDValue,
		IndexRange:   NewString(""),
		DataEncoding: NewQualifiedName(0, ""),
	}
	if diff := cmp.Diff(r, expected); diff != "" {
		t.Error(diff)
	}
}

func TestReadValueIDSerialize(t *testing.T) {
	expected := []byte{
		0x01, 0x00, 0xd0, 0x08, 0x0d, 0x00, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff,
		0xff, 0xff,
	}
	r := &ReadValueID{
		NodeID:       NewFourByteNodeID(0, 2256),
		AttributeID:  IntegerIDValue,
		IndexRange:   NewString(""),
		DataEncoding: NewQualifiedName(0, ""),
	}
	b, err := r.Serialize()
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(b, expected); diff != "" {
		t.Error(diff)
	}
}

func TestReadValueIDSerializeTo(t *testing.T) {
	expected := []byte{
		0x01, 0x00, 0xd0, 0x08, 0x0d, 0x00, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0xff, 0xff,
		0xff, 0xff,
	}
	r := &ReadValueID{
		NodeID:       NewFourByteNodeID(0, 2256),
		AttributeID:  IntegerIDValue,
		IndexRange:   NewString(""),
		DataEncoding: NewQualifiedName(0, ""),
	}
	b := make([]byte, r.Len())
	if err := r.SerializeTo(b); err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(b, expected); diff != "" {
		t.Error(diff)
	}
}

func TestReadValueIDLen(t *testing.T) {
	r := &ReadValueID{
		NodeID:       NewFourByteNodeID(0, 2256),
		AttributeID:  IntegerIDValue,
		IndexRange:   NewString(""),
		DataEncoding: NewQualifiedName(0, ""),
	}
	if r.Len() != 18 {
		t.Errorf("Len doesn't match. Want: %d, Got: %d", 18, r.Len())
	}
}

func TestNewReadValueIDArray(t *testing.T) {
	ids := []*ReadValueID{
		&ReadValueID{
			NodeID:       NewFourByteNodeID(0, 2256),
			AttributeID:  IntegerIDValue,
			IndexRange:   NewString(""),
			DataEncoding: NewQualifiedName(0, ""),
		},
		&ReadValueID{
			NodeID:       NewNumericNodeID(0, 2256),
			AttributeID:  IntegerIDDataType,
			IndexRange:   NewString("5:7"),
			DataEncoding: NewQualifiedName(1, "something"),
		},
	}
	arr := NewReadValueIDArray(ids)
	expected := &ReadValueIDArray{
		ArraySize:    2,
		ReadValueIDs: ids,
	}
	if diff := cmp.Diff(arr, expected); diff != "" {
		t.Error(diff)
	}
}

func TestNewReadValueIDArrayNil(t *testing.T) {
	arr := NewReadValueIDArray(nil)
	expected := &ReadValueIDArray{
		ArraySize: 0,
	}
	if diff := cmp.Diff(arr, expected); diff != "" {
		t.Error(diff)
	}
}

func TestDecodeReadValueIDArray(t *testing.T) {
	in := []byte{
		0x03, 0x00, 0x00, 0x00, 0x03, 0x01, 0x00, 0x0b,
		0x00, 0x00, 0x00, 0x54, 0x65, 0x6d, 0x70, 0x65,
		0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x02, 0x00,
		0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff, 0x03, 0x01, 0x00, 0x0b,
		0x00, 0x00, 0x00, 0x54, 0x65, 0x6d, 0x70, 0x65,
		0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x03, 0x00,
		0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff, 0x03, 0x01, 0x00, 0x0b,
		0x00, 0x00, 0x00, 0x54, 0x65, 0x6d, 0x70, 0x65,
		0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x04, 0x00,
		0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff,
	}
	arr, err := DecodeReadValueIDArray(in)
	if err != nil {
		t.Error(err)
	}
	expected := &ReadValueIDArray{
		ArraySize: 3,
		ReadValueIDs: []*ReadValueID{
			{
				NodeID:       NewStringNodeID(1, "Temperature"),
				AttributeID:  IntegerIDNodeClass,
				IndexRange:   NewString(""),
				DataEncoding: NewQualifiedName(0, ""),
			},
			{
				NodeID:       NewStringNodeID(1, "Temperature"),
				AttributeID:  IntegerIDBrowseName,
				IndexRange:   NewString(""),
				DataEncoding: NewQualifiedName(0, ""),
			},
			{
				NodeID:       NewStringNodeID(1, "Temperature"),
				AttributeID:  IntegerIDDisplayName,
				IndexRange:   NewString(""),
				DataEncoding: NewQualifiedName(0, ""),
			},
		},
	}
	if diff := cmp.Diff(arr, expected); diff != "" {
		t.Error(diff)
	}
}

func TestReadValueIDArrayDecodeFromBytes(t *testing.T) {
	arr := &ReadValueIDArray{}
	in := []byte{
		0x03, 0x00, 0x00, 0x00, 0x03, 0x01, 0x00, 0x0b,
		0x00, 0x00, 0x00, 0x54, 0x65, 0x6d, 0x70, 0x65,
		0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x02, 0x00,
		0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff, 0x03, 0x01, 0x00, 0x0b,
		0x00, 0x00, 0x00, 0x54, 0x65, 0x6d, 0x70, 0x65,
		0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x03, 0x00,
		0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff, 0x03, 0x01, 0x00, 0x0b,
		0x00, 0x00, 0x00, 0x54, 0x65, 0x6d, 0x70, 0x65,
		0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x04, 0x00,
		0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff,
	}
	if err := arr.DecodeFromBytes(in); err != nil {
		t.Error(err)
	}
	expected := &ReadValueIDArray{
		ArraySize: 3,
		ReadValueIDs: []*ReadValueID{
			{
				NodeID:       NewStringNodeID(1, "Temperature"),
				AttributeID:  IntegerIDNodeClass,
				IndexRange:   NewString(""),
				DataEncoding: NewQualifiedName(0, ""),
			},
			{
				NodeID:       NewStringNodeID(1, "Temperature"),
				AttributeID:  IntegerIDBrowseName,
				IndexRange:   NewString(""),
				DataEncoding: NewQualifiedName(0, ""),
			},
			{
				NodeID:       NewStringNodeID(1, "Temperature"),
				AttributeID:  IntegerIDDisplayName,
				IndexRange:   NewString(""),
				DataEncoding: NewQualifiedName(0, ""),
			},
		},
	}
	if diff := cmp.Diff(arr, expected); diff != "" {
		t.Error(diff)
	}
}

func TestReadValueIDArraySerialize(t *testing.T) {
	expected := []byte{
		0x03, 0x00, 0x00, 0x00, 0x03, 0x01, 0x00, 0x0b,
		0x00, 0x00, 0x00, 0x54, 0x65, 0x6d, 0x70, 0x65,
		0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x02, 0x00,
		0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff, 0x03, 0x01, 0x00, 0x0b,
		0x00, 0x00, 0x00, 0x54, 0x65, 0x6d, 0x70, 0x65,
		0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x03, 0x00,
		0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff, 0x03, 0x01, 0x00, 0x0b,
		0x00, 0x00, 0x00, 0x54, 0x65, 0x6d, 0x70, 0x65,
		0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x04, 0x00,
		0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff,
	}
	arr := &ReadValueIDArray{
		ArraySize: 3,
		ReadValueIDs: []*ReadValueID{
			{
				NodeID:       NewStringNodeID(1, "Temperature"),
				AttributeID:  IntegerIDNodeClass,
				IndexRange:   NewString(""),
				DataEncoding: NewQualifiedName(0, ""),
			},
			{
				NodeID:       NewStringNodeID(1, "Temperature"),
				AttributeID:  IntegerIDBrowseName,
				IndexRange:   NewString(""),
				DataEncoding: NewQualifiedName(0, ""),
			},
			{
				NodeID:       NewStringNodeID(1, "Temperature"),
				AttributeID:  IntegerIDDisplayName,
				IndexRange:   NewString(""),
				DataEncoding: NewQualifiedName(0, ""),
			},
		},
	}
	b, err := arr.Serialize()
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(b, expected); diff != "" {
		t.Error(diff)
	}
}

func TestReadValueIDArraySerializeTo(t *testing.T) {
	expected := []byte{
		0x03, 0x00, 0x00, 0x00, 0x03, 0x01, 0x00, 0x0b,
		0x00, 0x00, 0x00, 0x54, 0x65, 0x6d, 0x70, 0x65,
		0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x02, 0x00,
		0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff, 0x03, 0x01, 0x00, 0x0b,
		0x00, 0x00, 0x00, 0x54, 0x65, 0x6d, 0x70, 0x65,
		0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x03, 0x00,
		0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff, 0x03, 0x01, 0x00, 0x0b,
		0x00, 0x00, 0x00, 0x54, 0x65, 0x6d, 0x70, 0x65,
		0x72, 0x61, 0x74, 0x75, 0x72, 0x65, 0x04, 0x00,
		0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00,
		0xff, 0xff, 0xff, 0xff,
	}
	arr := &ReadValueIDArray{
		ArraySize: 3,
		ReadValueIDs: []*ReadValueID{
			{
				NodeID:       NewStringNodeID(1, "Temperature"),
				AttributeID:  IntegerIDNodeClass,
				IndexRange:   NewString(""),
				DataEncoding: NewQualifiedName(0, ""),
			},
			{
				NodeID:       NewStringNodeID(1, "Temperature"),
				AttributeID:  IntegerIDBrowseName,
				IndexRange:   NewString(""),
				DataEncoding: NewQualifiedName(0, ""),
			},
			{
				NodeID:       NewStringNodeID(1, "Temperature"),
				AttributeID:  IntegerIDDisplayName,
				IndexRange:   NewString(""),
				DataEncoding: NewQualifiedName(0, ""),
			},
		},
	}
	b := make([]byte, arr.Len())
	if err := arr.SerializeTo(b); err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(b, expected); diff != "" {
		t.Error(diff)
	}
}

func TestReadValueIDArrayLen(t *testing.T) {
	arr := &ReadValueIDArray{
		ArraySize: 3,
		ReadValueIDs: []*ReadValueID{
			{
				NodeID:       NewStringNodeID(1, "Temperature"),
				AttributeID:  IntegerIDNodeClass,
				IndexRange:   NewString(""),
				DataEncoding: NewQualifiedName(0, ""),
			},
			{
				NodeID:       NewStringNodeID(1, "Temperature"),
				AttributeID:  IntegerIDBrowseName,
				IndexRange:   NewString(""),
				DataEncoding: NewQualifiedName(0, ""),
			},
			{
				NodeID:       NewStringNodeID(1, "Temperature"),
				AttributeID:  IntegerIDDisplayName,
				IndexRange:   NewString(""),
				DataEncoding: NewQualifiedName(0, ""),
			},
		},
	}
	if arr.Len() != 100 {
		t.Errorf("Len doesn't match. Want: %d, Got: %d", 100, arr.Len())
	}
}
