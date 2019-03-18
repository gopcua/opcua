// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

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
// type ActivateSessionRequest struct {
// 	RequestHeader              *RequestHeader
// 	ClientSignature            *SignatureData
// 	ClientSoftwareCertificates []*SignedSoftwareCertificate
// 	LocaleIDs                  []string
// 	UserIdentityToken          *ExtensionObject
// 	UserTokenSignature         *SignatureData
// }

// NewActivateSessionRequest creates a new ActivateSessionRequest.
func NewActivateSessionRequest(reqHeader *RequestHeader, sig *SignatureData, locales []string, userToken interface{}, tokenSig *SignatureData) *ActivateSessionRequest {
	return &ActivateSessionRequest{
		RequestHeader:              reqHeader,
		ClientSignature:            sig,
		ClientSoftwareCertificates: nil,
		LocaleIDs:                  locales,
		UserIdentityToken:          NewExtensionObject(0x01, userToken),
		UserTokenSignature:         tokenSig,
	}
}
