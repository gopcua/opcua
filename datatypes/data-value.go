// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"time"

	"github.com/gopcua/opcua/ua"
)

// DataValue is always preceded by a mask that indicates which fields are present in the stream.
//
// Specification: Part 6, 5.2.2.17
type DataValue struct {
	EncodingMask      byte
	Value             *Variant
	Status            uint32
	SourceTimestamp   time.Time
	SourcePicoSeconds uint16
	ServerTimestamp   time.Time
	ServerPicoSeconds uint16
}

// NewDataValue creates a new DataValue.
func NewDataValue(hasValue, hasStatus, hasSrcTs, hasSrcPs, hasSvrTs, hasSvrPs bool, v *Variant, status uint32, srcTs time.Time, srcPs uint16, svrTs time.Time, svrPs uint16) *DataValue {
	d := &DataValue{
		Value:             v,
		Status:            status,
		SourceTimestamp:   srcTs,
		SourcePicoSeconds: srcPs,
		ServerTimestamp:   svrTs,
		ServerPicoSeconds: svrPs,
	}

	if hasValue {
		d.SetValueFlag()
	}
	if hasStatus {
		d.SetStatusFlag()
	}
	if hasSrcTs {
		d.SetSourceTimestampFlag()
	}
	if hasSrcPs {
		d.SetSourcePicoSecondsFlag()
	}
	if hasSvrTs {
		d.SetServerTimestampFlag()
	}
	if hasSvrPs {
		d.SetServerPicoSecondsFlag()
	}
	return d
}

func (d *DataValue) Decode(b []byte) (int, error) {
	buf := ua.NewBuffer(b)
	d.EncodingMask = buf.ReadByte()
	if d.HasValue() {
		d.Value = new(Variant)
		buf.ReadStruct(d.Value)
	}
	if d.HasStatus() {
		d.Status = buf.ReadUint32()
	}
	if d.HasSourceTimestamp() {
		d.SourceTimestamp = buf.ReadTime()
	}
	if d.HasSourcePicoSeconds() {
		d.SourcePicoSeconds = buf.ReadUint16()
	}
	if d.HasServerTimestamp() {
		d.ServerTimestamp = buf.ReadTime()
	}
	if d.HasServerPicoSeconds() {
		d.ServerPicoSeconds = buf.ReadUint16()
	}
	return buf.Pos(), buf.Error()
}

func (d *DataValue) Encode() ([]byte, error) {
	buf := ua.NewBuffer(nil)
	buf.WriteUint8(d.EncodingMask)
	if d.HasValue() {
		buf.WriteStruct(d.Value)
	}
	if d.HasStatus() {
		buf.WriteUint32(d.Status)
	}
	if d.HasSourceTimestamp() {
		buf.WriteTime(d.SourceTimestamp)
	}
	if d.HasSourcePicoSeconds() {
		buf.WriteUint16(d.SourcePicoSeconds)
	}
	if d.HasServerTimestamp() {
		buf.WriteTime(d.ServerTimestamp)
	}
	if d.HasServerPicoSeconds() {
		buf.WriteUint16(d.ServerPicoSeconds)
	}
	return buf.Bytes(), buf.Error()
}

// HasValue checks if DataValue has Value or not.
func (d *DataValue) HasValue() bool {
	return d.EncodingMask&0x1 == 1
}

// SetValueFlag sets value flag in EncodingMask in DataValue.
func (d *DataValue) SetValueFlag() {
	d.EncodingMask |= 0x1
}

// HasStatus checks if DataValue has Status or not.
func (d *DataValue) HasStatus() bool {
	return (d.EncodingMask>>1)&0x1 == 1
}

// SetStatusFlag sets status flag in EncodingMask in DataValue.
func (d *DataValue) SetStatusFlag() {
	d.EncodingMask |= 0x2
}

// HasSourceTimestamp checks if DataValue has SourceTimestamp or not.
func (d *DataValue) HasSourceTimestamp() bool {
	return (d.EncodingMask>>2)&0x1 == 1
}

// SetSourceTimestampFlag sets source timestamp flag in EncodingMask in DataValue.
func (d *DataValue) SetSourceTimestampFlag() {
	d.EncodingMask |= 0x4
}

// HasServerTimestamp checks if DataValue has ServerTimestamp or not.
func (d *DataValue) HasServerTimestamp() bool {
	return (d.EncodingMask>>3)&0x1 == 1
}

// SetServerTimestampFlag sets server timestamp flag in EncodingMask in DataValue.
func (d *DataValue) SetServerTimestampFlag() {
	d.EncodingMask |= 0x8
}

// HasSourcePicoSeconds checks if DataValue has SourcePicoSeconds or not.
func (d *DataValue) HasSourcePicoSeconds() bool {
	return (d.EncodingMask>>4)&0x1 == 1
}

// SetSourcePicoSecondsFlag sets source picoseconds flag in EncodingMask in DataValue.
func (d *DataValue) SetSourcePicoSecondsFlag() {
	d.EncodingMask |= 0x10
}

// HasServerPicoSeconds checks if DataValue has ServerPicoSeconds or not.
func (d *DataValue) HasServerPicoSeconds() bool {
	return (d.EncodingMask>>5)&0x1 == 1
}

// SetServerPicoSecondsFlag sets server picoseconds flag in EncodingMask in DataValue.
func (d *DataValue) SetServerPicoSecondsFlag() {
	d.EncodingMask |= 0x20
}
