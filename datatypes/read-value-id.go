package datatypes

import "encoding/binary"

// IntegerID is a UInt32 that is used as an identifier, such as a handle.
// All values, except for 0, are valid.
//
// Specification: Part 4, 7.14
type IntegerID uint32

// Identifiers assigned to Attributes.
//
// Specification: Part 6, A.1
const (
	IntegerIDNodeID IntegerID = iota + 1
	IntegerIDNodeClass
	IntegerIDBrowseName
	IntegerIDDisplayName
	IntegerIDDescription
	IntegerIDWriteMask
	IntegerIDUserWriteMask
	IntegerIDIsAbstract
	IntegerIDSymmetric
	IntegerIDInverseName
	IntegerIDContainsNoLoops
	IntegerIDEventNotifier
	IntegerIDValue
	IntegerIDDataType
	IntegerIDValueRank
	IntegerIDArrayDimensions
	IntegerIDAccessLevel
	IntegerIDUserAccessLevel
	IntegerIDMinimumSamplingInterval
	IntegerIDHistorizing
	IntegerIDExecutable
	IntegerIDUserExecutable
	IntegerIDDataTypeDefinition
	IntegerIDRolePermissions
	IntegerIDUserRolePermissions
	IntegerIDAccessRestrictions
	IntegerIDAccessLevelEx
)

// ReadValueID is an identifier for an item to read or to monitor.
//
// Specification: Part 4, 7.24
type ReadValueID struct {
	NodeID
	AttributeID  IntegerID
	IndexRange   *String
	DataEncoding *QualifiedName
}

// DecodeReadValueID decodes given bytes into ReadValueID.
func DecodeReadValueID(b []byte) (*ReadValueID, error) {
	r := &ReadValueID{}
	if err := r.DecodeFromBytes(b); err != nil {
		return nil, err
	}
	return r, nil
}

// DecodeFromBytes decodes given bytes into OPC UA ReadValueID.
func (r *ReadValueID) DecodeFromBytes(b []byte) error {
	nodeID, err := DecodeNodeID(b)
	if err != nil {
		return err
	}
	r.NodeID = nodeID
	offset := r.NodeID.Len()

	// attribute id
	r.AttributeID = IntegerID(binary.LittleEndian.Uint32(b[offset:]))
	offset += 4

	// index range
	r.IndexRange = &String{}
	if err := r.IndexRange.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += r.IndexRange.Len()

	// data encoding
	r.DataEncoding = &QualifiedName{}
	if err := r.DataEncoding.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	return nil
}

// Serialize serializes ReadValueID into bytes.
func (r *ReadValueID) Serialize() ([]byte, error) {
	b := make([]byte, r.Len())
	if err := r.SerializeTo(b); err != nil {
		return nil, err
	}
	return b, nil
}

// SerializeTo serializes ReadValueID into bytes.
func (r *ReadValueID) SerializeTo(b []byte) error {
	offset := 0

	// node id
	if r.NodeID != nil {
		if err := r.NodeID.SerializeTo(b); err != nil {
			return err
		}
		offset += r.NodeID.Len()
	}

	// attribute id
	binary.LittleEndian.PutUint32(b[offset:offset+4], uint32(r.AttributeID))
	offset += 4

	// index range
	if r.IndexRange != nil {
		if err := r.IndexRange.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += r.IndexRange.Len()
	}

	// data encoding
	if r.DataEncoding != nil {
		return r.DataEncoding.SerializeTo(b[offset:])
	}

	return nil
}

// Len returns the actual length of ReadValueID in int.
func (r *ReadValueID) Len() int {
	// attribute id
	length := 4

	if r.NodeID != nil {
		length += r.NodeID.Len()
	}

	if r.IndexRange != nil {
		length += r.IndexRange.Len()
	}

	if r.DataEncoding != nil {
		length += r.DataEncoding.Len()
	}

	return length
}

// ReadValueIDArray represents an array of ReadValueIDs.
// It does not correspond to a certain type from the specification
// but makes encoding and decoding easier.
type ReadValueIDArray struct {
	ArraySize    int32
	ReadValueIDs []*ReadValueID
}

// NewReadValueIDArray creates a new ReadValueIDArray from multiple ReadValueIDs.
func NewReadValueIDArray(ids []*ReadValueID) *ReadValueIDArray {
	if ids == nil {
		r := &ReadValueIDArray{
			ArraySize: 0,
		}
		return r
	}

	r := &ReadValueIDArray{
		ArraySize:    int32(len(ids)),
		ReadValueIDs: ids,
	}

	return r
}

// DecodeReadValueIDArray decodes given bytes into ReadValueIDArray.
func DecodeReadValueIDArray(b []byte) (*ReadValueIDArray, error) {
	r := &ReadValueIDArray{}
	if err := r.DecodeFromBytes(b); err != nil {
		return nil, err
	}
	return r, nil
}

// DecodeFromBytes decodes given bytes into ReadValueIDArray.
func (r *ReadValueIDArray) DecodeFromBytes(b []byte) error {
	r.ArraySize = int32(binary.LittleEndian.Uint32(b[:4]))
	if r.ArraySize <= 0 {
		return nil
	}

	offset := 4
	for i := 1; i <= int(r.ArraySize); i++ {
		id, err := DecodeReadValueID(b[offset:])
		if err != nil {
			return err
		}
		r.ReadValueIDs = append(r.ReadValueIDs, id)
		offset += id.Len()
	}

	return nil
}

// Serialize serializes ReadValueIDArray into bytes.
func (r *ReadValueIDArray) Serialize() ([]byte, error) {
	b := make([]byte, r.Len())
	if err := r.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes ReadValueIDArray into bytes.
func (r *ReadValueIDArray) SerializeTo(b []byte) error {
	var offset = 4
	binary.LittleEndian.PutUint32(b[:4], uint32(r.ArraySize))

	for _, id := range r.ReadValueIDs {
		if err := id.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += id.Len()
	}

	return nil
}

// Len returns the actual length in int.
func (r *ReadValueIDArray) Len() int {
	l := 4
	for _, id := range r.ReadValueIDs {
		l += id.Len()
	}
	return l
}
