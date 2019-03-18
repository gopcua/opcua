// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// ApplicationType definitions.
//
// Specification: Part 4, 7.1

// ApplicationDescription represents an ApplicationDescription.
//
// Specification: Part 4, 7.1
// type ApplicationDescription struct {
// 	ApplicationURI      string
// 	ProductURI          string
// 	ApplicationName     *LocalizedText
// 	ApplicationType     uint32
// 	GatewayServerURI    string
// 	DiscoveryProfileURI string
// 	DiscoveryURIs       []string
// }

// // NewApplicationDescription creates a new NewApplicationDescription.
// func NewApplicationDescription(appURI, prodURI, appName string, appType ApplicationType, gwURI, profileURI string, discovURLs []string) *ApplicationDescription {
// 	return &ApplicationDescription{
// 		ApplicationURI:      appURI,
// 		ProductURI:          prodURI,
// 		ApplicationName:     NewLocalizedText("", appName),
// 		ApplicationType:     appType,
// 		GatewayServerURI:    gwURI,
// 		DiscoveryProfileURI: profileURI,
// 		DiscoveryURLs:       discovURLs,
// 	}
// }

// // String returns ApplicationDescription in string.
// func (a *ApplicationDescription) String() string {
// 	return fmt.Sprintf("%s, %s, %s, %s, %s, %v",
// 		a.ApplicationURI,
// 		a.ProductURI,
// 		a.ApplicationName.Text,
// 		a.GatewayServerURI,
// 		a.DiscoveryProfileURI,
// 		a.DiscoveryURLs,
// 	)
// }
