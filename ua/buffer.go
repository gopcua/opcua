// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"encoding/binary"
	"io"
	"math"
	"time"

	"github.com/gopcua/opcua/errors"
)

const (
	null    = 0xffffffff
	f32qnan = 0xffc00000
	f64qnan = 0xfff8000000000000
)

// The range OPC 10000-6 §5.2.2.5 defines for a date/time: 1601-01-01 to
// 9999-12-31T23:59:59Z, the latter as a tick count in 100 nanosecond
// intervals since 1601-01-01.
const maxDateTimeTicks = int64(2650467743990000000)

var (
	minDateTime = time.Date(1601, time.January, 1, 0, 0, 0, 0, time.UTC)
	maxDateTime = time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC)
)

type Buffer struct {
	buf []byte
	pos int
	err error
}

func NewBuffer(b []byte) *Buffer {
	return &Buffer{buf: b}
}

func (b *Buffer) Error() error {
	return b.err
}

func (b *Buffer) Bytes() []byte {
	if b.err != nil {
		return nil
	}
	return b.buf[b.pos:]
}

func (b *Buffer) Pos() int {
	return b.pos
}

func (b *Buffer) Len() int {
	return len(b.buf) - b.pos
}

func (b *Buffer) ReadBool() bool {
	return b.ReadByte() > 0
}

func (b *Buffer) ReadByte() byte {
	if b.err != nil {
		return 0
	}
	d := b.ReadN(1)
	if d == nil {
		return 0
	}
	return d[0]
}

func (b *Buffer) ReadInt8() int8 {
	return int8(b.ReadByte())
}

func (b *Buffer) ReadInt16() int16 {
	return int16(b.ReadUint16())
}

func (b *Buffer) ReadUint16() uint16 {
	if b.err != nil {
		return 0
	}
	d := b.ReadN(2)
	if d == nil {
		return 0
	}
	return binary.LittleEndian.Uint16(d)
}

func (b *Buffer) ReadInt32() int32 {
	return int32(b.ReadUint32())
}

func (b *Buffer) ReadUint32() uint32 {
	if b.err != nil {
		return 0
	}
	d := b.ReadN(4)
	if d == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(d)
}

func (b *Buffer) ReadInt64() int64 {
	return int64(b.ReadUint64())
}

func (b *Buffer) ReadUint64() uint64 {
	if b.err != nil {
		return 0
	}
	d := b.ReadN(8)
	if d == nil {
		return 0
	}
	return binary.LittleEndian.Uint64(d)
}

func (b *Buffer) ReadFloat32() float32 {
	if b.err != nil {
		return 0
	}
	bits := b.ReadUint32()
	if b.err != nil {
		return 0
	}
	if bits == f32qnan {
		return float32(math.NaN())
	}
	return math.Float32frombits(bits)
}

func (b *Buffer) ReadFloat64() float64 {
	if b.err != nil {
		return 0
	}
	bits := b.ReadUint64()
	if b.err != nil {
		return 0
	}
	if bits == f64qnan {
		return math.NaN()
	}
	return math.Float64frombits(bits)
}

func (b *Buffer) ReadString() string {
	return string(b.ReadBytes())
}

func (b *Buffer) ReadBytes() []byte {
	n := b.ReadUint32()
	if b.err != nil {
		return nil
	}
	if n == 0 || n == null {
		return nil
	}
	d := b.ReadN(int(n))
	if b.err != nil {
		return nil
	}
	return d
}

func (b *Buffer) ReadStruct(r interface{}) {
	if b.err != nil {
		return
	}
	var n int
	var err error
	switch x := r.(type) {
	case BinaryDecoder:
		n, err = x.Decode(b.buf[b.pos:])
	default:
		n, err = Decode(b.buf[b.pos:], r)
	}
	if err != nil {
		b.err = err
		return
	}
	b.pos += n
}

