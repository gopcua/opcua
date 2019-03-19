// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"time"
)

// These flags define which fields of a DataValue are set.
// Bits are or'ed together if multiple fields are set.
const (
	DataValueValue             = 0x1
	DataValueStatus            = 0x2
	DataValueSourceTimestamp   = 0x4
	DataValueServerTimestamp   = 0x8
	DataValueSourcePicoseconds = 0x10
	DataValueServerPicoseconds = 0x20
)

// DataValue is always preceded by a mask that indicates which fields are present in the stream.
//
// Specification: Part 6, 5.2.2.17
type DataValue struct {
	EncodingMask      byte
	Value             *Variant
	Status            uint32
	SourceTimestamp   time.Time
	SourcePicoseconds uint16
	ServerTimestamp   time.Time
	ServerPicoseconds uint16
}

func (d *DataValue) Decode(b []byte) (int, error) {
	buf := NewBuffer(b)
	d.EncodingMask = buf.ReadByte()
	if d.Has(DataValueValue) {
		d.Value = new(Variant)
		buf.ReadStruct(d.Value)
	}
	if d.Has(DataValueStatus) {
		d.Status = buf.ReadUint32()
	}
	if d.Has(DataValueSourceTimestamp) {
		d.SourceTimestamp = buf.ReadTime()
	}
	if d.Has(DataValueSourcePicoseconds) {
		d.SourcePicoseconds = buf.ReadUint16()
	}
	if d.Has(DataValueServerTimestamp) {
		d.ServerTimestamp = buf.ReadTime()
	}
	if d.Has(DataValueServerPicoseconds) {
		d.ServerPicoseconds = buf.ReadUint16()
	}
	return buf.Pos(), buf.Error()
}

func (d *DataValue) Encode() ([]byte, error) {
	buf := NewBuffer(nil)
	buf.WriteUint8(d.EncodingMask)

	if d.Has(DataValueValue) {
		buf.WriteStruct(d.Value)
	}
	if d.Has(DataValueStatus) {
		buf.WriteUint32(d.Status)
	}
	if d.Has(DataValueSourceTimestamp) {
		buf.WriteTime(d.SourceTimestamp)
	}
	if d.Has(DataValueSourcePicoseconds) {
		buf.WriteUint16(d.SourcePicoseconds)
	}
	if d.Has(DataValueServerTimestamp) {
		buf.WriteTime(d.ServerTimestamp)
	}
	if d.Has(DataValueServerPicoseconds) {
		buf.WriteUint16(d.ServerPicoseconds)
	}
	return buf.Bytes(), buf.Error()
}

func (d *DataValue) Has(mask byte) bool {
	return d.EncodingMask&mask == mask
}

func (d *DataValue) UpdateMask() {
	d.EncodingMask = 0
	if d.Value != nil {
		d.EncodingMask |= DataValueValue
	}
	if d.Status != 0 {
		d.EncodingMask |= DataValueStatus
	}
	if !d.SourceTimestamp.IsZero() {
		d.EncodingMask |= DataValueSourceTimestamp
	}
	if !d.ServerTimestamp.IsZero() {
		d.EncodingMask |= DataValueServerTimestamp
	}
	if d.SourcePicoseconds > 0 {
		d.EncodingMask |= DataValueSourcePicoseconds
	}
	if d.ServerPicoseconds > 0 {
		d.EncodingMask |= DataValueServerPicoseconds
	}
}
