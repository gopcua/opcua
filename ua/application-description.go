// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"fmt"
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
	ApplicationURI      string
	ProductURI          string
	ApplicationName     *LocalizedText
	ApplicationType     uint32
	GatewayServerURI    string
	DiscoveryProfileURI string
	DiscoveryURIs       []string
}

// NewApplicationDescription creates a new NewApplicationDescription.
func NewApplicationDescription(appURI, prodURI, appName string, appType uint32, gwURI, profileURI string, discovURIs []string) *ApplicationDescription {
	return &ApplicationDescription{
		ApplicationURI:      appURI,
		ProductURI:          prodURI,
		ApplicationName:     NewLocalizedText("", appName),
		ApplicationType:     appType,
		GatewayServerURI:    gwURI,
		DiscoveryProfileURI: profileURI,
		DiscoveryURIs:       discovURIs,
	}
}

// String returns ApplicationDescription in string.
func (a *ApplicationDescription) String() string {
	return fmt.Sprintf("%s, %s, %s, %s, %s, %v",
		a.ApplicationURI,
		a.ProductURI,
		a.ApplicationName.Text,
		a.GatewayServerURI,
		a.DiscoveryProfileURI,
		a.DiscoveryURIs,
	)
}
