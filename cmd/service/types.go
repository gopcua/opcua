// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/wmnsk/gopcua/cmd/service/opc"
)

type Kind uint

const (
	Invalid Kind = iota
	Bit
	Bool
	Int8
	Int16
	Int32
	Int64
	Uint8
	Uint16
	Uint32
	Uint64
	Float32
	Float64
	Enum
	Slice
	LenSlice
	String
	Struct
	Interface
	Scalar
)

func (k Kind) String() string {
	switch k {
	case Bit:
		return "bit"
	case Bool:
		return "bool"
	case Int8:
		return "int8"
	case Int16:
		return "int16"
	case Int32:
		return "int32"
	case Int64:
		return "int64"
	case Uint8:
		return "uint8"
	case Uint16:
		return "uint16"
	case Uint32:
		return "uint32"
	case Uint64:
		return "uint64"
	case Float32:
		return "float32"
	case Float64:
		return "float64"
	case Enum:
		return "enum"
	case Slice:
		return "slice"
	case LenSlice:
		return "len_slice"
	case String:
		return "string"
	case Struct:
		return "struct"
	case Interface:
		return "interface"
	case Scalar:
		return "scalar"
	default:
		return fmt.Sprintf("invalid kind %d", k)
	}
}

// builtins is the list of types which already exist.
var builtins = []*Type{
	&Type{Name: "Bit", XmlName: "opc:Bit", Kind: Bit, Builtin: true, GoType: "bool", RWPackagePrefix: "ua."},
	&Type{Name: "Boolean", XmlName: "opc:Boolean", Kind: Bool, Builtin: true, GoType: "bool", RWPackagePrefix: "ua."},
	&Type{Name: "SByte", XmlName: "opc:SByte", Kind: Int8, Builtin: true, GoType: "int8", RWPackagePrefix: "ua."},
	&Type{Name: "Int16", XmlName: "opc:Int16", Kind: Int16, Builtin: true, GoType: "int16", RWPackagePrefix: "ua."},
	&Type{Name: "Int32", XmlName: "opc:Int32", Kind: Int32, Builtin: true, GoType: "int32", RWPackagePrefix: "ua."},
	&Type{Name: "Int64", XmlName: "opc:Int64", Kind: Int64, Builtin: true, GoType: "int64", RWPackagePrefix: "ua."},
	&Type{Name: "Byte", XmlName: "opc:Byte", Kind: Uint8, Builtin: true, GoType: "uint8", RWPackagePrefix: "ua."},
	&Type{Name: "UInt16", XmlName: "opc:UInt16", Kind: Uint16, Builtin: true, GoType: "uint16", RWPackagePrefix: "ua."},
	&Type{Name: "UInt32", XmlName: "opc:UInt32", Kind: Uint32, Builtin: true, GoType: "uint32", RWPackagePrefix: "ua."},
	&Type{Name: "UInt64", XmlName: "opc:UInt64", Kind: Uint64, Builtin: true, GoType: "uint64", RWPackagePrefix: "ua."},
	&Type{Name: "Float", XmlName: "opc:Float", Kind: Float32, Builtin: true, GoType: "float32", RWPackagePrefix: "ua."},
	&Type{Name: "Double", XmlName: "opc:Double", Kind: Float64, Builtin: true, GoType: "float64", RWPackagePrefix: "ua."},
	&Type{Name: "String", XmlName: "opc:String", Kind: String, Builtin: true, GoType: "string", RWPackagePrefix: "ua."},
	&Type{Name: "ByteString", XmlName: "opc:ByteString", Kind: LenSlice, Builtin: true, GoType: "[]byte", RWPackagePrefix: "ua."},
	&Type{Name: "Char", XmlName: "opc:Char", Kind: Uint8, Builtin: true, GoType: "byte", RWPackagePrefix: "ua."},
	&Type{Name: "CharArray", XmlName: "opc:CharArray", Kind: LenSlice, Builtin: true, GoType: "[]byte", RWPackagePrefix: "ua."},

	&Type{Name: "ByteStringNodeId", XmlName: "ua:ByteStringNodeId", Kind: Struct, Builtin: true, GoType: "ua.ByteStringNodeId", RWPackagePrefix: "ua."},
	&Type{Name: "DataValue", XmlName: "ua:DataValue", Kind: Struct, Builtin: true, GoType: "ua.DataValue", RWPackagePrefix: "ua."},
	&Type{Name: "DateTime", XmlName: "opc:DateTime", Kind: Scalar, Builtin: true, GoType: "time.Time", RWPackagePrefix: "ua."},
	&Type{Name: "DiagnosticInfo", XmlName: "ua:DiagnosticInfo", Kind: Struct, Builtin: true, GoType: "ua.DiagnosticInfo", RWPackagePrefix: "ua."},
	&Type{Name: "ExpandedNodeId", XmlName: "ua:ExpandedNodeId", Kind: Struct, Builtin: true, GoType: "ua.ExpandedNodeId", RWPackagePrefix: "ua."},
	&Type{Name: "ExtensionObject", XmlName: "ua:ExtensionObject", Kind: Struct, Builtin: true, GoType: "ua.ExtensionObject", RWPackagePrefix: "ua."},
	&Type{Name: "FourByteNodeId", XmlName: "ua:FourByteNodeId", Kind: Struct, Builtin: true, GoType: "ua.FourByteNodeId", RWPackagePrefix: "ua."},
	&Type{Name: "Guid", XmlName: "opc:Guid", Kind: Struct, Builtin: true, GoType: "ua.Guid", RWPackagePrefix: "ua."},
	&Type{Name: "GuidNodeId", XmlName: "ua:GuidNodeId", Kind: Struct, Builtin: true, GoType: "ua.GuidNodeId", RWPackagePrefix: "ua."},
	&Type{Name: "LocalizedText", XmlName: "ua:LocalizedText", Kind: Struct, Builtin: true, GoType: "ua.LocalizedText", RWPackagePrefix: "ua."},
	&Type{Name: "NodeId", XmlName: "ua:NodeId", Kind: Interface, Builtin: true, GoType: "ua.NodeId", RWPackagePrefix: "ua."},
	&Type{Name: "NodeIdType", XmlName: "ua:NodeIdType", Kind: Enum, Builtin: true, GoType: "ua.NodeIdType", RWPackagePrefix: "ua."},
	&Type{Name: "StatusCode", XmlName: "ua:StatusCode", Kind: Uint8, Builtin: true, GoType: "ua.StatusCode", RWPackagePrefix: "ua."},
	&Type{Name: "NumericNodeId", XmlName: "ua:NumericNodeId", Kind: Struct, Builtin: true, GoType: "ua.NumericNodeId", RWPackagePrefix: "ua."},
	&Type{Name: "QualifiedName", XmlName: "ua:QualifiedName", Kind: Struct, Builtin: true, GoType: "ua.QualifiedName", RWPackagePrefix: "ua."},
	&Type{Name: "StringNodeId", XmlName: "ua:StringNodeId", Kind: Struct, Builtin: true, GoType: "ua.StringNodeId", RWPackagePrefix: "ua."},
	&Type{Name: "TwoByteNodeId", XmlName: "ua:TwoByteNodeId", Kind: Struct, Builtin: true, GoType: "ua.TwoByteNodeId", RWPackagePrefix: "ua."},
	&Type{Name: "Variant", XmlName: "ua:Variant", Kind: Struct, Builtin: true, GoType: "ua.Variant", RWPackagePrefix: "ua."},
	&Type{Name: "XmlElement", XmlName: "ua:XmlElement", Kind: Struct, Builtin: true, GoType: "ua.XmlElement", RWPackagePrefix: "ua."},
}

