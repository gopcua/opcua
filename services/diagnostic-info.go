// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/datatypes"
)

// DiagnosticInfo represents the DiagnosticInfo.
//
// Specification: Part 4, 7.8
type DiagnosticInfo struct {
	EncodingMask        uint8
	SymbolicID          int32
	NamespaceURI        int32
	Locale              int32
	LocalizedText       int32
	AdditionalInfo      *datatypes.String
	InnerStatusCode     uint32
	InnerDiagnosticInfo *DiagnosticInfo
}

// NewDiagnosticInfo creates a new DiagnosticInfo.
func NewDiagnosticInfo(hasSymID, hasURI, hasText, hasLocale, hasInfo, hasInnerStatus, hasInnerDiag bool, symID, uri, locale, text int32, info *datatypes.String, code uint32, diag *DiagnosticInfo) *DiagnosticInfo {
	d := &DiagnosticInfo{
		SymbolicID:          symID,
		NamespaceURI:        uri,
		Locale:              locale,
		LocalizedText:       text,
		AdditionalInfo:      info,
		InnerStatusCode:     code,
		InnerDiagnosticInfo: diag,
	}

	if hasSymID {
		d.SetSymbolicIDFlag()
	}
	if hasURI {
		d.SetNamespaceURIFlag()
	}
	if hasText {
		d.SetLocalizedTextFlag()
	}
	if hasLocale {
		d.SetLocaleFlag()
	}
	if hasInfo {
		d.SetAdditionalInfoFlag()
	}
	if hasInnerStatus {
		d.SetInnerStatusCodeFlag()
	}
	if hasInnerDiag {
		d.SetInnerDiagnosticInfoFlag()
	}

	return d
}

// NewNullDiagnosticInfo creates a DiagnosticInfo without any info.
func NewNullDiagnosticInfo() *DiagnosticInfo {
	return NewDiagnosticInfo(
		false, false, false, false, false, false, false,
		0, 0, 0, 0, nil, 0, nil,
	)
}

// DecodeDiagnosticInfo decodes given bytes into DiagnosticInfo.
func DecodeDiagnosticInfo(b []byte) (*DiagnosticInfo, error) {
	d := &DiagnosticInfo{}
	if err := d.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return d, nil
}

// DecodeFromBytes decodes given bytes into DiagnosticInfo.
func (d *DiagnosticInfo) DecodeFromBytes(b []byte) error {
	var offset = 1
	d.EncodingMask = b[0]
	if d.HasSymbolicID() {
		d.SymbolicID = int32(binary.LittleEndian.Uint32(b[offset : offset+4]))
		offset += 4
	}
	if d.HasNamespaceURI() {
		d.NamespaceURI = int32(binary.LittleEndian.Uint32(b[offset : offset+4]))
		offset += 4
	}
	if d.HasLocale() {
		d.Locale = int32(binary.LittleEndian.Uint32(b[offset : offset+4]))
		offset += 4
	}
	if d.HasLocalizedText() {
		d.LocalizedText = int32(binary.LittleEndian.Uint32(b[offset : offset+4]))
		offset += 4
	}
	if d.HasAdditionalInfo() {
		d.AdditionalInfo = &datatypes.String{}
		if err := d.AdditionalInfo.DecodeFromBytes(b[offset:]); err != nil {
			return err
		}
		offset += d.AdditionalInfo.Len()
	}
	if d.HasInnerStatusCode() {
		d.InnerStatusCode = binary.LittleEndian.Uint32(b[offset : offset+4])
		offset += 4
	}
	if d.HasInnerDiagnosticInfo() {
		d.InnerDiagnosticInfo = &DiagnosticInfo{}
		if err := d.InnerDiagnosticInfo.DecodeFromBytes(b[offset:]); err != nil {
			return err
		}
		offset += d.InnerDiagnosticInfo.Len()
	}

	return nil
}

