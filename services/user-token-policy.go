// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"fmt"
)

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
type UserTokenPolicy struct {
	PolicyID          string
	TokenType         uint32
	IssuedTokenType   string
	IssuerEndpointURI string
	SecurityPolicyURI string
}

// NewUserTokenPolicy creates a new NewUserTokenPolicy.
func NewUserTokenPolicy(id string, tokenType uint32, issuedToken, issuerURI, secURI string) *UserTokenPolicy {
	return &UserTokenPolicy{
		PolicyID:          id,
		TokenType:         tokenType,
		IssuedTokenType:   issuedToken,
		IssuerEndpointURI: issuerURI,
		SecurityPolicyURI: secURI,
	}
}

//
func (u *UserTokenPolicy) String() string {
	return fmt.Sprintf("%s, %#x, %s, %s, %s",
		u.PolicyID,
		u.TokenType,
		u.IssuedTokenType,
		u.IssuerEndpointURI,
		u.SecurityPolicyURI,
	)
}
