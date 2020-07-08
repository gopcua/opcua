// Generated code. DO NOT EDIT

// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package server

import 	"github.com/gopcua/opcua/ua"

func PredefinedNodes() []Node{
	return []Node{
		&node{
			id: ua.NewNumericNodeID(0, 3062),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 3063),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 1),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("Boolean"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("SByte"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 3),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("Byte"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 4),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("Int16"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 5),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("UInt16"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 6),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("Int32"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 7),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("UInt32"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 8),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("Int64"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 9),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("UInt64"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 10),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("Float"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("Double"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("String"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 13),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DateTime"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("Guid"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ByteString"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("XmlElement"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("NodeId"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ExpandedNodeId"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("StatusCode"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 20),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("QualifiedName"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("LocalizedText"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 23),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DataValue"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 25),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DiagnosticInfo"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 50),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("Decimal"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 35),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("Organizes"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 36),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasEventSource"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 37),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasModellingRule"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 38),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasEncoding"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 39),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasDescription"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 40),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasTypeDefinition"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 41),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("GeneratesEvent"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 3065),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("AlwaysGeneratesEvent"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 45),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasSubtype"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 46),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasProperty"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 47),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasComponent"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 48),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasNotifier"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 49),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasOrderedComponent"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 51),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("FromState"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 52),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("ToState"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 53),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasCause"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 54),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasEffect"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 117),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasSubStateMachine"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 56),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasHistoricalConfiguration"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 58),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("BaseObjectType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 61),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("FolderType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 63),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("BaseDataVariableType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 68),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("PropertyType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 69),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("DataTypeDescriptionType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 72),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("DataTypeDictionaryType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 75),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("DataTypeSystemType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 76),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("DataTypeEncodingType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 120),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("NamingRuleType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 77),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("ModellingRuleType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 78),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Mandatory"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 80),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Optional"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 83),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("ExposesItsArray"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11508),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("OptionalPlaceholder"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11510),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("MandatoryPlaceholder"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 84),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Root"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 85),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Objects"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 86),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Types"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 87),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Views"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 88),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("ObjectTypes"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 89),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("VariableTypes"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 90),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("DataTypes"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 91),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("ReferenceTypes"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 92),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("XML Schema"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 93),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("OPC Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 129),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasArgumentDescription"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 131),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasOptionalInputArgumentDescription"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15957),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("http://opcfoundation.org/UA/"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 3068),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("NodeVersion"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12170),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("ViewVersion"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 3067),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("Icon"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 3069),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("LocalTime"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 3070),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("AllowNulls"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11433),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("ValueAsText"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11498),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("MaxStringLength"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15002),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("MaxCharacters"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12908),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("MaxByteStringLength"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11512),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("MaxArrayLength"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11513),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("EngineeringUnits"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11432),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("EnumStrings"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 3071),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("EnumValues"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12745),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("OptionSetValues"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 3072),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("InputArguments"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 3073),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("OutputArguments"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16306),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("DefaultInputValues"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17605),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("DefaultInstanceBrowseName"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2000),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ImageBMP"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2001),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ImageGIF"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2002),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ImageJPG"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2003),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ImagePNG"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16307),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("AudioDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2004),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("ServerType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2013),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("ServerCapabilitiesType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2020),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("ServerDiagnosticsType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2026),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("SessionsDiagnosticsSummaryType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2029),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("SessionDiagnosticsObjectType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2033),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("VendorServerInfoType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2034),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("ServerRedundancyType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2036),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("TransparentRedundancyType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2039),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("NonTransparentRedundancyType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11945),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("NonTransparentNetworkRedundancyType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11564),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("OperationLimitsType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11575),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("FileType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11595),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AddressSpaceFileType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11616),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("NamespaceMetadataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11645),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("NamespacesType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2340),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AggregateFunctionType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2138),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("ServerStatusType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 3051),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("BuildInfoType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2150),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("ServerDiagnosticsSummaryType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2164),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("SamplingIntervalDiagnosticsArrayType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2165),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("SamplingIntervalDiagnosticsType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2171),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("SubscriptionDiagnosticsArrayType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2172),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("SubscriptionDiagnosticsType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2196),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("SessionDiagnosticsArrayType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2197),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("SessionDiagnosticsVariableType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2243),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("SessionSecurityDiagnosticsArrayType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2244),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("SessionSecurityDiagnosticsType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11487),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("OptionSetType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16309),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("SelectionListType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17986),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("AudioVariableType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 3048),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("EventTypes"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2253),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Server"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11192),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("HistoryServerCapabilities"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11737),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("BitFieldMaskDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14533),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("KeyValuePair"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15528),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("EndpointType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2299),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("StateMachineType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2755),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("StateVariableType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2762),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("TransitionVariableType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2760),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("FiniteStateVariableType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2767),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("FiniteTransitionVariableType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2307),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("StateType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2309),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("InitialStateType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2310),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("TransitionType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15109),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("ChoiceStateType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15112),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasGuard"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15113),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("GuardVariableType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15128),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("ExpressionGuardVariableType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15317),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("ElseGuardVariableType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17709),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("RationalNumberType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17716),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("3DVectorType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18774),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("3DCartesianCoordinatesType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18781),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("3DOrientationType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18791),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("3DFrameType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18806),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("RationalNumber"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18808),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("3DVector"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18810),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("3DCartesianCoordinates"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18812),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("3DOrientation"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18814),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("3DFrame"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11939),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("OpenFileMode"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 13353),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("FileDirectoryType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16314),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("FileSystem"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15744),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("TemporaryFileTransferType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15803),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("FileTransferStateMachineType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15607),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("RoleSetType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15620),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("RoleType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15632),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("IdentityCriteriaType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15634),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("IdentityMappingRuleType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15644),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Anonymous"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15656),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("AuthenticatedUser"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15668),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Observer"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15680),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Operator"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16036),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Engineer"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15692),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Supervisor"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15716),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("ConfigureAdmin"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15704),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("SecurityAdmin"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17591),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("DictionaryFolderType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17594),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Dictionaries"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17597),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasDictionaryEntry"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17598),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("IrdiDictionaryEntryType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17600),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("UriDictionaryEntryType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17708),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("InterfaceTypes"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17603),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasInterface"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17604),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasAddIn"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 23498),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("CurrencyUnitType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 23501),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("CurrencyUnit"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2365),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("DataItemType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15318),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("BaseAnalogType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2368),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("AnalogItemType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17497),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("AnalogUnitType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17570),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("AnalogUnitRangeType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2373),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("TwoStateDiscreteType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2376),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("MultiStateDiscreteType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11238),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("MultiStateValueDiscreteType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12029),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("YArrayItemType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12038),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("XYArrayItemType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12047),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("ImageItemType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12057),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("CubeItemType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12068),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("NDimensionArrayItemType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 8995),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("TwoStateVariableType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 9002),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("ConditionVariableType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 9004),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasTrueSubState"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 9005),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasFalseSubState"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16361),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasAlarmSuppressionGroup"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16362),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("AlarmGroupMember"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2830),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("DialogConditionType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2881),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AcknowledgeableConditionType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2915),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AlarmConditionType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16405),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AlarmGroupType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2929),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("ShelvedStateMachineType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2955),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("LimitAlarmType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 9318),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("ExclusiveLimitStateMachineType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 9341),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("ExclusiveLimitAlarmType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 9906),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("NonExclusiveLimitAlarmType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 10060),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("NonExclusiveLevelAlarmType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 9482),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("ExclusiveLevelAlarmType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 10368),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("NonExclusiveDeviationAlarmType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 10214),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("NonExclusiveRateOfChangeAlarmType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 9764),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("ExclusiveDeviationAlarmType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 9623),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("ExclusiveRateOfChangeAlarmType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 10523),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("DiscreteAlarmType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 10637),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("OffNormalAlarmType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 10751),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("TripAlarmType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18347),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("InstrumentDiagnosticAlarmType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18496),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("SystemDiagnosticAlarmType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 13225),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("CertificateExpirationAlarmType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17080),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("DiscrepancyAlarmType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2790),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AuditConditionEventType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2803),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AuditConditionEnableEventType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2829),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AuditConditionCommentEventType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 8927),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AuditConditionRespondEventType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 8944),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AuditConditionAcknowledgeEventType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 8961),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AuditConditionConfirmEventType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11093),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AuditConditionShelvingEventType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17225),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AuditConditionSuppressionEventType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17242),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AuditConditionSilenceEventType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15013),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AuditConditionResetEventType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17259),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AuditConditionOutOfServiceEventType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 9006),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasCondition"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17276),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasEffectDisable"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17983),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasEffectEnable"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17984),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasEffectSuppressed"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17985),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasEffectUnsuppressed"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17279),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AlarmMetricsType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17277),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("AlarmRateVariableType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2391),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("ProgramStateMachineType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 3806),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("ProgramTransitionAuditEventType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2380),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("ProgramDiagnosticType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15383),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("ProgramDiagnostic2Type"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11214),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("Annotations"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2318),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("HistoricalDataConfigurationType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11202),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("HA Configuration"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11215),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("HistoricalEventFilter"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2330),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("HistoryServerCapabilitiesType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12522),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("TrustListType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12552),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("TrustListMasks"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12554),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("TrustListDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19297),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("TrustListOutOfDateAlarmType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12555),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("CertificateGroupType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 13813),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("CertificateGroupFolderType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12558),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("HttpsCertificateType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15181),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("UserCredentialCertificateType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12559),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("RsaMinApplicationCertificateType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12560),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("RsaSha256ApplicationCertificateType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12581),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("ServerConfigurationType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12637),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("ServerConfiguration"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17496),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("KeyCredentialConfigurationFolderType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18155),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("KeyCredentialConfiguration"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18001),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("KeyCredentialConfigurationType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18029),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("KeyCredentialUpdatedAuditEventType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18047),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("KeyCredentialDeletedAuditEventType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17732),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("AuthorizationServices"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17852),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AuthorizationServiceConfigurationType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11187),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("AggregateConfigurationType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2341),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Interpolative"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2342),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Average"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2343),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("TimeAverage"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11285),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("TimeAverage2"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2344),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Total"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11304),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Total2"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2346),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Minimum"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2347),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Maximum"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2348),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("MinimumActualTime"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2349),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("MaximumActualTime"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2350),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Range"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11286),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Minimum2"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11287),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Maximum2"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11305),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("MinimumActualTime2"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11306),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("MaximumActualTime2"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11288),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Range2"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2351),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("AnnotationCount"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2352),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Count"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11307),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("DurationInStateZero"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11308),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("DurationInStateNonZero"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2355),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("NumberOfTransitions"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2357),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Start"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2358),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("End"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2359),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Delta"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11505),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("StartBound"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11506),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("EndBound"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11507),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("DeltaBounds"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2360),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("DurationGood"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2361),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("DurationBad"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2362),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("PercentGood"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2363),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("PercentBad"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 2364),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("WorstQuality"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11292),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("WorstQuality2"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11426),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("StandardDeviationSample"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11427),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("StandardDeviationPopulation"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11428),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("VarianceSample"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11429),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("VariancePopulation"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15487),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("StructureDescription"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15488),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("EnumDescription"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15005),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("SimpleTypeDescription"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15006),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("UABinaryFileDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14647),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("PubSubState"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14523),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DataSetMetaDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14524),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("FieldMetaData"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15904),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DataSetFieldFlags"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14593),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ConfigurationVersionDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15578),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("PublishedDataSetDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14273),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("PublishedVariableDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15581),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("PublishedDataItemsDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15582),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("PublishedEventsDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15583),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DataSetFieldContentMask"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15597),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DataSetWriterDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15480),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("WriterGroupDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15617),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("PubSubConnectionDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15510),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("NetworkAddressUrlDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15520),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ReaderGroupDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15623),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DataSetReaderDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15631),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("TargetVariablesDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14744),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("FieldTargetDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15874),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("OverrideValueHandling"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15635),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("SubscribedDataSetMirrorDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15530),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("PubSubConfigurationDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 20408),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DataSetOrderingType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15642),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("UadpNetworkMessageContentMask"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15645),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("UadpWriterGroupMessageDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15646),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("UadpDataSetMessageContentMask"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15652),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("UadpDataSetWriterMessageDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15653),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("UadpDataSetReaderMessageDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15654),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("JsonNetworkMessageContentMask"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15657),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("JsonWriterGroupMessageDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15658),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("JsonDataSetMessageContentMask"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15664),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("JsonDataSetWriterMessageDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15665),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("JsonDataSetReaderMessageDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17467),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DatagramConnectionTransportDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15532),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DatagramWriterGroupTransportDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15007),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("BrokerConnectionTransportDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15008),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("BrokerTransportQualityOfService"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15667),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("BrokerWriterGroupTransportDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15669),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("BrokerDataSetWriterTransportDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15670),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("BrokerDataSetReaderTransportDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15906),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("PubSubKeyServiceType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15452),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("SecurityGroupFolderType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15471),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("SecurityGroupType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14416),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("PublishSubscribeType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14443),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("PublishSubscribe"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14476),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasPubSubConnection"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14509),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("PublishedDataSetType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15489),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("ExtensionFieldsType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14936),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("DataSetToWriter"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14534),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("PublishedDataItemsType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14572),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("PublishedEventsType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14477),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("DataSetFolderType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14209),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("PubSubConnectionType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17725),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("WriterGroupType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15296),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasDataSetWriter"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18804),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasWriterGroup"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17999),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("ReaderGroupType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15297),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasDataSetReader"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18805),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ReferenceType_32"),
				ua.AttributeIDBrowseName: ua.MustVariant("HasReaderGroup"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15298),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("DataSetWriterType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15306),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("DataSetReaderType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15108),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("SubscribedDataSetType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15111),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("TargetVariablesType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15127),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("SubscribedDataSetMirrorType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14643),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("PubSubStatusType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19723),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DiagnosticsLevel"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19725),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("PubSubDiagnosticsCounterType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19730),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("PubSubDiagnosticsCounterClassification"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19732),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("PubSubDiagnosticsRootType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19786),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("PubSubDiagnosticsConnectionType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19834),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("PubSubDiagnosticsWriterGroupType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19903),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("PubSubDiagnosticsReaderGroupType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19968),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("PubSubDiagnosticsDataSetWriterType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 20027),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("PubSubDiagnosticsDataSetReaderType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21105),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("UadpWriterGroupMessageType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21111),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("UadpDataSetWriterMessageType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21116),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("UadpDataSetReaderMessageType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21126),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("JsonWriterGroupMessageType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21128),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("JsonDataSetWriterMessageType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21130),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("JsonDataSetReaderMessageType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15064),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("DatagramConnectionTransportType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21133),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("DatagramWriterGroupTransportType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15155),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("BrokerConnectionTransportType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21136),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("BrokerWriterGroupTransportType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21138),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("BrokerDataSetWriterTransportType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21142),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("BrokerDataSetReaderTransportType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21147),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("ObjectType_8"),
				ua.AttributeIDBrowseName: ua.MustVariant("NetworkAddressUrlType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19077),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("MultiStateDictionaryEntryDiscreteBaseType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19084),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("VariableType_16"),
				ua.AttributeIDBrowseName: ua.MustVariant("MultiStateDictionaryEntryDiscreteType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 256),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("IdType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 257),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("NodeClass"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 94),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("PermissionType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15031),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("AccessLevelType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15406),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("AccessLevelExType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15033),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("EventNotifierType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 95),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("AccessRestrictionType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 96),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("RolePermissionType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 98),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("StructureType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 101),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("StructureField"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 99),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("StructureDefinition"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 100),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("EnumDefinition"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 296),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("Argument"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 7594),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("EnumValueType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 102),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("EnumField"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12755),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("OptionSet"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12877),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("NormalizedString"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12878),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DecimalString"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12879),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DurationString"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12880),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("TimeString"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12881),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DateString"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 290),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("Duration"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 294),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("UtcTime"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 295),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("LocaleId"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 8912),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("TimeZoneDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17588),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("Index"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 288),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("IntegerId"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 307),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ApplicationType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 308),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ApplicationDescription"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 20998),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("VersionTime"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12189),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ServerOnNetwork"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 311),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ApplicationInstanceCertificate"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 302),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("MessageSecurityMode"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 303),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("UserTokenType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 304),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("UserTokenPolicy"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 312),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("EndpointDescription"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 432),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("RegisteredServer"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12890),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DiscoveryConfiguration"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12891),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("MdnsDiscoveryConfiguration"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 315),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("SecurityTokenRequestType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 344),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("SignedSoftwareCertificate"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 388),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("SessionAuthenticationToken"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 319),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("AnonymousIdentityToken"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 322),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("UserNameIdentityToken"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 325),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("X509IdentityToken"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 938),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("IssuedIdentityToken"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 348),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("NodeAttributesMask"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 376),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("AddNodesItem"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 379),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("AddReferencesItem"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 382),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DeleteNodesItem"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 385),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DeleteReferencesItem"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 347),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("AttributeWriteMask"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 521),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ContinuationPoint"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 537),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("RelativePathElement"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 540),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("RelativePath"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 289),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("Counter"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 291),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("NumericRange"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 292),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("Time"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 293),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("Date"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 331),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("EndpointConfiguration"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 576),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("FilterOperator"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 583),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ContentFilterElement"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 586),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ContentFilter"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 589),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("FilterOperand"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 592),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ElementOperand"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 595),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("LiteralOperand"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 598),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("AttributeOperand"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 601),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("SimpleAttributeOperand"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 659),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("HistoryEvent"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11234),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("HistoryUpdateType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11293),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("PerformUpdateType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 719),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("MonitoringFilter"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 725),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("EventFilter"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 948),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("AggregateConfiguration"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 920),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("HistoryEventFieldList"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 338),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("BuildInfo"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 851),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("RedundancySupport"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 852),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ServerState"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 853),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("RedundantServerDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11943),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("EndpointUrlListDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11944),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("NetworkGroupDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 856),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("SamplingIntervalDiagnosticsDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 859),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ServerDiagnosticsSummaryDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 862),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ServerStatusDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 865),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("SessionDiagnosticsDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 868),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("SessionSecurityDiagnosticsDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 871),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ServiceCounterDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 299),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("StatusResult"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 874),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("SubscriptionDiagnosticsDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 877),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ModelChangeStructureDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 897),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("SemanticChangeStructureDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 884),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("Range"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 887),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("EUInformation"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12077),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("AxisScaleEnumeration"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12171),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ComplexNumberType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12172),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("DoubleComplexNumberType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12079),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("AxisInformation"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12080),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("XVType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 894),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ProgramDiagnosticDataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15396),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ProgramDiagnostic2DataType"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 891),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("Annotation"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 890),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("DataType_64"),
				ua.AttributeIDBrowseName: ua.MustVariant("ExceptionDeviationFormat"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14846),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15671),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18815),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18816),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18817),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18818),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18819),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18820),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18821),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18822),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18823),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15736),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 23507),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12680),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15676),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 125),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 126),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 127),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15421),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15422),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 124),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14839),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14847),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15677),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15678),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14323),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15679),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15681),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15682),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15683),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15688),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15689),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21150),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15691),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15693),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15694),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15695),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21151),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21152),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21153),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15701),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15702),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15703),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15705),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15706),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15707),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15712),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14848),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15713),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21154),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15715),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15717),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15718),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15719),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15724),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15725),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17468),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21155),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15479),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15727),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15729),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15733),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 128),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 121),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14844),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 122),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 123),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 298),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 8251),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14845),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12765),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12766),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 8917),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 310),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12207),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 306),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 314),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 434),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12900),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12901),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 346),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 318),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 321),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 324),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 327),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 940),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 378),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 381),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 384),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 387),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 539),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 542),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 333),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 585),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 588),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 591),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 594),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 597),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 600),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 603),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 661),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 721),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 727),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 950),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 922),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 340),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 855),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11957),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11958),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 858),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 861),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 864),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 867),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 870),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 873),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 301),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 876),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 879),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 899),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 886),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 889),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12181),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12182),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12089),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12090),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 896),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15397),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 893),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default Binary"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 7617),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("Opc.Ua"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14802),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15949),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18851),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18852),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18853),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18854),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18855),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18856),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18857),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18858),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 18859),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15728),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 23520),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12676),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15950),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14796),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15589),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15590),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15529),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15531),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14794),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14795),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14803),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15951),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15952),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14319),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15953),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15954),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15955),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15956),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15987),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15988),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21174),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15990),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15991),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15992),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15993),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21175),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21176),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21177),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15995),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15996),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16007),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16008),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16009),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16010),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16011),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14804),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16012),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21178),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16014),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16015),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16016),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16017),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16018),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16019),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17472),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21179),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15579),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16021),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16022),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16023),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16126),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14797),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14800),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14798),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14799),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 297),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 7616),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 14801),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12757),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12758),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 8913),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 309),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12195),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 305),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 313),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 433),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12892),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12893),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 345),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 317),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 320),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 323),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 326),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 939),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 377),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 380),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 383),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 386),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 538),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 541),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 332),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 584),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 587),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 590),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 593),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 596),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 599),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 602),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 660),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 720),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 726),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 949),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 921),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 339),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 854),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11949),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 11950),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 857),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 860),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 863),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 866),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 869),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 872),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 300),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 875),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 878),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 898),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 885),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 888),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12173),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12174),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12081),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 12082),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 895),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15401),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 892),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default XML"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 8252),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Variable_2"),
				ua.AttributeIDBrowseName: ua.MustVariant("Opc.Ua"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15041),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16150),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19064),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19065),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19066),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19067),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19068),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19069),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19070),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19071),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 19072),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15042),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 23528),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15044),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16151),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15057),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15058),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15059),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15700),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15714),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15050),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15051),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15049),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16152),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16153),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15060),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16154),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16155),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16156),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16157),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16158),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16159),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21198),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16161),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16280),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16281),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16282),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21199),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21200),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21201),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16284),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16285),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16286),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16287),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16288),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16308),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16310),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15061),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16311),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21202),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16323),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16391),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16392),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16393),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16394),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16404),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 17476),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 21203),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15726),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16524),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16525),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 16526),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15062),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15063),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15065),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15066),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15067),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15081),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15082),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15083),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15084),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15085),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15086),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15087),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15095),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15098),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15099),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15102),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15105),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15106),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15136),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15140),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15141),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15142),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15143),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15144),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15165),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15169),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15172),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15175),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15188),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15189),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15199),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15204),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15205),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15206),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15207),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15208),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15209),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15210),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15273),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15293),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15295),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15304),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15349),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15361),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15362),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15363),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15364),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15365),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15366),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15367),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15368),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15369),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15370),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15371),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15372),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15373),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15374),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15375),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15376),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15377),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15378),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15379),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15380),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15381),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15405),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
		&node{
			id: ua.NewNumericNodeID(0, 15382),
			attr: map[ua.AttributeID]*ua.Variant{
				ua.AttributeIDNodeClass: ua.MustVariant("Object_1"),
				ua.AttributeIDBrowseName: ua.MustVariant("Default JSON"),
			},
		},
	}
}
