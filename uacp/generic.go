// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

import (
	"fmt"

	"github.com/wmnsk/gopcua/errors"
)

// Generic is an alias of OPC UA Header.
// This is implemented for handling undefined type of messages.
type Generic struct {
	*Header
}

// NewGeneric creates a new OPC UA Generic.
func NewGeneric(msgType, chunkType string, payload []byte) *Generic {
	g := &Generic{
		Header: NewHeader(
			msgType,
			chunkType,
			payload,
		),
	}
	g.SetLength()

	return g
}

// DecodeGeneric decodes given bytes into OPC UA Generic.
func DecodeGeneric(b []byte) (*Generic, error) {
	g := &Generic{}
	if err := g.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return g, nil
}

// DecodeFromBytes decodes given bytes into OPC UA Generic.
func (g *Generic) DecodeFromBytes(b []byte) error {
	var err error
	if len(b) < 8 {
		return &errors.ErrTooShortToDecode{g, "should be longer than 8 bytes"}
	}

	g.Header, err = DecodeHeader(b)
	if err != nil {
		return err
	}
	return nil
}

// Serialize serializes OPC UA Generic into bytes.
func (g *Generic) Serialize() ([]byte, error) {
	return g.Header.Serialize()
}

// SerializeTo serializes OPC UA Generic into given bytes.
// TODO: add error handling.
func (g *Generic) SerializeTo(b []byte) error {
	return g.Header.SerializeTo(b)
}

// Len returns the actual length of Generic in int.
func (g *Generic) Len() int {
	return g.Header.Len()
}

// SetLength sets the length of Generic.
func (g *Generic) SetLength() {
	g.Header.SetLength()
}

// String returns Generic in string.
func (g *Generic) String() string {
	return fmt.Sprintf(
		"Header: %v, Payload: %s",
		g.Header,
		g.Header.Payload,
	)
}
