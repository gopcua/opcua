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
	TypeID                     *datatypes.ExpandedNodeID
	RequestHeader              *RequestHeader
	ClientSignature            *SignatureData
	ClientSoftwareCertificates []*SignedSoftwareCertificate
	LocaleIDs                  []string
	UserIdentityToken          *datatypes.ExtensionObject
	UserTokenSignature         *SignatureData
}

// NewActivateSessionRequest creates a new ActivateSessionRequest.
func NewActivateSessionRequest(reqHeader *RequestHeader, sig *SignatureData, locales []string, userToken datatypes.UserIdentityToken, tokenSig *SignatureData) *ActivateSessionRequest {
	return &ActivateSessionRequest{
		TypeID:                     datatypes.NewFourByteExpandedNodeID(0, ServiceTypeActivateSessionRequest),
		RequestHeader:              reqHeader,
		ClientSignature:            sig,
		ClientSoftwareCertificates: nil,
		LocaleIDs:                  locales,
		UserIdentityToken:          datatypes.NewExtensionObject(0x01, userToken),
		UserTokenSignature:         tokenSig,
	}
}

// ServiceType returns type of Service.
func (a *ActivateSessionRequest) ServiceType() uint16 {
	return ServiceTypeActivateSessionRequest
}
