// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"fmt"

	"github.com/wmnsk/gopcua/id"
)

// LocalizedText represents a LocalizedText.
// A LocalizedText structure contains two fields that could be missing.
// For that reason, the encoding uses a bit mask to indicate which fields
// are actually present in the encoded form.
//
// Specification: Part 6, 5.2.2.14
type LocalizedText struct {
	EncodingMask uint8
	Locale       *String
	Text         *String
}

// NewLocalizedText creates a new NewLocalizedText.
func NewLocalizedText(locale, text string) *LocalizedText {
	var l = &LocalizedText{}
	if locale != "" {
		l.Locale = NewString(locale)
		l.SetLocaleMask()
	}
	if text != "" {
		l.Text = NewString(text)
		l.SetTextMask()
	}

	return l
}

// DecodeLocalizedText decodes given bytes into LocalizedText.
func DecodeLocalizedText(b []byte) (*LocalizedText, error) {
	l := &LocalizedText{}
	if err := l.DecodeFromBytes(b); err != nil {
		return nil, err
	}
	return l, nil
}

// DecodeFromBytes decodes given bytes into LocalizedText.
func (l *LocalizedText) DecodeFromBytes(b []byte) error {
	l.EncodingMask = b[0]

	var offset = 1
	if l.HasLocale() {
		l.Locale = &String{}
		if err := l.Locale.DecodeFromBytes(b[offset:]); err != nil {
			return err
		}
		offset += l.Locale.Len()
	}

	if l.HasText() {
		l.Text = &String{}
		if err := l.Text.DecodeFromBytes(b[offset:]); err != nil {
			return err
		}
		offset += l.Text.Len()
	}

	return nil
}

// Serialize serializes LocalizedText into bytes.
func (l *LocalizedText) Serialize() ([]byte, error) {
	b := make([]byte, l.Len())
	if err := l.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes LocalizedText into bytes.
func (l *LocalizedText) SerializeTo(b []byte) error {
	b[0] = l.EncodingMask

	var offset = 1
	if l.Locale != nil {
		if err := l.Locale.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += l.Locale.Len()
	}

	if l.Text != nil {
		if err := l.Text.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += l.Text.Len()
	}

	return nil
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

// Len returns the actual length of LocalizedText in int.
func (l *LocalizedText) Len() int {
	var ll = 1
	if l.Locale != nil {
		ll += l.Locale.Len()
	}
	if l.Text != nil {
		ll += l.Text.Len()
	}

	return ll
}

// String returns LocalizedText in string.
func (l *LocalizedText) String() string {
	return fmt.Sprintf("%x, %s, %s", l.EncodingMask, l.Locale, l.Text)
}

// DataType returns type of Data.
func (l *LocalizedText) DataType() uint16 {
	return id.LocalizedText
}
