// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/wmnsk/gopcua"
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

	return &GUID{
		Data1: binary.BigEndian.Uint32(b[:4]),
		Data2: binary.BigEndian.Uint16(b[4:6]),
		Data3: binary.BigEndian.Uint16(b[6:8]),
		Data4: binary.BigEndian.Uint64(b[8:16]),
	}
}

func (g *GUID) Decode(b []byte) (int, error) {
	buf := gopcua.NewBuffer(b)
	g.Data1 = buf.ReadUint32()
	g.Data2 = buf.ReadUint16()
	g.Data3 = buf.ReadUint16()
	g.Data4 = buf.ReadUint64()
	return buf.Pos(), buf.Error()
}

func (g *GUID) Encode() ([]byte, error) {
	buf := gopcua.NewBuffer(nil)
	buf.WriteUint32(g.Data1)
	buf.WriteUint16(g.Data2)
	buf.WriteUint16(g.Data3)
	buf.WriteUint64(g.Data4)
	return buf.Bytes(), buf.Error()
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
