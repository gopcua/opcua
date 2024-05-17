package ua

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
	"time"

	"github.com/gopcua/opcua/errors"
)

const DefaultBufSize = 1024

type Stream struct {
	buf []byte
	pos int
	err error
}

func NewStream(size int) *Stream {
	return &Stream{
		buf: make([]byte, 0, size),
	}
}

func (s *Stream) Error() error {
	return s.err
}

func (s *Stream) Len() int {
	return len(s.buf)
}

func (s *Stream) Reset() {
	s.buf = s.buf[:0]
	s.pos = 0
	s.err = nil
}

func (s *Stream) Bytes() []byte {
	return s.buf
}

func (b *Stream) ReadN(n int) []byte {
	if b.err != nil {
		return nil
	}
	d := b.buf[b.pos:]
	if n > len(d) {
		b.err = io.ErrUnexpectedEOF
		return nil
	}
	b.pos += n
	return d[:n]
}

func (b *Stream) WriteBool(v bool) {
	if v {
		b.WriteUint8(1)
	} else {
		b.WriteUint8(0)
	}
}

func (b *Stream) WriteByte(n byte) {
	b.buf = append(b.buf, n)
}

func (b *Stream) WriteInt8(n int8) {
	b.buf = append(b.buf, byte(n))
}

func (b *Stream) WriteUint8(n uint8) {
	b.buf = append(b.buf, byte(n))
}

func (b *Stream) WriteInt16(n int16) {
	b.WriteUint16(uint16(n))
}

func (b *Stream) WriteUint16(n uint16) {
	d := make([]byte, 2)
	binary.LittleEndian.PutUint16(d, n)
	b.Write(d)
}

func (b *Stream) WriteInt32(n int32) {
	b.WriteUint32(uint32(n))
}

func (b *Stream) WriteUint32(n uint32) {
	d := make([]byte, 4)
	binary.LittleEndian.PutUint32(d, n)
	b.Write(d)
}

func (b *Stream) WriteInt64(n int64) {
	b.WriteUint64(uint64(n))
}

func (b *Stream) WriteUint64(n uint64) {
	d := make([]byte, 8)
	binary.LittleEndian.PutUint64(d, n)
	b.Write(d)
}

func (b *Stream) WriteFloat32(n float32) {
	if math.IsNaN(float64(n)) {
		b.WriteUint32(f32qnan)
	} else {
		b.WriteUint32(math.Float32bits(n))
	}
}

func (b *Stream) WriteFloat64(n float64) {
	if math.IsNaN(n) {
		b.WriteUint64(f64qnan)
	} else {
		b.WriteUint64(math.Float64bits(n))
	}
}

func (b *Stream) WriteString(s string) {
	if s == "" {
		b.WriteUint32(null)
		return
	}
	b.WriteByteString([]byte(s))
}

func (b *Stream) WriteByteString(d []byte) {
	if b.err != nil {
		return
	}
	if len(d) > math.MaxInt32 {
		b.err = errors.Errorf("value too large")
		return
	}
	if d == nil {
		b.WriteUint32(null)
		return
	}
	b.WriteUint32(uint32(len(d)))
	b.Write(d)
}

func (b *Stream) WriteTime(v time.Time) {
	d := make([]byte, 8)
	if !v.IsZero() {
		// encode time in "100 nanosecond intervals since January 1, 1601"
		ts := uint64(v.UTC().UnixNano()/100 + 116444736000000000)
		binary.LittleEndian.PutUint64(d, ts)
	}
	b.Write(d)
}

func (b *Stream) Write(d []byte) {
	if b.err != nil {
		return
	}
	b.buf = append(b.buf, d...)
}

func (s *Stream) WriteStruct(w interface{}) {
	if s.err != nil {
		return
	}
	val := reflect.ValueOf(w)
	switch x := w.(type) {
	case ValEncoder:
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
	case isValEncoder(val):
		v := val.Interface().(ValEncoder)
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
