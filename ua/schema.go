package ua

import (
	"fmt"
	"reflect"
	"time"

	"github.com/pascaldekloe/name"
)

// Serializable checks if the Go struct can be serialized into the UA type name.
func Serializable(t reflect.Type, id string, types map[string]*StructureDefinition) error {
	uatyp, err := UAType(id, types)
	if err != nil {
		return err
	}
	uadef, godef := GoString(uatyp), GoString(anonType(t))
	if uadef != godef {
		return fmt.Errorf(`schema: Go type "%s" is not serializable into UA type "%s" (%s)`, godef, uadef, id)
	}
	return nil
}

// GoString returns a Go-syntax representation of the type of the value.
func GoString(t reflect.Type) string {
	return fmt.Sprintf("%T", reflect.New(t).Interface())
}

// anonType returns the struct without the name.
func anonType(t reflect.Type) reflect.Type {
	switch {
	// do not unpack time.Time
	case isTimeType(t):
		return t
	case t.Kind() == reflect.Slice:
		return reflect.SliceOf(anonType(t.Elem()))
	case t.Kind() == reflect.Struct:
		return reflect.StructOf(structFields(t))
	default:
		return t
	}
}

// structFields returns the fields of a struct as an array.
// it panics if t is not a struct.
func structFields(t reflect.Type) []reflect.StructField {
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("%s is not a struct", t.Name()))
	}
	var fields []reflect.StructField
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		f.Name = name.CamelCase(f.Name, true)
		f.Type = anonType(f.Type)
		fields = append(fields, f)
	}
	return fields
}

// uaType returns an anonymous Go representation of the OPC/UA struct definition with the NodeID 'id'.
func UAType(id string, types map[string]*StructureDefinition) (reflect.Type, error) {
	t := types[id]
	if t == nil {
		return nil, fmt.Errorf("schema: ua type %q not defined", id)
	}

	// we don't support structs with optional fields or unions (yet or maybe never)
	if t.StructureType != StructureTypeStructure {
		return nil, fmt.Errorf("schema: ua type %q has unsupported structure type %s", id, t.StructureType)
	}

	var fields []reflect.StructField
	for _, f := range t.Fields {
		if len(f.ArrayDimensions) > 1 {
			return nil, fmt.Errorf("schema: ua type %s.%s is a multi-dimensional array: %v", id, f.Name, f.ArrayDimensions)
		}

		var elem reflect.Type
		switch f.DataType.String() {
		case "i=1":
			elem = reflect.TypeOf(bool(false))
		case "i=2":
			elem = reflect.TypeOf(int8(0))
		case "i=3":
			elem = reflect.TypeOf(uint8(0))
		case "i=4":
			elem = reflect.TypeOf(int16(0))
		case "i=5":
			elem = reflect.TypeOf(uint16(0))
		case "i=6":
			elem = reflect.TypeOf(int32(0))
		case "i=7":
			elem = reflect.TypeOf(uint32(0))
		case "i=8":
			elem = reflect.TypeOf(int64(0))
		case "i=9":
			elem = reflect.TypeOf(uint64(0))
		case "i=10":
			elem = reflect.TypeOf(float32(0))
		case "i=11":
			elem = reflect.TypeOf(float64(0))
		case "i=12":
			// todo(fs): maybe encode length constrained strings as []rune
			// if f.MaxStringLength > 0 {
			// 	elem = reflect.TypeOf(make([]rune, f.MaxStringLength))
			// }
			elem = reflect.TypeOf(string(""))
		case "i=13":
			elem = reflect.TypeOf(time.Time{})
		default:
			fid := f.DataType.String()
			ft := types[fid]
			switch {
			case ft.BaseDataType.String() == "i=22":
				var err error
				elem, err = UAType(fid, types)
				if err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("schema: invalid data type %s", fid)
			}
		}

		typ := elem
		for rank := f.ValueRank; rank > 0; rank-- {
			typ = reflect.SliceOf(typ)
		}

		fname := name.CamelCase(f.Name, true)
		fields = append(fields, reflect.StructField{Name: fname, Type: typ})
	}

	return reflect.StructOf(fields), nil
}

func isTimeType(t reflect.Type) bool { return t == reflect.TypeOf(time.Time{}) }
