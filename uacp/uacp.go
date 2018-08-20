// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

// UACP is an interface for handling all type of UACP messages.
type UACP interface {
	Serialize() ([]byte, error)
	SerializeTo([]byte) error
	DecodeFromBytes([]byte) error
	Len() int
	String() string
	MessageTypeValue() string
	ChunkTypeValue() string
}

// Decode decodes given bytes as UACP.
func Decode(b []byte) (UACP, error) {
	var u UACP
	h, err := DecodeHeader(b)
	if err != nil {
		return nil, err
	}

	switch h.MessageTypeValue() {
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

	if err := u.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return u, nil
}

// Serialize serializes UACP messages regardless of its type.
func Serialize(message UACP) ([]byte, error) {
	b := make([]byte, message.Len())
	if err := message.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}
