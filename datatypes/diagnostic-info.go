// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"fmt"

	"github.com/wmnsk/gopcua"
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
	AdditionalInfo      string
	InnerStatusCode     uint32
	InnerDiagnosticInfo *DiagnosticInfo
}

// NewDiagnosticInfo creates a new DiagnosticInfo.
func NewDiagnosticInfo(hasSymID, hasURI, hasText, hasLocale, hasInfo, hasInnerStatus, hasInnerDiag bool, symID, uri, locale, text int32, info string, code uint32, diag *DiagnosticInfo) *DiagnosticInfo {
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
	return new(DiagnosticInfo)
}

func (d *DiagnosticInfo) Decode(b []byte) (int, error) {
	buf := gopcua.NewBuffer(b)
	d.EncodingMask = buf.ReadByte()
	if d.HasSymbolicID() {
		d.SymbolicID = buf.ReadInt32()
	}
	if d.HasNamespaceURI() {
		d.NamespaceURI = buf.ReadInt32()
	}
	if d.HasLocale() {
		d.Locale = buf.ReadInt32()
	}
	if d.HasLocalizedText() {
		d.LocalizedText = buf.ReadInt32()
	}
	if d.HasAdditionalInfo() {
		d.AdditionalInfo = buf.ReadString()
	}
	if d.HasInnerStatusCode() {
		d.InnerStatusCode = buf.ReadUint32()
	}
	if d.HasInnerDiagnosticInfo() {
		d.InnerDiagnosticInfo = new(DiagnosticInfo)
		buf.ReadStruct(d.InnerDiagnosticInfo)
	}
	return buf.Pos(), buf.Error()
}

func (d *DiagnosticInfo) Encode() ([]byte, error) {
	buf := gopcua.NewBuffer(nil)
	buf.WriteByte(d.EncodingMask)
	if d.HasSymbolicID() {
		buf.WriteInt32(d.SymbolicID)
	}
	if d.HasNamespaceURI() {
		buf.WriteInt32(d.NamespaceURI)
	}
	if d.HasLocale() {
		buf.WriteInt32(d.Locale)
	}
	if d.HasLocalizedText() {
		buf.WriteInt32(d.LocalizedText)
	}
	if d.HasAdditionalInfo() {
		buf.WriteString(d.AdditionalInfo)
	}
	if d.HasInnerStatusCode() {
		buf.WriteUint32(d.InnerStatusCode)
	}
	if d.HasInnerDiagnosticInfo() {
		buf.WriteStruct(d.InnerDiagnosticInfo)
	}
	return buf.Bytes(), buf.Error()
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
		str = append(str, d.AdditionalInfo)
	}
	if d.HasInnerStatusCode() {
		str = append(str, fmt.Sprintf("%d", d.InnerStatusCode))
	}
	if d.HasInnerDiagnosticInfo() {
		str = append(str, d.InnerDiagnosticInfo.String())
	}
	return fmt.Sprintf("%v", str)
}
