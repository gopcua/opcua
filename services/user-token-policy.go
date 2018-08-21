// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/binary"
	"fmt"

	"github.com/wmnsk/gopcua/datatypes"
)

// UserIdentityToken definitions.
const (
	UserTokenAnonymous uint32 = iota
	UserTokenUsername
	UserTokenCertificate
	UserTokenIssuedToken
)

// UserTokenPolicy represents an UserTokenPolicy.
type UserTokenPolicy struct {
	PolicyID          *datatypes.String
	TokenType         uint32
	IssuedTokenType   *datatypes.String
	IssuerEndpointURI *datatypes.String
	SecurityPolicyURI *datatypes.String
}

// NewUserTokenPolicy creates a new NewUserTokenPolicy.
func NewUserTokenPolicy(id string, tokenType uint32, issuedToken, issuerURI, secURI string) *UserTokenPolicy {
	return &UserTokenPolicy{
		PolicyID:          datatypes.NewString(id),
		TokenType:         tokenType,
		IssuedTokenType:   datatypes.NewString(issuedToken),
		IssuerEndpointURI: datatypes.NewString(issuerURI),
		SecurityPolicyURI: datatypes.NewString(secURI),
	}
}

// DecodeUserTokenPolicy decodes given bytes into UserTokenPolicy.
func DecodeUserTokenPolicy(b []byte) (*UserTokenPolicy, error) {
	u := &UserTokenPolicy{}
	if err := u.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return u, nil
}

// DecodeFromBytes decodes given bytes into UserTokenPolicy.
func (u *UserTokenPolicy) DecodeFromBytes(b []byte) error {
	var offset = 0
	u.PolicyID = &datatypes.String{}
	if err := u.PolicyID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += u.PolicyID.Len()

	u.TokenType = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	u.IssuedTokenType = &datatypes.String{}
	if err := u.IssuedTokenType.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += u.IssuedTokenType.Len()

	u.IssuerEndpointURI = &datatypes.String{}
	if err := u.IssuerEndpointURI.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += u.IssuerEndpointURI.Len()

	u.SecurityPolicyURI = &datatypes.String{}
	if err := u.SecurityPolicyURI.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += u.SecurityPolicyURI.Len()

	return nil
}

// Serialize serializes UserTokenPolicy into bytes.
func (u *UserTokenPolicy) Serialize() ([]byte, error) {
	b := make([]byte, u.Len())
	if err := u.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes UserTokenPolicy into bytes.
func (u *UserTokenPolicy) SerializeTo(b []byte) error {
	var offset = 0
	if u.PolicyID != nil {
		if err := u.PolicyID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += u.PolicyID.Len()
	}

	binary.LittleEndian.PutUint32(b[offset:offset+4], u.TokenType)
	offset += 4

	if u.IssuedTokenType != nil {
		if err := u.IssuedTokenType.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += u.IssuedTokenType.Len()
	}

	if u.IssuerEndpointURI != nil {
		if err := u.IssuerEndpointURI.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += u.IssuerEndpointURI.Len()
	}

	if u.SecurityPolicyURI != nil {
		if err := u.SecurityPolicyURI.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += u.SecurityPolicyURI.Len()
	}

	return nil
}

// Len returns the actual length of UserTokenPolicy in int.
func (u *UserTokenPolicy) Len() int {
	var l = 4
	if u.PolicyID != nil {
		l += u.PolicyID.Len()
	}

	if u.IssuedTokenType != nil {
		l += u.IssuedTokenType.Len()
	}

	if u.IssuerEndpointURI != nil {
		l += u.IssuerEndpointURI.Len()
	}

	if u.SecurityPolicyURI != nil {
		l += u.SecurityPolicyURI.Len()
	}

	return l
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

// UserTokenPolicyArray represents an UserTokenPolicyArray.
type UserTokenPolicyArray struct {
	ArraySize         int32
	UserTokenPolicies []*UserTokenPolicy
}

// NewUserTokenPolicyArray creates a new NewUserTokenPolicyArray.
func NewUserTokenPolicyArray(uts []*UserTokenPolicy) *UserTokenPolicyArray {
	if uts == nil {
		u := &UserTokenPolicyArray{
			ArraySize: 0,
		}
		return u
	}

	return &UserTokenPolicyArray{
		ArraySize:         int32(len(uts)),
		UserTokenPolicies: uts,
	}
}

// DecodeUserTokenPolicyArray decodes given bytes into UserTokenPolicyArray.
func DecodeUserTokenPolicyArray(b []byte) (*UserTokenPolicyArray, error) {
	u := &UserTokenPolicyArray{}
	if err := u.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return u, nil
}

// DecodeFromBytes decodes given bytes into UserTokenPolicyArray.
func (u *UserTokenPolicyArray) DecodeFromBytes(b []byte) error {
	u.ArraySize = int32(binary.LittleEndian.Uint32(b[:4]))
	if u.ArraySize <= 0 {
		return nil
	}

	var offset = 4
	for i := 0; i < int(u.ArraySize); i++ {
		ut, err := DecodeUserTokenPolicy(b[offset:])
		if err != nil {
			return err
		}
		u.UserTokenPolicies = append(u.UserTokenPolicies, ut)
		offset += ut.Len()
	}

	return nil
}

// Serialize serializes UserTokenPolicyArray into bytes.
func (u *UserTokenPolicyArray) Serialize() ([]byte, error) {
	b := make([]byte, u.Len())
	if err := u.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes UserTokenPolicyArray into bytes.
func (u *UserTokenPolicyArray) SerializeTo(b []byte) error {
	binary.LittleEndian.PutUint32(b[:4], uint32(u.ArraySize))

	var offset = 4
	for _, ut := range u.UserTokenPolicies {
		if err := ut.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += ut.Len()
	}

	return nil
}

// Len returns the actual length of UserTokenPolicyArray in int.
func (u *UserTokenPolicyArray) Len() int {
	var l = 4

	for _, ut := range u.UserTokenPolicies {
		l += ut.Len()
	}
	return l
}
