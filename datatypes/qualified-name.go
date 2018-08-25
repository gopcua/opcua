package datatypes

import "encoding/binary"

// QualifiedName contains a qualified name. It is, for example, used as BrowseName.
// The name part of the QualifiedName is restricted to 512 characters.
//
// Specification: Part 3, 8.3
type QualifiedName struct {
	NamespaceIndex uint16
	Name           *String
}

// NewQualifiedName creates a new QualifiedName.
func NewQualifiedName(index uint16, name string) *QualifiedName {
	value := []byte(name)
	length := -1

	if len(value) != 0 {
		length = len(value)
	}

	q := &QualifiedName{
		NamespaceIndex: index,
		Name: &String{
			Value:  value,
			Length: int32(length),
		},
	}
	return q
}

// DecodeQualifiedName decodes given bytes into QualifiedName.
func DecodeQualifiedName(b []byte) (*QualifiedName, error) {
	q := &QualifiedName{}
	if err := q.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return q, nil
}

// DecodeFromBytes decodes given bytes into OPC UA QualifiedName.
func (q *QualifiedName) DecodeFromBytes(b []byte) error {
	q.NamespaceIndex = binary.LittleEndian.Uint16(b[:2])
	q.Name = &String{}
	return q.Name.DecodeFromBytes(b[2:])
}

// Serialize serializes QualifiedName into bytes.
func (q *QualifiedName) Serialize() ([]byte, error) {
	b := make([]byte, q.Len())
	if err := q.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes QualifiedName into bytes.
func (q *QualifiedName) SerializeTo(b []byte) error {
	binary.LittleEndian.PutUint16(b[:2], q.NamespaceIndex)
	return q.Name.SerializeTo(b[2:])
}

// Len returns the actual length of QualifiedName in int.
func (q *QualifiedName) Len() int {
	return 2 + q.Name.Len()
}