// Type is the representation of an OPC/UA type.
type Type struct {
	// Name is the name of the type, e.g. string, Foo,
	Name string

	// XmlName is the fully qualified name of the type in the OPC/UA schema.
	// It is the name other structs use to refer to this type.
	XmlName string

	// ElemType is the type of slice and enum elements.
	ElemType *Type

	// Kind describes what kind of type this Type represents.
	Kind Kind

	// Builtin returns whether the type is a builtin type
	// from the datatypes package like string, uint32, ...
	Builtin bool

	// Interface describes whether the Go type is an interface.
	Interface bool

	// GoType is the Go type
	GoType string

	// Bits contains the number of bits required for an enum.
	Bits int

	// NodeID is the id for the binary encoding of a struct field.
	NodeID int

	// RWPackagePrefix contains the package name of the ReadWrite function.
	// e.g. 'ua.'
	RWPackagePrefix string

	// Fields contains the fields of a struct.
	Fields []*StructField

	// Values contains the values of an enum.
	Values []*EnumValue

	// Doc contains the optional documentation for the type.
	Doc string
}

func (t Type) String() string {
	s := "name: " + t.Name
	s += " xml_name: " + t.XmlName
	s += " kind: " + t.Kind.String()
	s += " go_type: " + t.GoType
	s += " builtin: " + strconv.FormatBool(t.Builtin)
	if t.Kind == Struct {
		s += " num_fields: " + strconv.Itoa(len(t.Fields))
	}
	return s
}

