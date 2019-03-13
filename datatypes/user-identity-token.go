// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"github.com/wmnsk/gopcua/id"
	"github.com/wmnsk/gopcua/ua"
)

// UserIdentityToken is an interface to handle all types of UserIdentityToken types as one type.
type UserIdentityToken interface {
	ExtensionObjectValue
}

// AnonymousIdentityToken is used to indicate that the Client has no user credentials.
//
// Specification: Part4, 7.36.5
type AnonymousIdentityToken struct {
	PolicyID string
}

// NewAnonymousIdentityToken creates a new AnonymousIdentityToken.
func NewAnonymousIdentityToken(policyID string) *AnonymousIdentityToken {
	return &AnonymousIdentityToken{
		PolicyID: policyID,
	}
}

func (t *AnonymousIdentityToken) Decode(b []byte) (int, error) {
	buf := ua.NewBuffer(b)
	t.PolicyID = buf.ReadString()
	return buf.Pos(), buf.Error()
}

func (t *AnonymousIdentityToken) Encode() ([]byte, error) {
	buf := ua.NewBuffer(nil)
	buf.WriteString(t.PolicyID)
	return buf.Bytes(), buf.Error()
}

// Type returns type of token defined in NodeIds.csv in int.
func (a *AnonymousIdentityToken) Type() int {
	return id.AnonymousIdentityToken_Encoding_DefaultBinary
}

// ID returns PolicyID in string.
func (a *AnonymousIdentityToken) ID() string {
	return a.PolicyID
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
	PolicyID            string
	UserName            string
	Password            []byte
	EncryptionAlgorithm string
}

// NewUserNameIdentityToken creates a new UserNameIdentityToken.
func NewUserNameIdentityToken(policyID, username string, password []byte, alg string) *UserNameIdentityToken {
	return &UserNameIdentityToken{
		PolicyID:            policyID,
		UserName:            username,
		Password:            password,
		EncryptionAlgorithm: alg,
	}
}

func (t *UserNameIdentityToken) Decode(b []byte) (int, error) {
	buf := ua.NewBuffer(b)
	t.PolicyID = buf.ReadString()
	t.UserName = buf.ReadString()
	t.Password = buf.ReadBytes()
	t.EncryptionAlgorithm = buf.ReadString()
	return buf.Pos(), buf.Error()
}

func (t *UserNameIdentityToken) Encode() ([]byte, error) {
	buf := ua.NewBuffer(nil)
	buf.WriteString(t.PolicyID)
	buf.WriteString(t.UserName)
	buf.WriteByteString(t.Password)
	buf.WriteString(t.EncryptionAlgorithm)
	return buf.Bytes(), buf.Error()
}

// Type returns type of token defined in NodeIds.csv in int.
func (u *UserNameIdentityToken) Type() int {
	return id.UserNameIdentityToken_Encoding_DefaultBinary
}

// ID returns PolicyID in string.
func (u *UserNameIdentityToken) ID() string {
	return u.PolicyID
}

// X509IdentityToken is used to pass an X.509 v3 Certificate which is issued by the user.
// This token shall always be accompanied by a Signature in the userTokenSignature parameter of
// ActivateSession if required by the SecurityPolicy. The Server should specify a SecurityPolicy for
// the UserTokenPolicy if the SecureChannel has a SecurityPolicy of None.
//
// Specification: Part4, 7.36.5
type X509IdentityToken struct {
	PolicyID        string
	CertificateData string
}

// NewX509IdentityToken creates a new X509IdentityToken.
func NewX509IdentityToken(policyID, cert string) *X509IdentityToken {
	return &X509IdentityToken{
		PolicyID:        policyID,
		CertificateData: cert,
	}
}

func (t *X509IdentityToken) Decode(b []byte) (int, error) {
	buf := ua.NewBuffer(b)
	t.PolicyID = buf.ReadString()
	t.CertificateData = buf.ReadString()
	return buf.Pos(), buf.Error()
}

func (t *X509IdentityToken) Encode() ([]byte, error) {
	buf := ua.NewBuffer(nil)
	buf.WriteString(t.PolicyID)
	buf.WriteString(t.CertificateData)
	return buf.Bytes(), buf.Error()
}

// Type returns type of token defined in NodeIds.csv in int.
func (x *X509IdentityToken) Type() int {
	return id.X509IdentityToken_Encoding_DefaultBinary
}

// ID returns PolicyID in string.
func (x *X509IdentityToken) ID() string {
	return x.PolicyID
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
	PolicyID            string
	TokenData           []byte
	EncryptionAlgorithm string
}

// NewIssuedIdentityToken creates a new IssuedIdentityToken.
func NewIssuedIdentityToken(policyID string, tokenData []byte, alg string) *IssuedIdentityToken {
	return &IssuedIdentityToken{
		PolicyID:            policyID,
		TokenData:           tokenData,
		EncryptionAlgorithm: alg,
	}
}

func (t *IssuedIdentityToken) Decode(b []byte) (int, error) {
	buf := ua.NewBuffer(b)
	t.PolicyID = buf.ReadString()
	t.TokenData = buf.ReadBytes()
	t.EncryptionAlgorithm = buf.ReadString()
	return buf.Pos(), buf.Error()
}

func (t *IssuedIdentityToken) Encode() ([]byte, error) {
	buf := ua.NewBuffer(nil)
	buf.WriteString(t.PolicyID)
	buf.WriteByteString(t.TokenData)
	buf.WriteString(t.EncryptionAlgorithm)
	return buf.Bytes(), buf.Error()
}

// Type returns type of token defined in NodeIds.csv in int.
func (i *IssuedIdentityToken) Type() int {
	return id.IssuedIdentityToken_Encoding_DefaultBinary
}

// ID returns PolicyID in string.
func (i *IssuedIdentityToken) ID() string {
	return i.PolicyID
}
