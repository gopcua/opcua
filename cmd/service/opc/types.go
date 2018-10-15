package opc

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
	Name   string   `xml:",attr"`
	Bits   int      `xml:"LengthInBits,attr"`
	Doc    string   `xml:"Documentation"`
	Values []*Value `xml:"EnumeratedValue"`
}

type Value struct {
	Name  string `xml:",attr"`
	Value int    `xml:",attr"`
}

type StructType struct {
	Name   string   `xml:",attr"`
	Doc    string   `xml:"Documentation"`
	Fields []*Field `xml:"Field"`
}

func (s *StructType) IsLengthField(f *Field) bool {
	for _, ff := range s.Fields {
		if f.Name == ff.LengthField {
			return true
		}
	}
	return false
}

type Field struct {
	Name        string `xml:",attr"`
	Type        string `xml:"TypeName,attr"`
	LengthField string `xml:",attr"`
	SwitchField string `xml:",attr"`
	SwitchValue string `xml:",attr"`
}

func (f *Field) IsSlice() bool {
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
	return d, nil
}