func (t *Type) GoString() string {
	return "type " + t.Name + " " + t.GoType
}

// EnumValue is a value of an enum type.
type EnumValue struct {
	Name  string
	Value int
}

func (e *EnumValue) GoString() string {
	return fmt.Sprintf("%s = 0x%x", e.Name, e.Value)
}

// StructField is a field of a struct type.
type StructField struct {
	Name    string
	XmlName string
	Type    *Type
}

func (f *StructField) String() string {
	s := "name: " + f.Name
	s += " xml_name: " + f.XmlName
	s += " type: " + f.Type.Name
	s += " go_type: " + f.Type.GoType
	return s
}

func (f *StructField) GoString() string {
	return f.Name + " " + f.Type.GoType
}

// Types generates the type definitions for all OPC/UA types in the type
// dictionary which are not builtin types. Each service object should have
// a corresponding entry in the nodeIDs map which contains the identifier
// of the binary encoding for that service object.
func Types(dict *opc.TypeDictionary, nodeIDs map[string]int) []*Type {
	types := map[string]*Type{}
	for _, t := range builtins {
		types[t.Name] = t
	}

	// create all enums
	for _, v := range dict.Enums {
		if types[v.Name] != nil {
			continue
		}
		t := &Type{
			Name:    v.Name,
			XmlName: "tns:" + v.Name, // non-builtins are in the tns namespace
			GoType:  v.Name,
			Kind:    Enum,
			Bits:    v.Bits,
			Doc:     v.Doc,
		}

		switch {
		case t.Bits <= 8:
			t.ElemType = types["Byte"]
		case t.Bits <= 16:
			t.ElemType = types["UInt16"]
		case t.Bits <= 32:
			t.ElemType = types["UInt32"]
		case t.Bits <= 64:
			t.ElemType = types["UInt64"]
		default:
			panic(fmt.Sprintf("No ElemType for %s", v.Name))
		}

		for _, val := range v.Values {
			ev := &EnumValue{
				Name:  v.Name + val.Name,
				Value: val.Value,
			}
			t.Values = append(t.Values, ev)
		}
		types[t.Name] = t
	}

	// create all structs
	for _, v := range dict.Types {
		if types[v.Name] != nil {
			continue
		}
		t := &Type{
			Name:    v.Name,
			XmlName: "tns:" + v.Name, // non-builtins are in the tns namespace
			Kind:    Struct,
			GoType:  v.Name,
			NodeID:  nodeIDs[v.Name],
		}
		types[t.Name] = t
	}

	// mark all structs as pointers
	for _, t := range types {
		if t.Kind == Struct {
			t.GoType = "*" + t.GoType
		}
	}

	// add struct fields for non-builtin types
	for _, opct := range dict.Types {
		t := types[opct.Name]

		// skip builtins since the code is already written
		if t.Builtin {
			continue
		}

		for _, opcf := range opct.Fields {
			// skip the field if it is a length field
			// since all arrays are length encoded
			if opct.IsLengthField(opcf) {
				continue
			}

			f := &StructField{
				Name:    opcf.Name,
				XmlName: opcf.Type,
			}

			// find the type for the field
			for _, typ := range types {
				if opcf.Type == typ.XmlName {
					f.Type = typ
					break
				}
			}

			// if the field is an array then use
			// the corresponding array type if it
			// exists or create one if it does not.
			if opcf.LengthField != "" {
				name := f.XmlName
				switch {
				case strings.HasPrefix(name, "opc:"):
					name = name[4:] + "Array"
				case strings.HasPrefix(name, "tns:"):
					name = name[4:] + "Array"
				case strings.HasPrefix(name, "ua:"):
					name = name[3:] + "Array"
				}

				typ := types[name]
				if typ == nil {
					typ = &Type{
						Name:     name,
						Kind:     LenSlice,
						GoType:   "[]" + f.Type.GoType,
						ElemType: f.Type,
					}
					types[name] = typ
				}
				f.Type = typ
			}

			t.Fields = append(t.Fields, f)
		}
	}

	var a []*Type
	for _, t := range types {
		a = append(a, t)
	}
	sort.Sort(byName(a))
	return a
}

// byName sorts Types by name.
type byName []*Type

func (a byName) Len() int           { return len(a) }
func (a byName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byName) Less(i, j int) bool { return a[i].Name < a[j].Name }
