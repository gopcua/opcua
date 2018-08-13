// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/binary"

	"github.com/wmnsk/gopcua/errors"
)

// GUID represents GUID in binary stream.
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 uint64
}

// DecodeFromBytes decodes given bytes into GUID.
func (g *GUID) DecodeFromBytes(b []byte) error {
	if len(b) != 16 {
		return &errors.ErrTooShortToDecode{g, "should be 16 bytes"}
	}

	g.Data1 = binary.LittleEndian.Uint32(b[:4])
	g.Data2 = binary.LittleEndian.Uint16(b[4:6])
	g.Data3 = binary.LittleEndian.Uint16(b[6:8])
	g.Data4 = binary.LittleEndian.Uint64(b[8:16])
	return nil
}
