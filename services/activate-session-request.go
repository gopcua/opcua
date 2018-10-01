// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"github.com/wmnsk/gopcua/datatypes"
)

// ActivateSessionRequest is used by the Client to specify the identity of the user
// associated with the Session. This Service request shall be issued by the Client
// before it issues any Service request other than CloseSession after CreateSession.
// Failure to do so shall cause the Server to close the Session.
//
// Whenever the Client calls this Service the Client shall prove that it is the same application that
// called the CreateSession Service. The Client does this by creating a signature with the private key
// associated with the clientCertificate specified in the CreateSession request. This signature is
// created by appending the last serverNonce provided by the Server to the serverCertificate and
// calculating the signature of the resulting sequence of bytes.
//
// Specification: Part 4, 5.6.3.2
type ActivateSessionRequest struct {
	TypeID *datatypes.ExpandedNodeID
	*RequestHeader
	ClientSignature            *SignatureData
	ClientSoftwareCertificates *SignedSoftwareCertificateArray
	LocaleIDs                  *datatypes.StringArray
	UserIdentityToken          *datatypes.ExtensionObject
	UserTokenSignature         *SignatureData
}

// NewActivateSessionRequest creates a new ActivateSessionRequest.
func NewActivateSessionRequest(reqHeader *RequestHeader, sig *SignatureData, certs []*SignedSoftwareCertificate, locales []string, userToken *datatypes.ExtensionObject, tokenSig *SignatureData) *ActivateSessionRequest {
	return &ActivateSessionRequest{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(
				0, ServiceTypeActivateSessionRequest,
			),
			"", 0,
		),
		RequestHeader:              reqHeader,
		ClientSignature:            sig,
		ClientSoftwareCertificates: NewSignedSoftwareCertificateArray(certs),
		LocaleIDs:                  datatypes.NewStringArray(locales),
		UserIdentityToken:          userToken,
		UserTokenSignature:         tokenSig,
	}
}

// DecodeActivateSessionRequest decodes given bytes into ActivateSessionRequest.
func DecodeActivateSessionRequest(b []byte) (*ActivateSessionRequest, error) {
	a := &ActivateSessionRequest{}
	if err := a.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return a, nil
}

// DecodeFromBytes decodes given bytes into ActivateSessionRequest.
func (a *ActivateSessionRequest) DecodeFromBytes(b []byte) error {
	offset := 0

	// type id
	a.TypeID = &datatypes.ExpandedNodeID{}
	if err := a.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.TypeID.Len()

	// request header
	a.RequestHeader = &RequestHeader{}
	if err := a.RequestHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.RequestHeader.Len() - len(a.RequestHeader.Payload)

	// client signature
	a.ClientSignature = &SignatureData{}
	if err := a.ClientSignature.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.ClientSignature.Len()

	// client software certificates
	a.ClientSoftwareCertificates = &SignedSoftwareCertificateArray{}
	if err := a.ClientSoftwareCertificates.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.ClientSoftwareCertificates.Len()

	// locale ids
	a.LocaleIDs = &datatypes.StringArray{}
	if err := a.LocaleIDs.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.LocaleIDs.Len()

	// user identity token
	a.UserIdentityToken = &datatypes.ExtensionObject{}
	if err := a.UserIdentityToken.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += a.UserIdentityToken.Len()

	// user token signature
	a.UserTokenSignature = &SignatureData{}
	return a.UserTokenSignature.DecodeFromBytes(b[offset:])
}

// Serialize serializes ActivateSessionRequest into bytes.
func (a *ActivateSessionRequest) Serialize() ([]byte, error) {
	b := make([]byte, a.Len())
	if err := a.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes ActivateSessionRequest into bytes.
func (a *ActivateSessionRequest) SerializeTo(b []byte) error {
	var offset = 0
	if a.TypeID != nil {
		if err := a.TypeID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.TypeID.Len()
	}

	if a.RequestHeader != nil {
		if err := a.RequestHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.RequestHeader.Len()
	}

	if a.ClientSignature != nil {
		if err := a.ClientSignature.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.ClientSignature.Len()
	}

	if a.ClientSoftwareCertificates != nil {
		if err := a.ClientSoftwareCertificates.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.ClientSoftwareCertificates.Len()
	}

	if a.LocaleIDs != nil {
		if err := a.LocaleIDs.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.LocaleIDs.Len()
	}

	if a.UserIdentityToken != nil {
		if err := a.UserIdentityToken.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += a.UserIdentityToken.Len()
	}

	if a.UserTokenSignature != nil {
		if err := a.UserTokenSignature.SerializeTo(b[offset:]); err != nil {
			return err
		}
	}

	return nil
}

// Len returns the actual length of ActivateSessionRequest.
func (a *ActivateSessionRequest) Len() int {
	length := 0

	if a.TypeID != nil {
		length += a.TypeID.Len()
	}

	if a.RequestHeader != nil {
		length += a.RequestHeader.Len()
	}

	if a.ClientSignature != nil {
		length += a.ClientSignature.Len()
	}

	if a.ClientSoftwareCertificates != nil {
		length += a.ClientSoftwareCertificates.Len()
	}

	if a.LocaleIDs != nil {
		length += a.LocaleIDs.Len()
	}

	if a.UserIdentityToken != nil {
		length += a.UserIdentityToken.Len()
	}

	if a.UserTokenSignature != nil {
		length += a.UserTokenSignature.Len()
	}

	return length
}

// ServiceType returns type of Service.
func (a *ActivateSessionRequest) ServiceType() uint16 {
	return ServiceTypeActivateSessionRequest
}
