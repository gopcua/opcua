// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uacp

// todo(fs): this test should be removed since the header is now sent in conn.go
// todo(fs): and is no longer part of the message itself.
// func TestHeader(t *testing.T) {
// 	cases := []codectest.Case{
// 		{
// 			Struct: &Header{
// 				MessageType: MessageTypeHello,
// 				ChunkType:   ChunkTypeFinal,
// 				MessageSize: 0xab,
// 			},
// 			Bytes: []byte{ // Hello message
// 				// MessageType: HEL
// 				0x48, 0x45, 0x4c,
// 				// Chunk Type: Final
// 				0x46,
// 				// MessageSize:
// 				0xab, 0x00,
// 			},
// 		},
// 	}
// 	codectest.Run(t, cases)
// }
