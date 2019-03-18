// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// EndpointDescription represents an EndpointDescription.
//
// Specification: Part 4, 7.10
// type EndpointDescription struct {
// 	EndpointURL         string
// 	Server              *ApplicationDescription
// 	ServerCertificate   []byte
// 	MessageSecurityMode uint32
// 	SecurityPolicyURI   string
// 	UserIdentityTokens  []*UserTokenPolicy
// 	TransportProfileURI string
// 	SecurityLevel       uint8
// }

// NewEndpointDescription creates a new NewEndpointDescription.
// func NewEndpointDescription(url string, server *ApplicationDescription, cert []byte, secMode MessageSecurityMode, secURI string, tokens []*UserTokenPolicy, transportURI string, secLevel uint8) *EndpointDescription {
// 	return &EndpointDescription{
// 		EndpointURL:         url,
// 		Server:              server,
// 		ServerCertificate:   cert,
// 		SecurityMode:        secMode,
// 		SecurityPolicyURI:   secURI,
// 		UserIdentityTokens:  tokens,
// 		TransportProfileURI: transportURI,
// 		SecurityLevel:       secLevel,
// 	}
// }

// // String returns EndpointDescription in string.
// func (e *EndpointDescription) String() string {
// 	return fmt.Sprintf("%s, %v, %x, %d, %s, %s, %d",
// 		e.EndpointURL,
// 		e.Server,
// 		e.ServerCertificate,
// 		e.SecurityMode,
// 		e.SecurityPolicyURI,
// 		e.TransportProfileURI,
// 		e.SecurityLevel,
// 	)
// }
