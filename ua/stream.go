package ua

import (
	"encoding/binary"
	"io"
	"math"
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

func (s *Stream) WrapError(err error) {
	s.err = errors.Join(err)
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
