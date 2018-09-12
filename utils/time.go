// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package utils

import (
	"encoding/binary"
	"time"
)

// EncodeTimestamp serializes time.Time into given bytes buffer
// in "100 nanosecond intervals since January 1, 1601" manner.
func EncodeTimestamp(b []byte, t time.Time) {
	binary.LittleEndian.PutUint64(b, uint64(t.UTC().UnixNano()/100+116444736000000000))
}

// DecodeTimestamp decodes given bytes into time.Time
// in "100 nanosecond intervals since January 1, 1601" manner.
func DecodeTimestamp(b []byte) time.Time {
	t := binary.LittleEndian.Uint64(b[:8])
	return time.Unix(0, int64((t-116444736000000000)*100)).UTC()
}
