package uacp

import "sync"

type Buffer struct {
	buf []byte
}

func (b *Buffer) Bytes() []byte {
	return b.buf
}

var bufferPool sync.Pool

func AllocBuffer(size int) *Buffer {
	v := bufferPool.Get()
	if v == nil {
		return &Buffer{
			buf: make([]byte, size),
		}
	}
	buf := v.(*Buffer)
	if cap(buf.buf) < size {
		buf.buf = make([]byte, size)
	} else {
		buf.buf = buf.buf[:size]
	}
	return buf
}

func FreeBuffer(b *Buffer) {
	b.buf = b.buf[:0]
	bufferPool.Put(b)
}