// Serialize serializes DiagnosticInfo into bytes.
func (d *DiagnosticInfo) Serialize() ([]byte, error) {
	b := make([]byte, d.Len())
	if err := d.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes DiagnosticInfo into bytes.
func (d *DiagnosticInfo) SerializeTo(b []byte) error {
	var offset = 1
	b[0] = d.EncodingMask
	if d.HasSymbolicID() {
		binary.LittleEndian.PutUint32(b[offset:offset+4], uint32(d.SymbolicID))
		offset += 4
	}
	if d.HasNamespaceURI() {
		binary.LittleEndian.PutUint32(b[offset:offset+4], uint32(d.NamespaceURI))
		offset += 4
	}
	if d.HasLocale() {
		binary.LittleEndian.PutUint32(b[offset:offset+4], uint32(d.Locale))
		offset += 4
	}
	if d.HasLocalizedText() {
		binary.LittleEndian.PutUint32(b[offset:offset+4], uint32(d.LocalizedText))
		offset += 4
	}
	if d.HasAdditionalInfo() {
		if err := d.AdditionalInfo.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += d.AdditionalInfo.Len()
	}
	if d.HasInnerStatusCode() {
		binary.LittleEndian.PutUint32(b[offset:offset+4], uint32(d.InnerStatusCode))
		offset += 4
	}
	if d.HasInnerDiagnosticInfo() {
		if err := d.InnerDiagnosticInfo.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += d.InnerDiagnosticInfo.Len()
	}

	return nil
}

// Len returns the actual length of DiagnosticInfo in int.
func (d *DiagnosticInfo) Len() int {
	l := 1
	if d.HasSymbolicID() {
		l += 4
	}
	if d.HasNamespaceURI() {
		l += 4
	}
	if d.HasLocalizedText() {
		l += 4
	}
	if d.HasLocale() {
		l += 4
	}
	if d.HasAdditionalInfo() {
		if d.AdditionalInfo != nil {
			l += d.AdditionalInfo.Len()
		}
	}
	if d.HasInnerStatusCode() {
		l += 4
	}
	if d.HasInnerDiagnosticInfo() {
		if d.InnerDiagnosticInfo != nil {
			l += d.InnerDiagnosticInfo.Len()
		}
	}

	return l
}

// HasSymbolicID checks if DiagnosticInfo has SymbolicID or not.
func (d *DiagnosticInfo) HasSymbolicID() bool {
	return d.EncodingMask&0x1 == 1
}

// SetSymbolicIDFlag sets SymbolicIDFlag in EncodingMask in DiagnosticInfo.
func (d *DiagnosticInfo) SetSymbolicIDFlag() {
	d.EncodingMask |= 0x1
}

// HasNamespaceURI checks if DiagnosticInfo has NamespaceURI or not.
func (d *DiagnosticInfo) HasNamespaceURI() bool {
	return (d.EncodingMask>>1)&0x1 == 1
}

// SetNamespaceURIFlag sets NamespaceURIFlag in EncodingMask in DiagnosticInfo.
func (d *DiagnosticInfo) SetNamespaceURIFlag() {
	d.EncodingMask |= 0x2
}

// HasLocalizedText checks if DiagnosticInfo has LocalizedText or not.
func (d *DiagnosticInfo) HasLocalizedText() bool {
	return (d.EncodingMask>>2)&0x1 == 1
}

// SetLocalizedTextFlag sets LocalizedTextFlag in EncodingMask in DiagnosticInfo.
func (d *DiagnosticInfo) SetLocalizedTextFlag() {
	d.EncodingMask |= 0x4
}

// HasLocale checks if DiagnosticInfo has Locale or not.
func (d *DiagnosticInfo) HasLocale() bool {
	return (d.EncodingMask>>3)&0x1 == 1
}

// SetLocaleFlag sets LocaleFlag in EncodingMask in DiagnosticInfo.
func (d *DiagnosticInfo) SetLocaleFlag() {
	d.EncodingMask |= 0x8
}

// HasAdditionalInfo checks if DiagnosticInfo has AdditionalInfo or not.
func (d *DiagnosticInfo) HasAdditionalInfo() bool {
	return (d.EncodingMask>>4)&0x1 == 1
}

// SetAdditionalInfoFlag sets AdditionalInfoFlag in EncodingMask in DiagnosticInfo.
func (d *DiagnosticInfo) SetAdditionalInfoFlag() {
	d.EncodingMask |= 0x10
}

// HasInnerStatusCode checks if DiagnosticInfo has InnerStatusCode or not.
func (d *DiagnosticInfo) HasInnerStatusCode() bool {
	return (d.EncodingMask>>5)&0x1 == 1
}

// SetInnerStatusCodeFlag sets InnerStatusCodeFlag in EncodingMask in DiagnosticInfo.
func (d *DiagnosticInfo) SetInnerStatusCodeFlag() {
	d.EncodingMask |= 0x20
}

// HasInnerDiagnosticInfo checks if DiagnosticInfo has InnerDiagnosticInfo or not.
func (d *DiagnosticInfo) HasInnerDiagnosticInfo() bool {
	return (d.EncodingMask>>6)&0x1 == 1
}

// SetInnerDiagnosticInfoFlag sets InnerDiagnosticInfoFlag in EncodingMask in DiagnosticInfo.
func (d *DiagnosticInfo) SetInnerDiagnosticInfoFlag() {
	d.EncodingMask |= 0x40
}

// datatypes.String returns DiagnosticInfo in string.
func (d *DiagnosticInfo) String() string {
	var str []string
	str = append(str, fmt.Sprintf("%x", d.EncodingMask))
	if d.HasSymbolicID() {
		str = append(str, fmt.Sprintf("%d", d.SymbolicID))
	}
	if d.HasNamespaceURI() {
		str = append(str, fmt.Sprintf("%d", d.NamespaceURI))
	}
	if d.HasLocale() {
		str = append(str, fmt.Sprintf("%d", d.Locale))
	}
	if d.HasLocalizedText() {
		str = append(str, fmt.Sprintf("%d", d.LocalizedText))
	}
	if d.HasAdditionalInfo() {
		str = append(str, fmt.Sprintf("%v", d.AdditionalInfo.Get()))
	}
	if d.HasInnerStatusCode() {
		str = append(str, fmt.Sprintf("%d", d.InnerStatusCode))
	}
	if d.HasInnerDiagnosticInfo() {
		str = append(str, fmt.Sprintf("%v", d.InnerDiagnosticInfo.String()))
	}

	return fmt.Sprintf("%v", str)
}
