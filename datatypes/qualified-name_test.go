package datatypes

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// test string "foobar"
var foobar = []byte{0x01, 0x00, 0x06, 0x00, 0x00, 0x00, 0x66, 0x6f, 0x6f, 0x62, 0x61, 0x72}

func TestNewQualifiedName(t *testing.T) {
	q := NewQualifiedName(1, "foo")
	expected := &QualifiedName{
		NamespaceIndex: 1,
		Name: &String{
			Value:  []byte("foo"),
			Length: 3,
		},
	}
	if diff := cmp.Diff(q, expected); diff != "" {
		t.Error(diff)
	}
}

func TestNewQualifiedNameEmptyName(t *testing.T) {
	q := NewQualifiedName(1, "")
	expected := &QualifiedName{
		NamespaceIndex: 1,
		Name: &String{
			Length: -1,
		},
	}
	if diff := cmp.Diff(q, expected); diff != "" {
		t.Error(diff)
	}
}

func TestDecodeQualifiedName(t *testing.T) {
	q, err := DecodeQualifiedName(foobar)
	if err != nil {
		t.Fatal(err)
	}
	expected := &QualifiedName{
		NamespaceIndex: 1,
		Name: &String{
			Value:  []byte("foobar"),
			Length: 6,
		},
	}
	if diff := cmp.Diff(q, expected); diff != "" {
		t.Error(diff)
	}
}

func TestQualifiedNameDecodeFromBytes(t *testing.T) {
	q := &QualifiedName{}
	if err := q.DecodeFromBytes(foobar); err != nil {
		t.Fatal(err)
	}
	expected := &QualifiedName{
		NamespaceIndex: 1,
		Name: &String{
			Value:  []byte("foobar"),
			Length: 6,
		},
	}
	if diff := cmp.Diff(q, expected); diff != "" {
		t.Error(diff)
	}
}

func TestQualifiedNameSerialize(t *testing.T) {
	q := &QualifiedName{
		NamespaceIndex: 1,
		Name: &String{
			Value:  []byte("foobar"),
			Length: 6,
		},
	}
	b, err := q.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(b, foobar); diff != "" {
		t.Error(diff)
	}
}

func TestQualifiedNameSerializeTo(t *testing.T) {
	q := &QualifiedName{
		NamespaceIndex: 1,
		Name: &String{
			Value:  []byte("foobar"),
			Length: 6,
		},
	}
	b := make([]byte, q.Len())
	if err := q.SerializeTo(b); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(b, foobar); diff != "" {
		t.Error(diff)
	}
}

func TestQualifiedNameLen(t *testing.T) {
	q := &QualifiedName{
		NamespaceIndex: 1,
		Name: &String{
			Value:  []byte("foobar"),
			Length: 6,
		},
	}
	if q.Len() != 12 {
		t.Errorf("Len doesn't match. Want: %d, Got: %d", 12, q.Len())
	}
}
