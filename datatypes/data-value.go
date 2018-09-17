// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/binary"
	"time"

	"github.com/wmnsk/gopcua/utils"
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
func NewDataValue(hasValue, hasSrcTs, hasSrcPs, hasSvrTs, hasServerPc bool, v *Variant, status uint32, srcTs time.Time, srcPs uint16, svrTs time.Time, svrPs uint16) *DataValue {
	d := &DataValue{
		Value:             v,
		Status:            status,
		SourceTimestamp:   srcTs,
		SourcePicoSeconds: srcPs,
		ServerTimestamp:   svrTs,
		ServerPicoSeconds: svrPs,
	}

	if d.HasValue() {
		d.SetValueFlag()
	}

	if d.HasStatus() {
		d.SetStatusFlag()
	}

	if d.HasSourceTimestamp() {
		d.SetSourceTimestampFlag()
	}

	if d.HasSourcePicoSeconds() {
		d.SetSourcePicoSecondsFlag()
	}

	if d.HasServerTimestamp() {
		d.SetServerTimestampFlag()
	}

	if d.HasServerPicoSeconds() {
		d.SetServerPicoSecondsFlag()
	}

	return d
}

// DecodeDataValue decodes given bytes into DataValue.
func DecodeDataValue(b []byte) (*DataValue, error) {
	d := &DataValue{}
	if err := d.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return d, nil
}

// DecodeFromBytes decodes given bytes into DataValue.
func (d *DataValue) DecodeFromBytes(b []byte) error {
	d.EncodingMask = b[0]

	offset := 1

	if d.HasValue() {
		d.Value = &Variant{}
		if err := d.Value.DecodeFromBytes(b[offset:]); err != nil {
			return err
		}
		offset += d.Value.Len()
	}

	if d.HasStatus() {
		d.Status = binary.LittleEndian.Uint32(b[offset : offset+4])
		offset += 4
	}

	if d.HasSourceTimestamp() {
		d.SourceTimestamp = utils.DecodeTimestamp(b[offset : offset+8])
		offset += 8
	}

	if d.HasSourcePicoSeconds() {
		d.SourcePicoSeconds = binary.LittleEndian.Uint16(b[offset : offset+2])
		offset += 2
	}

	if d.HasServerTimestamp() {
		d.ServerTimestamp = utils.DecodeTimestamp(b[offset : offset+8])
		offset += 8
	}

	if d.HasServerPicoSeconds() {
		d.ServerPicoSeconds = binary.LittleEndian.Uint16(b[offset : offset+2])
		offset += 2
	}

	return nil
}

// Serialize serializes DataValue into bytes.
func (d *DataValue) Serialize() ([]byte, error) {
	b := make([]byte, d.Len())
	if err := d.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes DataValue into bytes.
func (d *DataValue) SerializeTo(b []byte) error {
	b[0] = d.EncodingMask

	offset := 1
	if d.HasValue() {
		if err := d.Value.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += d.Value.Len()
	}

	if d.HasStatus() {
		binary.LittleEndian.PutUint32(b[offset:offset+4], d.Status)
		offset += 4
	}

	if d.HasSourceTimestamp() {
		utils.EncodeTimestamp(b[offset:offset+8], d.SourceTimestamp)
		offset += 8
	}

	if d.HasSourcePicoSeconds() {
		binary.LittleEndian.PutUint16(b[offset:offset+2], d.SourcePicoSeconds)
		offset += 2
	}

	if d.HasServerTimestamp() {
		utils.EncodeTimestamp(b[offset:offset+8], d.ServerTimestamp)
		offset += 8
	}

	if d.HasServerPicoSeconds() {
		binary.LittleEndian.PutUint16(b[offset:offset+2], d.ServerPicoSeconds)
		offset += 2
	}

	return nil
}

// Len returns the actual length of DataValue in int.
func (d *DataValue) Len() int {
	length := 1

	if d.HasValue() {
		if d.Value != nil {
			length += d.Value.Len()
		}
	}

	if d.HasStatus() {
		length += 4
	}

	if d.HasSourceTimestamp() {
		length += 8
	}

	if d.HasSourcePicoSeconds() {
		length += 2
	}

	if d.HasServerTimestamp() {
		length += 8
	}

	if d.HasServerPicoSeconds() {
		length += 2
	}

	return length
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
