// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package utils

// Uint24To32 converts 24bits-length []byte value into the uint32 with 8bits of zeros as prefix.
// This function is used for the fields with 3 octets.
func Uint24To32(b []byte) uint32 {
	if len(b) != 3 {
		return 0
	}
	return uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
}

// Uint32To24 converts the uint32 value into 24bits-length []byte. The values in 25-32 bit are cut off.
// This function is used for the fields with 3 octets.
func Uint32To24(n uint32) []byte {
	return []byte{uint8(n >> 16), uint8(n >> 8), uint8(n)}
}
