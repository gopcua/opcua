// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// IntegerID is a UInt32 that is used as an identifier, such as a handle.
// All values, except for 0, are valid.
//
// Specification: Part 4, 7.14
// type IntegerID uint32

// Identifiers assigned to Attributes.
//
// Specification: Part 6, A.1
const (
	IntegerIDInvalid = iota
	IntegerIDNodeID
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
// type ReadValueID struct {
// 	NodeID       *NodeID
// 	AttributeID  uint32
// 	IndexRange   string
// 	DataEncoding *QualifiedName
// }

// NewReadValueID creates a new ReadValueID.
// func NewReadValueID(nodeID *NodeID, attrID uint32, idxRange string, qIdx uint16, qName string) *ReadValueID {
// 	return &ReadValueID{
// 		NodeID:       nodeID,
// 		AttributeID:  attrID,
// 		IndexRange:   idxRange,
// 		DataEncoding: NewQualifiedName(qIdx, qName),
// 	}
// }

// func (v *ReadValueID) Decode(b []byte) (int, error) {
// 	buf := NewBuffer(b)
// 	v.NodeID = new(NodeID)
// 	buf.ReadStruct(v.NodeID)
// 	v.AttributeID = buf.ReadUint32()
// 	v.IndexRange = buf.ReadString()
// 	v.DataEncoding = new(QualifiedName)
// 	buf.ReadStruct(v.DataEncoding)
// 	return buf.Pos(), buf.Error()
// }

// func (v *ReadValueID) Encode() ([]byte, error) {
// 	buf := NewBuffer(nil)
// 	buf.WriteStruct(v.NodeID)
// 	buf.WriteUint32(v.AttributeID)
// 	buf.WriteString(v.IndexRange)
// 	buf.WriteStruct(v.DataEncoding)
// 	return buf.Bytes(), buf.Error()
// }
