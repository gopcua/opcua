// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import "github.com/wmnsk/gopcua/datatypes"

// AdditionalHeader represents the AdditionalHeader.
type AdditionalHeader struct {
	TypeID       *datatypes.ExpandedNodeID
	EncodingMask uint8
}

// DecodeFromBytes decodes given bytes into AdditionalHeader.
func (a *AdditionalHeader) DecodeFromBytes(b []byte) error {
	if err := a.TypeID.DecodeFromBytes(b); err != nil {
		return err
	}
	a.EncodingMask = b[a.TypeID.Len()]

	return nil
}
