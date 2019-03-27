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
	TypeGUID            = 14
	TypeByteString      = 15
	TypeXMLElement      = 16
	TypeNodeID          = 17
	TypeExpandedNodeID  = 18
	TypeStatusCode      = 19
	TypeQualifiedName   = 20
	TypeLocalizedText   = 21
	TypeExtensionObject = 22
	TypeDataValue       = 23
	TypeVariant         = 24
	TypeDiagnosticInfo  = 25
)
