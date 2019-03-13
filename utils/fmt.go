// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package utils

import (
	"bytes"
	"fmt"
)

// Wireshark prints the given byte slice in same format as the Wireshark application.
// It prints 16 bytes per line with a space after 8 bytes.
// This is useful for debugging messages sent over the wire
// and for comparing messages with other implementations.
// The offset is required because the message might not start at the beginning of the line.
// Without the offset the printed message is shifted compared to the message in wireshark.
func Wireshark(offset int, message []byte) string {
	var buf bytes.Buffer
	var line []byte

	// add offset
	message = append(make([]byte, offset), message...)

	// add content
	for i, b := range message {
		line = append(line, b)

		// add space after first 8 bytes but not at the end of the line, i.e. after 16 bytes
		if (i+1)%8 == 0 && (i+1)%16 != 0 {
			fmt.Fprintf(&buf, "% x  ", line)
			line = []byte{}
		}

		// add line break after 16 bytes and at the end of the message
		if (i+1)%16 == 0 || i == len(message)-1 {
			fmt.Fprintf(&buf, "% x\n", line)
			line = []byte{}
		}
	}
	return buf.String()
}
