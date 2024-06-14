package codec

import "encoding/binary"

type Stream struct {
	buf []byte
}

func NewStream(buf []byte) *Stream {
	return &Stream{buf: buf}
}

func (s *Stream) Write(p []byte) (n int, err error) {
	s.buf = append(s.buf, p...)
	return len(p), nil
}

func (s *Stream) WriteString(str string) (n int, err error) {
	s.buf = append(s.buf, str...)
	return len(str), nil
}

func (s *Stream) WriteByte(b byte) error {
	s.buf = append(s.buf, b)
	return nil
}

func (s *Stream) Reset() {
	s.buf = s.buf[:0]
}

func (s *Stream) Bytes() []byte {
	return s.buf[:len(s.buf)]
}

func (s *Stream) WriteUint16(n uint16) {
	s.buf = binary.LittleEndian.AppendUint16(s.buf, n)
}

func (s *Stream) WriteUint32(n uint32) {
	s.buf = binary.LittleEndian.AppendUint32(s.buf, n)
}

func (s *Stream) WriteUint64(n uint64) {
	s.buf = binary.LittleEndian.AppendUint64(s.buf, n)
}