func (b *Buffer) ReadTime() time.Time {
	d := b.ReadN(8)
	if b.err != nil {
		return time.Time{}
	}
	// the field is a signed Int64 (OPC 10000-6 §5.2.2.5), so read it as one
	// and saturate tick counts outside the range the clause covers.
	ts := int64(binary.LittleEndian.Uint64(d))
	switch {
	case ts <= 0:
		// a value at or below 0 decodes as time.Time{}, the value tick 0
		// decodes to - gopcua's null DateTime (see the IsZero callers).
		return time.Time{}
	case ts > maxDateTimeTicks:
		// the clause names its high end to the second, so the ticks after
		// 9999-12-31T23:59:59.0000000Z - the maximum Int64 included - are
		// later than the latest time it can represent and decode as it.
		return maxDateTime
	}
	// decode time in "100 nanosecond intervals since January 1, 1601" manner.
	// A full-range tick count exceeds int64 nanoseconds, so the conversion
	// goes through seconds and a sub-second remainder.
	sec, rem := ts/1e7, ts%1e7
	return time.Unix(sec-11644473600, rem*100).UTC()
}

func (b *Buffer) ReadN(n int) []byte {
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

func (b *Buffer) WriteBool(v bool) {
	if v {
		b.WriteUint8(1)
	} else {
		b.WriteUint8(0)
	}
}

func (b *Buffer) WriteByte(n byte) {
	b.buf = append(b.buf, n)
}

func (b *Buffer) WriteInt8(n int8) {
	b.buf = append(b.buf, byte(n))
}

func (b *Buffer) WriteUint8(n uint8) {
	b.buf = append(b.buf, byte(n))
}

func (b *Buffer) WriteInt16(n int16) {
	b.WriteUint16(uint16(n))
}

func (b *Buffer) WriteUint16(n uint16) {
	d := make([]byte, 2)
	binary.LittleEndian.PutUint16(d, n)
	b.Write(d)
}

func (b *Buffer) WriteInt32(n int32) {
	b.WriteUint32(uint32(n))
}

func (b *Buffer) WriteUint32(n uint32) {
	d := make([]byte, 4)
	binary.LittleEndian.PutUint32(d, n)
	b.Write(d)
}

func (b *Buffer) WriteInt64(n int64) {
	b.WriteUint64(uint64(n))
}

func (b *Buffer) WriteUint64(n uint64) {
	d := make([]byte, 8)
	binary.LittleEndian.PutUint64(d, n)
	b.Write(d)
}

func (b *Buffer) WriteFloat32(n float32) {
	if math.IsNaN(float64(n)) {
		b.WriteUint32(f32qnan)
	} else {
		b.WriteUint32(math.Float32bits(n))
	}
}

func (b *Buffer) WriteFloat64(n float64) {
	if math.IsNaN(n) {
		b.WriteUint64(f64qnan)
	} else {
		b.WriteUint64(math.Float64bits(n))
	}
}

func (b *Buffer) WriteString(s string) {
	if s == "" {
		b.WriteUint32(null)
		return
	}
	b.WriteByteString([]byte(s))
}

func (b *Buffer) WriteByteString(d []byte) {
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

func (b *Buffer) WriteStruct(w interface{}) {
	if b.err != nil {
		return
	}
	var d []byte
	switch x := w.(type) {
	case BinaryEncoder:
		d, b.err = x.Encode()
	default:
		d, b.err = Encode(w)
	}
	b.Write(d)
}

func (b *Buffer) WriteTime(v time.Time) {
	d := make([]byte, 8)
	if !v.IsZero() {
		// encode time in "100 nanosecond intervals since January 1, 1601",
		// clamped to the range OPC 10000-6 §5.2.2.5 defines: a time at or
		// before 1601-01-01 encodes as 0 and a time at or after
		// 9999-12-31T23:59:59Z as the maximum Int64.
		t := v.UTC()
		switch {
		case !t.After(minDateTime):
			// d stays zero
		case !t.Before(maxDateTime):
			binary.LittleEndian.PutUint64(d, math.MaxInt64)
		default:
			ts := t.Unix()*1e7 + int64(t.Nanosecond())/100 + 116444736000000000
			binary.LittleEndian.PutUint64(d, uint64(ts))
		}
	}
	b.Write(d)
}

func (b *Buffer) Write(d []byte) {
	if b.err != nil {
		return
	}
	b.buf = append(b.buf, d...)
}
