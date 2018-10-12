// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/errors"
)

// ApplicationType definitions.
//
// Specification: Part 4, 7.1
const (
	AppTypeServer uint32 = iota
	AppTypeClient
	AppTypeClientAndServer
	AppTypeDiscoveryServer
)

// ApplicationDescription represents an ApplicationDescription.
//
// Specification: Part 4, 7.1
type ApplicationDescription struct {
	ApplicationURI      *String
	ProductURI          *String
	ApplicationName     *LocalizedText
	ApplicationType     uint32
	GatewayServerURI    *String
	DiscoveryProfileURI *String
	DiscoveryURIs       *StringArray
}

// NewApplicationDescription creates a new NewApplicationDescription.
func NewApplicationDescription(appURI, prodURI, appName string, appType uint32, gwURI, profileURI string, discovURIs []string) *ApplicationDescription {
	return &ApplicationDescription{
		ApplicationURI:      NewString(appURI),
		ProductURI:          NewString(prodURI),
		ApplicationName:     NewLocalizedText("", appName),
		ApplicationType:     appType,
		GatewayServerURI:    NewString(gwURI),
		DiscoveryProfileURI: NewString(profileURI),
		DiscoveryURIs:       NewStringArray(discovURIs),
	}
}

// DecodeApplicationDescription decodes given bytes into ApplicationDescription.
func DecodeApplicationDescription(b []byte) (*ApplicationDescription, error) {
	a := &ApplicationDescription{}
	if err := a.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return a, nil
}

// DecodeFromBytes decodes given bytes into ApplicationDescription.
func (a *ApplicationDescription) DecodeFromBytes(b []byte) error {
	var offset = 0
	a.ApplicationURI = &String{}
	if err := a.ApplicationURI.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.ApplicationURI.Len()

	a.ProductURI = &String{}
	if err := a.ProductURI.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.ProductURI.Len()

	a.ApplicationName = &LocalizedText{}
	if err := a.ApplicationName.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.ApplicationName.Len()

	a.ApplicationType = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	a.GatewayServerURI = &String{}
	if err := a.GatewayServerURI.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.GatewayServerURI.Len()

	a.DiscoveryProfileURI = &String{}
	if err := a.DiscoveryProfileURI.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.DiscoveryProfileURI.Len()

	a.DiscoveryURIs = &StringArray{}
	if err := a.DiscoveryURIs.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.DiscoveryURIs.Len()

	return nil
}

// Serialize serializes ApplicationDescription into bytes.
func (a *ApplicationDescription) Serialize() ([]byte, error) {
	b := make([]byte, a.Len())
	if err := a.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes ApplicationDescription into bytes.
func (a *ApplicationDescription) SerializeTo(b []byte) error {
	var offset = 0
	if a.ApplicationURI != nil {
		if err := a.ApplicationURI.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.ApplicationURI.Len()
	}

	if a.ProductURI != nil {
		if err := a.ProductURI.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.ProductURI.Len()
	}

	if a.ApplicationName != nil {
		if err := a.ApplicationName.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.ApplicationName.Len()
	}

	binary.LittleEndian.PutUint32(b[offset:offset+4], a.ApplicationType)
	offset += 4

	if a.GatewayServerURI != nil {
		if err := a.GatewayServerURI.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.GatewayServerURI.Len()
	}

	if a.DiscoveryProfileURI != nil {
		if err := a.DiscoveryProfileURI.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.DiscoveryProfileURI.Len()
	}

	if a.DiscoveryURIs != nil {
		if err := a.DiscoveryURIs.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.DiscoveryURIs.Len()
	}

	return nil
}

// Len returns the actual length of ApplicationDescription in int.
func (a *ApplicationDescription) Len() int {
	var l = 4
	if a.ApplicationURI != nil {
		l += a.ApplicationURI.Len()
	}
	if a.ProductURI != nil {
		l += a.ProductURI.Len()
	}
	if a.ApplicationName != nil {
		l += a.ApplicationName.Len()
	}
	if a.GatewayServerURI != nil {
		l += a.GatewayServerURI.Len()
	}
	if a.DiscoveryProfileURI != nil {
		l += a.DiscoveryProfileURI.Len()
	}
	if a.DiscoveryURIs != nil {
		l += a.DiscoveryURIs.Len()
	}

	return l
}

// String returns ApplicationDescription in string.
func (a *ApplicationDescription) String() string {
	return fmt.Sprintf("%s, %s, %s, %s, %s, %v",
		a.ApplicationURI.Get(),
		a.ProductURI.Get(),
		a.ApplicationName.Text,
		a.GatewayServerURI.Get(),
		a.DiscoveryProfileURI.Get(),
		a.DiscoveryURIs.Strings,
	)
}

// ApplicationDescriptionArray represents an ApplicationDescriptionArray.
type ApplicationDescriptionArray struct {
	ArraySize               int32
	ApplicationDescriptions []*ApplicationDescription
}

// NewApplicationDescriptionArray creates an NewApplicationDescriptionArray from multiple ApplicationDescription.
func NewApplicationDescriptionArray(descs []*ApplicationDescription) *ApplicationDescriptionArray {
	e := &ApplicationDescriptionArray{
		ArraySize: int32(len(descs)),
	}
	e.ApplicationDescriptions = append(e.ApplicationDescriptions, descs...)

	return e
}

// DecodeApplicationDescriptionArray decodes given bytes into ApplicationDescriptionArray.
func DecodeApplicationDescriptionArray(b []byte) (*ApplicationDescriptionArray, error) {
	e := &ApplicationDescriptionArray{}
	if err := e.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return e, nil
}

// DecodeFromBytes decodes given bytes into ApplicationDescriptionArray.
func (e *ApplicationDescriptionArray) DecodeFromBytes(b []byte) error {
	if len(b) < 4 {
		return errors.NewErrTooShortToDecode(e, "should be longer than 4 bytes.")
	}

	e.ArraySize = int32(binary.LittleEndian.Uint32(b[:4]))
	if e.ArraySize <= 0 {
		return nil
	}

	var offset = 4
	for i := 0; i < int(e.ArraySize); i++ {
		ed, err := DecodeApplicationDescription(b[offset:])
		if err != nil {
			return err
		}
		e.ApplicationDescriptions = append(e.ApplicationDescriptions, ed)
		offset += e.ApplicationDescriptions[i].Len()
	}

	return nil
}

// Serialize serializes ApplicationDescriptionArray into bytes.
func (e *ApplicationDescriptionArray) Serialize() ([]byte, error) {
	b := make([]byte, e.Len())
	if err := e.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes ApplicationDescriptionArray into bytes.
func (e *ApplicationDescriptionArray) SerializeTo(b []byte) error {
	binary.LittleEndian.PutUint32(b[:4], uint32(e.ArraySize))
	if e.ArraySize <= 0 {
		return nil
	}

	var offset = 4
	for _, ed := range e.ApplicationDescriptions {
		if err := ed.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += ed.Len()
	}

	return nil
}

// Len returns the actual length of ApplicationDescriptionArray in int.
func (e *ApplicationDescriptionArray) Len() int {
	var l = 4
	for _, ed := range e.ApplicationDescriptions {
		l += ed.Len()
	}

	return l
}
