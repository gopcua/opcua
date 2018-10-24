// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import "github.com/wmnsk/gopcua"

// Decode decodes given bytes as UACP.
func Decode(b []byte) (interface{}, error) {
	hdr := new(Header)
	n, err := hdr.Decode(b)
	if err != nil {
		return nil, err
	}
	b = b[n:]

	var u interface{}
	switch hdr.MessageType {
	case MessageTypeHello:
		u = &Hello{}
	case MessageTypeAcknowledge:
		u = &Acknowledge{}
	case MessageTypeError:
		u = &Error{}
	case MessageTypeReverseHello:
		u = &ReverseHello{}
	default:
		u = &Generic{}
	}

	if err := gopcua.Decode(b, u); err != nil {
		return nil, err
	}
	return u, nil
}

func Encode(msgType string, chunkType byte, v interface{}) ([]byte, error) {
	body, err := gopcua.Encode(v)
	if err != nil {
		return nil, err
	}
	hdr := &Header{
		MessageType: msgType,
		ChunkType:   chunkType,
		MessageSize: uint32(len(body) + 12),
	}
	b, err := hdr.Encode()
	if err != nil {
		return nil, err
	}
	b = append(b, body...)
	return b, nil
}
