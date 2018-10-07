// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"github.com/wmnsk/gopcua/id"
)

// AnonymousIdentityToken is used to indicate that the Client has no user credentials.
//
// Specification: Part4, 7.36.5
type AnonymousIdentityToken struct {
	PolicyID *String
}

// NewAnonymousIdentityToken creates a new AnonymousIdentityToken.
func NewAnonymousIdentityToken(policyID string) *AnonymousIdentityToken {
	return &AnonymousIdentityToken{
		PolicyID: NewString(policyID),
	}
}

// DecodeAnonymousIdentityToken decodes given bytes as AnonymousIdentityToken.
func DecodeAnonymousIdentityToken(b []byte) (*AnonymousIdentityToken, error) {
	a := &AnonymousIdentityToken{}
	if err := a.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return a, nil
}

// DecodeFromBytes decodes given bytes as AnonymousIdentityToken.
func (a *AnonymousIdentityToken) DecodeFromBytes(b []byte) error {
	a.PolicyID = &String{}
	return a.PolicyID.DecodeFromBytes(b)
}

// Serialize serializes AnonymousIdentityToken into bytes.
func (a *AnonymousIdentityToken) Serialize() ([]byte, error) {
	b := make([]byte, a.Len())
	if err := a.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes AnonymousIdentityToken into bytes.
func (a *AnonymousIdentityToken) SerializeTo(b []byte) error {
	if a.PolicyID != nil {
		if err := a.PolicyID.SerializeTo(b); err != nil {
			return err
		}
	}
	return nil
}

// Len returns the actual Length of AnonymousIdentityToken in int.
func (a *AnonymousIdentityToken) Len() int {
	l := 0
	if a.PolicyID != nil {
		l += a.PolicyID.Len()
	}

	return l
}

// Type returns PolicyID in int.
func (a *AnonymousIdentityToken) Type() int {
	return id.AnonymousIdentityToken_Encoding_DefaultBinary
}

// UserNameIdentityToken is used to pass simple username/password credentials to the Server.
//
// This token shall be encrypted by the Client if required by the SecurityPolicy of the
// UserTokenPolicy. The Server should specify a SecurityPolicy for the UserTokenPolicy if the
// SecureChannel has a SecurityPolicy of None and no transport layer encryption is available. If
// None is specified for the UserTokenPolicy and SecurityPolicy is None then the password only
// contains the UTF-8 encoded password. The SecurityPolicy of the SecureChannel is used if no
// SecurityPolicy is specified in the UserTokenPolicy.
//
// If the token is to be encrypted the password shall be converted to a UTF-8 ByteString, encrypted
// and then serialized as shown in Table 181.
// The Server shall decrypt the password and verify the ServerNonce.
//
// If the SecurityPolicy is None then the password only contains the UTF-8 encoded password. This
// configuration should not be used unless the network is encrypted in some other manner such as a
// VPN. The use of this configuration without network encryption would result in a serious security
// fault, in that it would cause the appearance of a secure user access, but it would make the
// password visible in clear text.
//
// Specification: Part4, 7.36.4
type UserNameIdentityToken struct {
	PolicyID            *String
	UserName            *String
	Password            *ByteString
	EncryptionAlgorithm *String
}

// NewUserNameIdentityToken creates a new UserNameIdentityToken.
func NewUserNameIdentityToken(policyID, username string, password []byte, alg string) *UserNameIdentityToken {
	return &UserNameIdentityToken{
		PolicyID:            NewString(policyID),
		UserName:            NewString(username),
		Password:            NewByteString(password),
		EncryptionAlgorithm: NewString(alg),
	}
}

// DecodeUserNameIdentityToken decodes given bytes as UserNameIdentityToken.
func DecodeUserNameIdentityToken(b []byte) (*UserNameIdentityToken, error) {
	u := &UserNameIdentityToken{}
	if err := u.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return u, nil
}

// DecodeFromBytes decodes given bytes as UserNameIdentityToken.
func (u *UserNameIdentityToken) DecodeFromBytes(b []byte) error {
	offset := 0
	u.PolicyID = &String{}
	if err := u.PolicyID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += u.PolicyID.Len()

	u.UserName = &String{}
	if err := u.UserName.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += u.UserName.Len()

	u.Password = &ByteString{}
	if err := u.Password.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += u.Password.Len()

	u.EncryptionAlgorithm = &String{}
	return u.EncryptionAlgorithm.DecodeFromBytes(b[offset:])
}

// Serialize serializes UserNameIdentityToken into bytes.
func (u *UserNameIdentityToken) Serialize() ([]byte, error) {
	b := make([]byte, u.Len())
	if err := u.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes UserNameIdentityToken into bytes.
func (u *UserNameIdentityToken) SerializeTo(b []byte) error {
	offset := 0
	if u.PolicyID != nil {
		if err := u.PolicyID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += u.PolicyID.Len()
	}

	if u.UserName != nil {
		if err := u.UserName.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += u.UserName.Len()
	}

	if u.Password != nil {
		if err := u.Password.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += u.Password.Len()
	}

	if u.EncryptionAlgorithm != nil {
		if err := u.EncryptionAlgorithm.SerializeTo(b[offset:]); err != nil {
			return err
		}
	}

	return nil
}

// Len returns the actual Length of UserNameIdentityToken in int.
func (u *UserNameIdentityToken) Len() int {
	l := 0
	if u.PolicyID != nil {
		l += u.PolicyID.Len()
	}

	if u.UserName != nil {
		l += u.UserName.Len()
	}

	if u.Password != nil {
		l += u.Password.Len()
	}

	if u.EncryptionAlgorithm != nil {
		l += u.EncryptionAlgorithm.Len()
	}

	return l
}

// Type returns PolicyID in int.
func (u *UserNameIdentityToken) Type() int {
	return id.UserNameIdentityToken_Encoding_DefaultBinary
}

// X509IdentityToken is used to pass an X.509 v3 Certificate which is issued by the user.
// This token shall always be accompanied by a Signature in the userTokenSignature parameter of
// ActivateSession if required by the SecurityPolicy. The Server should specify a SecurityPolicy for
// the UserTokenPolicy if the SecureChannel has a SecurityPolicy of None.
//
// Specification: Part4, 7.36.5
type X509IdentityToken struct {
	PolicyID        *String
	CertificateData *String
}

// NewX509IdentityToken creates a new X509IdentityToken.
func NewX509IdentityToken(policyID, cert string) *X509IdentityToken {
	return &X509IdentityToken{
		PolicyID:        NewString(policyID),
		CertificateData: NewString(cert),
	}
}

// DecodeX509IdentityToken decodes given bytes as X509IdentityToken.
func DecodeX509IdentityToken(b []byte) (*X509IdentityToken, error) {
	x := &X509IdentityToken{}
	if err := x.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return x, nil
}

// DecodeFromBytes decodes given bytes as X509IdentityToken.
func (x *X509IdentityToken) DecodeFromBytes(b []byte) error {
	offset := 0
	x.PolicyID = &String{}
	if err := x.PolicyID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += x.PolicyID.Len()

	x.CertificateData = &String{}
	return x.CertificateData.DecodeFromBytes(b[offset:])
}

// Serialize serializes X509IdentityToken into bytes.
func (x *X509IdentityToken) Serialize() ([]byte, error) {
	b := make([]byte, x.Len())
	if err := x.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes X509IdentityToken into bytes.
func (x *X509IdentityToken) SerializeTo(b []byte) error {
	offset := 0
	if x.PolicyID != nil {
		if err := x.PolicyID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += x.PolicyID.Len()
	}

	if x.CertificateData != nil {
		if err := x.CertificateData.SerializeTo(b[offset:]); err != nil {
			return err
		}
	}

	return nil
}

// Len returns the actual Length of X509IdentityToken in int.
func (x *X509IdentityToken) Len() int {
	l := 0
	if x.PolicyID != nil {
		l += x.PolicyID.Len()
	}

	if x.CertificateData != nil {
		l += x.CertificateData.Len()
	}

	return l
}

// Type returns PolicyID in int.
func (x *X509IdentityToken) Type() int {
	return id.X509IdentityToken_Encoding_DefaultBinary
}

// IssuedIdentityToken is used to pass SecurityTokens issued by an external Authorization
// Service to the Server. These tokens may be text or binary.
// OAuth2 defines a standard for Authorization Services that produce JSON Web Tokens (JWT).
// These JWTs are passed as an Issued Token to an OPC UA Server which uses the signature
// contained in the JWT to validate the token. Part 6 describes OAuth2 and JWTs in more detail. If
// the token is encrypted, it shall use the EncryptedSecret format defined in 7.36.2.3.
// This token shall be encrypted by the Client if required by the SecurityPolicy of the
// UserTokenPolicy. The Server should specify a SecurityPolicy for the UserTokenPolicy if the
// SecureChannel has a SecurityPolicy of None and no transport layer encryption is available. The
// SecurityPolicy of the SecureChannel is used If no SecurityPolicy is specified in the
// UserTokenPolicy.
// If the SecurityPolicy is not None, the tokenData shall be encoded in UTF-8 (if it is not already
// binary), signed and encrypted according the rules specified for the tokenType of the associated
// UserTokenPolicy (see 7.37).
// If the SecurityPolicy is None then the tokenData only contains the UTF-8 encoded tokenData. This
// configuration should not be used unless the network is encrypted in some other manner such as a
// VPN. The use of this configuration without network encryption would result in a serious security
// fault, in that it would cause the appearance of a secure user access, but it would make the token
// visible in clear text.
//
// Specification: Part4, 7.36.6
type IssuedIdentityToken struct {
	PolicyID            *String
	TokenData           *ByteString
	EncryptionAlgorithm *String
}

// NewIssuedIdentityToken creates a new IssuedIdentityToken.
func NewIssuedIdentityToken(policyID string, tokenData []byte, alg string) *IssuedIdentityToken {
	return &IssuedIdentityToken{
		PolicyID:            NewString(policyID),
		TokenData:           NewByteString(tokenData),
		EncryptionAlgorithm: NewString(alg),
	}
}

// DecodeIssuedIdentityToken decodes given bytes as IssuedIdentityToken.
func DecodeIssuedIdentityToken(b []byte) (*IssuedIdentityToken, error) {
	i := &IssuedIdentityToken{}
	if err := i.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return i, nil
}

// DecodeFromBytes decodes given bytes as IssuedIdentityToken.
func (i *IssuedIdentityToken) DecodeFromBytes(b []byte) error {
	offset := 0
	i.PolicyID = &String{}
	if err := i.PolicyID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += i.PolicyID.Len()

	i.TokenData = &ByteString{}
	if err := i.TokenData.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += i.TokenData.Len()

	i.EncryptionAlgorithm = &String{}
	return i.EncryptionAlgorithm.DecodeFromBytes(b[offset:])
}

// Serialize serializes IssuedIdentityToken into bytes.
func (i *IssuedIdentityToken) Serialize() ([]byte, error) {
	b := make([]byte, i.Len())
	if err := i.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes IssuedIdentityToken into bytes.
func (i *IssuedIdentityToken) SerializeTo(b []byte) error {
	offset := 0
	if i.PolicyID != nil {
		if err := i.PolicyID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += i.PolicyID.Len()
	}

	if i.TokenData != nil {
		if err := i.TokenData.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += i.TokenData.Len()
	}

	if i.EncryptionAlgorithm != nil {
		if err := i.EncryptionAlgorithm.SerializeTo(b[offset:]); err != nil {
			return err
		}
	}

	return nil
}

// Len returns the actual Length of IssuedIdentityToken in int.
func (i *IssuedIdentityToken) Len() int {
	l := 0
	if i.PolicyID != nil {
		l += i.PolicyID.Len()
	}

	if i.TokenData != nil {
		l += i.TokenData.Len()
	}

	if i.EncryptionAlgorithm != nil {
		l += i.EncryptionAlgorithm.Len()
	}

	return l
}

// Type returns PolicyID in int.
func (i *IssuedIdentityToken) Type() int {
	return id.IssuedIdentityToken_Encoding_DefaultBinary
}
