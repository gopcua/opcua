// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// UserIdentityToken structure used in the Server Service Set allows Clients to specify the
// identity of the user they are acting on behalf of. The exact mechanism used to identify users
// depends on the system configuration.
//
// Specification: Part 4, 7.36.1
const (
	UserTokenAnonymous uint32 = iota
	UserTokenUsername
	UserTokenCertificate
	UserTokenIssuedToken
)

// UserTokenPolicy represents an UserTokenPolicy.
//
// Specification: Part 4, 7.37
// type UserTokenPolicy struct {
// 	PolicyID          string
// 	TokenType         uint32
// 	IssuedTokenType   string
// 	IssuerEndpointURI string
// 	SecurityPolicyURI string
// }

// NewUserTokenPolicy creates a new NewUserTokenPolicy.
// func NewUserTokenPolicy(id string, tokenType UserTokenType, issuedToken, issuerURL, secURI string) *UserTokenPolicy {
// 	return &UserTokenPolicy{
// 		PolicyID:          id,
// 		TokenType:         tokenType,
// 		IssuedTokenType:   issuedToken,
// 		IssuerEndpointURL: issuerURL,
// 		SecurityPolicyURI: secURI,
// 	}
// }

// //
// func (u *UserTokenPolicy) String() string {
// 	return fmt.Sprintf("%s, %#x, %s, %s, %s",
// 		u.PolicyID,
// 		u.TokenType,
// 		u.IssuedTokenType,
// 		u.IssuerEndpointURL,
// 		u.SecurityPolicyURI,
// 	)
// }
