// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/datatypes"
)

// ApplicationType definitions.
//
// Part 4
// 7.1
const (
	AppTypeServer uint32 = iota
	AppTypeClient
	AppTypeClientAndServer
	AppTypeDiscoveryServer
)

// ApplicationDescription represents an ApplicationDescription.
//
// Part 4
// 7.1
type ApplicationDescription struct {
	ApplicationURI      *datatypes.String
	ProductURI          *datatypes.String
	ApplicationName     *datatypes.LocalizedText
	ApplicationType     uint32
	GatewayServerURI    *datatypes.String
	DiscoveryProfileURI *datatypes.String
	DiscoveryURIs       *datatypes.StringArray
}

// NewApplicationDescription creates a new NewApplicationDescription.
func NewApplicationDescription(appURI, prodURI, appName string, appType uint32, gwURI, profileURI string, discovURIs []string) *ApplicationDescription {
	return &ApplicationDescription{
		ApplicationURI:      datatypes.NewString(appURI),
		ProductURI:          datatypes.NewString(prodURI),
		ApplicationName:     datatypes.NewLocalizedText("", appName),
		ApplicationType:     appType,
		GatewayServerURI:    datatypes.NewString(gwURI),
		DiscoveryProfileURI: datatypes.NewString(prodURI),
		DiscoveryURIs:       datatypes.NewStringArray(discovURIs),
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
	a.ApplicationURI = &datatypes.String{}
	if err := a.ApplicationURI.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.ApplicationURI.Len()

	a.ProductURI = &datatypes.String{}
	if err := a.ProductURI.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.ProductURI.Len()

	a.ApplicationName = &datatypes.LocalizedText{}
	if err := a.ApplicationName.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.ApplicationName.Len()

	a.ApplicationType = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	a.GatewayServerURI = &datatypes.String{}
	if err := a.GatewayServerURI.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.GatewayServerURI.Len()

	a.DiscoveryProfileURI = &datatypes.String{}
	if err := a.DiscoveryProfileURI.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.DiscoveryProfileURI.Len()

	a.DiscoveryURIs = &datatypes.StringArray{}
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
