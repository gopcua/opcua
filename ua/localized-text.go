// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// todo(fs): fix mask

// LocalizedText represents a LocalizedText.
// A LocalizedText structure contains two fields that could be missing.
// For that reason, the encoding uses a bit mask to indicate which fields
// are actually present in the encoded forl.
//
// Specification: Part 6, 5.2.2.14
type LocalizedText struct {
	Locale string
	Text   string
}

func (l *LocalizedText) Decode(b []byte) (int, error) {
	buf := NewBuffer(b)
	mask := buf.ReadByte()

	l.Locale = ""
	if mask&0x1 == 1 {
		l.Locale = buf.ReadString()
	}

	l.Text = ""
	if mask&0x2 == 2 {
		l.Text = buf.ReadString()
	}

	return buf.Pos(), buf.Error()
}

func (l *LocalizedText) Encode() ([]byte, error) {
	buf := NewBuffer(nil)

	var mask byte
	if l.Locale != "" {
		mask |= 0x1
	}
	if l.Text != "" {
		mask |= 0x2
	}
	buf.WriteUint8(mask)

	if l.Locale != "" {
		buf.WriteString(l.Locale)
	}
	if l.Text != "" {
		buf.WriteString(l.Text)
	}
	return buf.Bytes(), buf.Error()
}
