// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"github.com/wmnsk/gopcua"
)

// QualifiedName contains a qualified name. It is, for example, used as BrowseName.
// The name part of the QualifiedName is restricted to 512 characters.
//
// Specification: Part 3, 8.3
type QualifiedName struct {
	NamespaceIndex uint16
	Name           string
}

// NewQualifiedName creates a new QualifiedName.
func NewQualifiedName(index uint16, name string) *QualifiedName {
	return &QualifiedName{
		NamespaceIndex: index,
		Name:           name,
	}
}

func (n *QualifiedName) Decode(b []byte) (int, error) {
	buf := gopcua.NewBuffer(b)
	n.NamespaceIndex = buf.ReadUint16()
	n.Name = buf.ReadString()
	return buf.Pos(), buf.Error()
}

func (n *QualifiedName) Encode() ([]byte, error) {
	buf := gopcua.NewBuffer(nil)
	buf.WriteUint16(n.NamespaceIndex)
	buf.WriteString(n.Name)
	return buf.Bytes(), buf.Error()
}
