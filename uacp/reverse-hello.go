// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"fmt"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/errors"
)

// ReverseHello represents a OPC UA ReverseHello.
type ReverseHello struct {
	*Header
	ServerURI   *datatypes.String
	EndPointURL *datatypes.String
}

// NewReverseHello creates a new OPC UA ReverseHello.
func NewReverseHello(serverURI, endpoint string) *ReverseHello {
	r := &ReverseHello{
		Header: NewHeader(
			MessageTypeReverseHello,
			ChunkTypeFinal,
			nil,
		),
		ServerURI:   datatypes.NewString(serverURI),
		EndPointURL: datatypes.NewString(endpoint),
	}
	r.SetLength()

	return r
}

// DecodeReverseHello decodes given bytes into OPC UA ReverseHello.
func DecodeReverseHello(b []byte) (*ReverseHello, error) {
	r := &ReverseHello{}
	if err := r.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return r, nil
}

// DecodeFromBytes decodes given bytes into OPC UA ReverseHello.
func (r *ReverseHello) DecodeFromBytes(b []byte) error {
	var err error
	if len(b) < 8 {
		return errors.NewErrTooShortToDecode(r, "should be longer than 8 bytes")
	}

	r.Header, err = DecodeHeader(b)
	if err != nil {
		return err
	}
	b = r.Header.Payload

	var offset = 0
	r.ServerURI = &datatypes.String{}
	if err := r.ServerURI.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += r.ServerURI.Len()

	r.EndPointURL = &datatypes.String{}
	return r.EndPointURL.DecodeFromBytes(b[offset:])
}

// Serialize serializes OPC UA ReverseHello into bytes.
func (r *ReverseHello) Serialize() ([]byte, error) {
	b := make([]byte, int(r.MessageSize))
	if err := r.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes OPC UA ReverseHello into given bytes.
func (r *ReverseHello) SerializeTo(b []byte) error {
	if r == nil {
		return errors.NewErrReceiverNil(r)
	}
	r.Header.Payload = make([]byte, r.Len()-8)

	var offset = 0
	if r.ServerURI != nil {
		if err := r.ServerURI.SerializeTo(r.Header.Payload[offset:]); err != nil {
			return err
		}
		offset += r.ServerURI.Len()
	}

	if r.EndPointURL != nil {
		if err := r.EndPointURL.SerializeTo(r.Header.Payload[offset:]); err != nil {
			return err
		}
	}

	r.Header.SetLength()
	return r.Header.SerializeTo(b)
}

// Len returns the actual length of ReverseHello in int.
func (r *ReverseHello) Len() int {
	var l = 8
	if r.ServerURI != nil {
		l += r.ServerURI.Len()
	}

	if r.EndPointURL != nil {
		l += r.EndPointURL.Len()
	}

	return l
}

// SetLength sets the length of ReverseHello.
func (r *ReverseHello) SetLength() {
	r.MessageSize = 8

	if r.ServerURI != nil {
		r.MessageSize += uint32(r.ServerURI.Len())
	}

	if r.EndPointURL != nil {
		r.MessageSize += uint32(r.EndPointURL.Len())
	}
}

// String returns ReverseHello in string.
func (r *ReverseHello) String() string {
	return fmt.Sprintf(
		"Header: %v, ServerURI: %s, EndPointURL: %s",
		r.Header,
		r.ServerURI.Get(),
		r.EndPointURL.Get(),
	)
}
