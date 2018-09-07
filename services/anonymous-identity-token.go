package services

import "github.com/wmnsk/gopcua/datatypes"

// AnonymousIdentityToken is used to indicate that the Client has no user credentials.
//
// Specification: Part 4, 7.36.3
type AnonymousIdentityToken struct {
	PolicyID *datatypes.String
}

// NewAnonymousIdentityToken creates a new AnonymousIdentityToken.
func NewAnonymousIdentityToken() *AnonymousIdentityToken {
	return &AnonymousIdentityToken{
		PolicyID: datatypes.NewString("anonymous"),
	}
}

// DecodeAnonymousIdentityToken decodes given bytes into AnonymousIdentityToken.
func DecodeAnonymousIdentityToken(b []byte) (*AnonymousIdentityToken, error) {
	a := &AnonymousIdentityToken{}
	if err := a.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return a, nil
}

// DecodeFromBytes decodes given bytes into AnonymousIdentityToken.
func (a *AnonymousIdentityToken) DecodeFromBytes(b []byte) error {
	a.PolicyID = &datatypes.String{}
	return a.PolicyID.DecodeFromBytes(b)

	return nil
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

// Len returns the actual length of AnonymousIdentityToken.
func (a *AnonymousIdentityToken) Len() int {
	length := 0
	if a.PolicyID != nil {
		length += a.PolicyID.Len()
	}

	return length
}

// ServiceType returns type of Service.
func (a *AnonymousIdentityToken) ServiceType() uint16 {
	return 321
}
