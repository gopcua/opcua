// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/gopcua/opcua/debug"
	"github.com/gopcua/opcua/errors"
)

// debugCodec enables printing of debug messages in the opcua codec.
var debugCodec = debug.FlagSet("codec")

// BinaryEncoder is the interface implemented by an object that can
// marshal itself into a binary OPC/UA representation.
type BinaryEncoder interface {
	Encode(s *Stream) error
}

var binaryEncoder = reflect.TypeOf((*BinaryEncoder)(nil)).Elem()

func isBinaryEncoder(val reflect.Value) bool {
	return val.Type().Implements(binaryEncoder)
}

func (s *Stream) WriteAny(w interface{}) {
	if s.err != nil {
		return
	}
	val := reflect.ValueOf(w)
	switch x := w.(type) {
	case BinaryEncoder:
		s.err = x.Encode(s)
	default:
		s.err = s.encode(val, val.Type().String())
	}
}

func (s *Stream) encode(val reflect.Value, name string) error {
	if debugCodec {
		fmt.Printf("encode: %s has type %s and is a %s\n", name, val.Type(), val.Type().Kind())
	}

	switch {
	case isBinaryEncoder(val):
		v := val.Interface().(BinaryEncoder)
		return v.Encode(s)

	case isTime(val):
		s.WriteTime(val.Convert(timeType).Interface().(time.Time))

	default:
		switch val.Kind() {
		case reflect.Bool:
			s.WriteBool(val.Bool())
		case reflect.Int8:
			s.WriteInt8(int8(val.Int()))
		case reflect.Uint8:
			s.WriteUint8(uint8(val.Uint()))
		case reflect.Int16:
			s.WriteInt16(int16(val.Int()))
		case reflect.Uint16:
			s.WriteUint16(uint16(val.Uint()))
		case reflect.Int32:
			s.WriteInt32(int32(val.Int()))
		case reflect.Uint32:
			s.WriteUint32(uint32(val.Uint()))
		case reflect.Int64:
			s.WriteInt64(int64(val.Int()))
		case reflect.Uint64:
			s.WriteUint64(uint64(val.Uint()))
		case reflect.Float32:
			s.WriteFloat32(float32(val.Float()))
		case reflect.Float64:
			s.WriteFloat64(float64(val.Float()))
		case reflect.String:
			s.WriteString(val.String())
		case reflect.Ptr:
			if val.IsNil() {
				return nil
			}
			return s.encode(val.Elem(), name)
		case reflect.Struct:
			return s.writeStruct(val, name)
		case reflect.Slice:
			return s.writeSlice(val, name)
		case reflect.Array:
			return s.writeArray(val, name)
		default:
			return errors.Errorf("unsupported type: %s", val.Type())
		}
	}
	return s.Error()
}

func (s *Stream) writeStruct(val reflect.Value, name string) error {
	valt := val.Type()
	for i := 0; i < val.NumField(); i++ {
		ft := valt.Field(i)
		fname := name + "." + ft.Name
		if err := s.encode(val.Field(i), fname); err != nil {
			return err
		}
	}
	return nil
}

func (s *Stream) writeSlice(val reflect.Value, name string) error {
	if val.IsNil() {
		s.WriteUint32(null)
		return s.Error()
	}

	if val.Len() > math.MaxInt32 {
		return errors.Errorf("array too large")
	}

	s.WriteUint32(uint32(val.Len()))

	// fast path for []byte
	if val.Type().Elem().Kind() == reflect.Uint8 {
		// fmt.Println("[]byte fast path")
		s.Write(val.Bytes())
		return s.Error()
	}

	// loop over elements
	for i := 0; i < val.Len(); i++ {
		ename := fmt.Sprintf("%s[%d]", name, i)
		s.encode(val.Index(i), ename)
		if s.Error() != nil {
			return s.Error()
		}
	}
	return s.Error()
}

func (s *Stream) writeArray(val reflect.Value, name string) error {
	if val.Len() > math.MaxInt32 {
		return errors.Errorf("array too large: %d > %d", val.Len(), math.MaxInt32)
	}

	s.WriteUint32(uint32(val.Len()))

	// fast path for []byte
	if val.Type().Elem().Kind() == reflect.Uint8 {
		// fmt.Println("encode: []byte fast path")
		b := make([]byte, val.Len())
		reflect.Copy(reflect.ValueOf(b), val)
		s.Write(b)
		return s.Error()
	}

	// loop over elements
	// we write all the elements, also the zero values
	for i := 0; i < val.Len(); i++ {
		ename := fmt.Sprintf("%s[%d]", name, i)
		s.encode(val.Index(i), ename)
		if s.Error() != nil {
			return s.Error()
		}
	}
	return s.Error()
}
