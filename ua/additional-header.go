// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// AdditionalHeader represents the AdditionalHeader.
// TODO: add body handling.
// type AdditionalHeader struct {
// 	TypeID       *ExpandedNodeID
// 	EncodingMask uint8
// }

// // NewAdditionalHeader creates a new AdditionalHeader.
// func NewAdditionalHeader(typeID *ExpandedNodeID, mask uint8) *AdditionalHeader {
// 	return &AdditionalHeader{
// 		TypeID:       typeID,
// 		EncodingMask: mask,
// 	}
// }

// // NewNullAdditionalHeader creates a new AdditionalHeader without meaningful values.
// func NewNullAdditionalHeader() *AdditionalHeader {
// 	return &AdditionalHeader{
// 		TypeID:       NewTwoByteExpandedNodeID(0),
// 		EncodingMask: 0x00,
// 	}
// }

// // HasBinaryBody checks if an AdditionalHeader has binary body above.
// func (a *AdditionalHeader) HasBinaryBody() bool {
// 	return a.EncodingMask&0x1 == 1
// }

// // HasXMLBody checks if an AdditionalHeader has binary body above.
// func (a *AdditionalHeader) HasXMLBody() bool {
// 	return (a.EncodingMask>>1)&0x1 == 1
// }
