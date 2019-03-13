// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"fmt"

	"github.com/wmnsk/gopcua/id"
	"github.com/wmnsk/gopcua/ua"
)

// LocalizedText represents a LocalizedText.
// A LocalizedText structure contains two fields that could be missing.
// For that reason, the encoding uses a bit mask to indicate which fields
// are actually present in the encoded form.
//
// Specification: Part 6, 5.2.2.14
type LocalizedText struct {
	EncodingMask uint8
	Locale       string
	Text         string
}

// NewLocalizedText creates a new NewLocalizedText.
func NewLocalizedText(locale, text string) *LocalizedText {
	l := &LocalizedText{
		Locale: locale,
		Text:   text,
	}
	if locale != "" {
		l.SetLocaleMask()
	}
	if text != "" {
		l.SetTextMask()
	}
	return l
}

func (m *LocalizedText) Decode(b []byte) (int, error) {
	buf := ua.NewBuffer(b)
	m.EncodingMask = buf.ReadByte()
	if m.HasLocale() {
		m.Locale = buf.ReadString()
	}
	if m.HasText() {
		m.Text = buf.ReadString()
	}
	return buf.Pos(), buf.Error()
}

func (m *LocalizedText) Encode() ([]byte, error) {
	buf := ua.NewBuffer(nil)
	buf.WriteUint8(m.EncodingMask)
	if m.HasLocale() {
		buf.WriteString(m.Locale)
	}
	if m.HasText() {
		buf.WriteString(m.Text)
	}
	return buf.Bytes(), buf.Error()
}

// HasLocale checks if the LocalizedText has HasLocale mask in EncodingMask.
func (l *LocalizedText) HasLocale() bool {
	return l.EncodingMask&0x1 == 1
}

// SetLocaleMask sets the HasLocale mask in EncodingMask.
func (l *LocalizedText) SetLocaleMask() {
	l.EncodingMask |= 0x1
}

// HasText checks if the LocalizedText has HasText mask in EncodingMask.
func (l *LocalizedText) HasText() bool {
	return (l.EncodingMask>>1)&0x1 == 1
}

// SetTextMask sets the HasText mask in EncodingMask.
func (l *LocalizedText) SetTextMask() {
	l.EncodingMask |= 0x2
}

// String returns LocalizedText in string.
func (l *LocalizedText) String() string {
	return fmt.Sprintf("%x, %s, %s", l.EncodingMask, l.Locale, l.Text)
}

// DataType returns type of Data.
func (l *LocalizedText) DataType() uint16 {
	return id.LocalizedText
}
