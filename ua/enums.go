// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// additional enum values which are not generated.

const (
	NodeClassAll NodeClass = 0xff
)

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

// datatypes
const (
	TypeBoolean         = 1
	TypeSByte           = 2
	TypeByte            = 3
	TypeInt16           = 4
	TypeUint16          = 5
	TypeInt32           = 6
	TypeUint32          = 7
	TypeInt64           = 8
	TypeUint64          = 9
	TypeFloat           = 10
	TypeDouble          = 11
	TypeString          = 12
	TypeDateTime        = 13
	TypeGuid            = 14
	TypeByteString      = 15
	TypeXmlElement      = 16
	TypeNodeId          = 17
	TypeExpandedNodeId  = 18
	TypeStatusCode      = 19
	TypeQualifiedName   = 20
	TypeLocalizedText   = 21
	TypeExtensionObject = 22
	TypeDataValue       = 23
	TypeVariant         = 24
	TypeDiagnosticInfo  = 25
)
