// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"encoding/xml"
	"os"
)

type TypeDictionary struct {
	XMLName xml.Name      `xml:"TypeDictionary"`
	Types   []*StructType `xml:"StructuredType"`
	Enums   []*EnumType   `xml:"EnumeratedType"`
}

type EnumType struct {
	Name   string       `xml:",attr"`
	Bits   int          `xml:"LengthInBits,attr"`
	Doc    string       `xml:"Documentation"`
	Values []*EnumValue `xml:"EnumeratedValue"`
}

type EnumValue struct {
	Name  string `xml:",attr"`
	Value int    `xml:",attr"`
}

type StructType struct {
	Name     string         `xml:",attr"`
	BaseType string         `xml:"BaseType,attr"`
	Doc      string         `xml:"Documentation"`
	Fields   []*StructField `xml:"Field"`
}

func (s *StructType) IsLengthField(f *StructField) bool {
	for _, ff := range s.Fields {
		if f.Name == ff.LengthField {
			return true
		}
	}
	return false
}

type StructField struct {
	Name        string `xml:",attr"`
	Type        string `xml:"TypeName,attr"`
	LengthField string `xml:",attr"`
	SwitchField string `xml:",attr"`
	SwitchValue string `xml:",attr"`
	IsEnum      bool
}

func (f *StructField) IsSlice() bool {
	return f.LengthField != ""
}

func ReadTypes(filename string) (*TypeDictionary, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	d := new(TypeDictionary)
	if err := xml.NewDecoder(f).Decode(&d); err != nil {
		return nil, err
	}

	enums := map[string]bool{}
	for _, e := range d.Enums {
		enums["tns:"+e.Name] = true
	}

	for _, t := range d.Types {
		for _, f := range t.Fields {
			f.IsEnum = enums[f.Type]
		}
	}
	return d, nil
}
