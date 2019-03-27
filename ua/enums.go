// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// additional enum values which are not generated.

const (
	NodeClassAll NodeClass = 0xff
)

type AttributeID uint32

// Identifiers assigned to Attributes.
//
// Specification: Part 6, A.1
const (
	AttributeIDInvalid                 AttributeID = 0
	AttributeIDNodeID                  AttributeID = 1
	AttributeIDNodeClass               AttributeID = 2
	AttributeIDBrowseName              AttributeID = 3
	AttributeIDDisplayName             AttributeID = 4
	AttributeIDDescription             AttributeID = 5
	AttributeIDWriteMask               AttributeID = 6
	AttributeIDUserWriteMask           AttributeID = 7
	AttributeIDIsAbstract              AttributeID = 8
	AttributeIDSymmetric               AttributeID = 9
	AttributeIDInverseName             AttributeID = 10
	AttributeIDContainsNoLoops         AttributeID = 11
	AttributeIDEventNotifier           AttributeID = 12
	AttributeIDValue                   AttributeID = 13
	AttributeIDDataType                AttributeID = 14
	AttributeIDValueRank               AttributeID = 15
	AttributeIDArrayDimensions         AttributeID = 16
	AttributeIDAccessLevel             AttributeID = 17
	AttributeIDUserAccessLevel         AttributeID = 18
	AttributeIDMinimumSamplingInterval AttributeID = 19
	AttributeIDHistorizing             AttributeID = 20
	AttributeIDExecutable              AttributeID = 21
	AttributeIDUserExecutable          AttributeID = 22
	AttributeIDDataTypeDefinition      AttributeID = 23
	AttributeIDRolePermissions         AttributeID = 24
	AttributeIDUserRolePermissions     AttributeID = 25
	AttributeIDAccessRestrictions      AttributeID = 26
	AttributeIDAccessLevelEx           AttributeID = 27
)

type Type byte

// Built-in Type identifiers.
//
// All OPC UA DataEncodings are based on rules that are defined for a standard set of built-in types. These built-in types are then used to construct structures, arrays and Messages.
//
// Part 6, 5.1.2
const (
	TypeBoolean         Type = 1
	TypeSByte           Type = 2
	TypeByte            Type = 3
	TypeInt16           Type = 4
	TypeUint16          Type = 5
	TypeInt32           Type = 6
	TypeUint32          Type = 7
	TypeInt64           Type = 8
	TypeUint64          Type = 9
	TypeFloat           Type = 10
	TypeDouble          Type = 11
	TypeString          Type = 12
	TypeDateTime        Type = 13
	TypeGUID            Type = 14
	TypeByteString      Type = 15
	TypeXMLElement      Type = 16
	TypeNodeID          Type = 17
	TypeExpandedNodeID  Type = 18
	TypeStatusCode      Type = 19
	TypeQualifiedName   Type = 20
	TypeLocalizedText   Type = 21
	TypeExtensionObject Type = 22
	TypeDataValue       Type = 23
	TypeVariant         Type = 24
	TypeDiagnosticInfo  Type = 25
)
