// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/wmnsk/gopcua/errors"
)

// GUID represents GUID in binary stream. It is a 16-byte globally unique identifier.
//
// Specification: Part 6, 5.1.3
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 uint64
}

// NewGUID creates a new GUID.
// Input should be GUID string of 16 hexadecimal characters like 1111AAAA-22BB-33CC-44DD-55EE77FF9900.
// Dash can be omitted, and alphabets are not case-sensitive.
func NewGUID(guid string) *GUID {
	h := strings.Replace(guid, "-", "", -1)
	b, err := hex.DecodeString(h)
	if err != nil {
		return nil
	}
	if len(b) < 16 {
		return nil
	}

	g := &GUID{}
	g.Data1 = binary.BigEndian.Uint32(b[:4])
	g.Data2 = binary.BigEndian.Uint16(b[4:6])
	g.Data3 = binary.BigEndian.Uint16(b[6:8])
	g.Data4 = binary.BigEndian.Uint64(b[8:16])

	return g
}

// DecodeGUID decodes given bytes into GUID.
func DecodeGUID(b []byte) (*GUID, error) {
	g := &GUID{}
	if err := g.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return g, nil
}

// DecodeFromBytes decodes given bytes into GUID.
func (g *GUID) DecodeFromBytes(b []byte) error {
	if len(b) < 16 {
		return &errors.ErrTooShortToDecode{g, "should be 16 bytes"}
	}

	g.Data1 = binary.LittleEndian.Uint32(b[:4])
	g.Data2 = binary.LittleEndian.Uint16(b[4:6])
	g.Data3 = binary.LittleEndian.Uint16(b[6:8])
	g.Data4 = binary.LittleEndian.Uint64(b[8:16])
	return nil
}

// Serialize serializes GUID into bytes.
func (g *GUID) Serialize() ([]byte, error) {
	b := make([]byte, g.Len())
	if err := g.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes GUID into given bytes.
func (g *GUID) SerializeTo(b []byte) error {
	binary.LittleEndian.PutUint32(b[:4], g.Data1)
	binary.LittleEndian.PutUint16(b[4:6], g.Data2)
	binary.LittleEndian.PutUint16(b[6:8], g.Data3)
	binary.LittleEndian.PutUint64(b[8:16], g.Data4)

	return nil
}

// Len returns the actual size of GUID in int.
func (g *GUID) Len() int {
	return 16
}

// String returns GUID in human-readable string.
func (g *GUID) String() string {
	d4 := make([]byte, 8)
	binary.BigEndian.PutUint64(d4, g.Data4)

	return fmt.Sprintf("%X-%X-%X-%X-%X",
		g.Data1,
		g.Data2,
		g.Data3,
		d4[:2],
		d4[2:],
	)
}
