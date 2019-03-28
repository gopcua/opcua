// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	mrand "math/rand"
	"time"

	"github.com/gopcua/opcua/keyring"
	"github.com/gopcua/opcua/ua"

	"github.com/pkg/errors"
)

// Config represents a configuration which UASC client/server has in common.
type Config struct {
	// SecureChannelID is a unique identifier for the SecureChannel assigned by the Server.
	// If a Server receives a SecureChannelId which it does not recognize it shall return an
	// appropriate transport layer error.
	//
	// When a Server starts the first SecureChannelId used should be a value that is likely to
	// be unique after each restart. This ensures that a Server restart does not cause
	// previously connected Clients to accidentally ‘reuse’ SecureChannels that did not belong
	// to them.
	SecureChannelID uint32

	// ServerEndpoint is endpoint description to connect to.  Received from the GetEndpointsRequest
	// Contains:
	//   - SecurityPolicyURI
	//   - MessageSecurityMode
	//   - Server Certificate
	//   - Authentication Profile URI
	ServerEndpoint *ua.EndpointDescription

	// SecurityPolicyURI is the URI of the Security Policy used to secure the Message.
	// This field is encoded as a UTF-8 string without a null terminator.
	SecurityPolicyURI string

	// Certificate []byte
	// Thumbprint  []byte
	// LocalCertificate is the X.509 v3 Certificate assigned to the sending application Instance.
	// This is a DER encoded blob.
	// The structure of an X.509 v3 Certificate is defined in X.509 v3.
	// The DER format for a Certificate is defined in X690.
	// This indicates what Private Key was used to sign the MessageChunk.
	// The Stack shall close the channel and report an error to the application if
	// the Certificate is too large for the buffer size supported by the
	// transport layer.
	// This field shall be null if the Message is not signed.
	LocalCertificate []byte

	// LocalThumbprint is the thumbprint of the X.509 v3 Certificate assigned to the receiving
	// application Instance.
	// The thumbprint is the CertificateDigest of the DER encoded form of the
	// Certificate.
	// This indicates what public key was used to encrypt the MessageChunk.
	// This field shall be null if the Message is not encrypted.
	LocalThumbprint []byte

	// RemoteCertificate is the X.509 v3 Certificate assigned to the receiving application Instance.
	// This is a DER encoded blob.
	// The structure of an X.509 v3 Certificate is defined in X.509 v3.
	// The DER format for a Certificate is defined in X690.
	// This indicates what Private Key was used to sign the MessageChunk.
	// The Stack shall close the channel and report an error to the application if
	// the Certificate is too large for the buffer size supported by the
	// transport layer.
	// This field shall be null if the Message is not signed.
	//RemoteCertificate []byte

	// RemoteThumbprint is the thumbprint of the X.509 v3 Certificate assigned to the receiving
	// application Instance.
	// The thumbprint is the CertificateDigest of the DER encoded form of the
	// Certificate.
	// This indicates what public key was used to encrypt the MessageChunk.
	// This field shall be null if the Message is not encrypted.
	//RemoteThumbprint []byte

	// SequenceNumber is a monotonically increasing sequence number assigned by the sender to each
	// MessageChunk sent over the SecureChannel.
	SequenceNumber uint32

	// RequestID is an identifier assigned by the Client to OPC UA request Message. All MessageChunks
	// for the request and the associated response use the same identifier
	RequestID uint32

	// SecurityMode is The type of security to apply to the messages. The type MessageSecurityMode
	// is defined in 7.15.
	// A SecureChannel may have to be created even if the securityMode is NONE. The exact behaviour
	// depends on the mapping used and is described in the Part 6.
	// SecurityMode ua.MessageSecurityMode

	// SecurityTokenID is a unique identifier for the SecureChannel SecurityToken used to secure the Message.
	// This identifier is returned by the Server in an OpenSecureChannel response Message.
	// If a Server receives a TokenId which it does not recognize it shall return an appropriate
	// transport layer error.
	SecurityTokenID uint32

	// Lifetime is the requested lifetime, in milliseconds, for the new SecurityToken when the
	// SecureChannel works as client. It specifies when the Client expects to renew the SecureChannel
	// by calling the OpenSecureChannel Service again. If a SecureChannel is not renewed, then all
	// Messages sent using the current SecurityTokens shall be rejected by the receiver.
	// Lifetime can also be the revised lifetime, the lifetime of the SecurityToken in milliseconds.
	// The UTC expiration time for the token may be calculated by adding the lifetime to the createdAt time.
	Lifetime uint32
}

// // NewConfig creates a new Config.
// //
// // This contains all the parameter Config has, but the ones should be set depends on the application type.
// // It is good idea to use NewClientConfig or NewServerConfig instead if you don't have specific purpose to
// // create Config with full parameters.
// func NewConfig(chanID uint32, policyURI string, cert, thumbprint []byte, seqNum, reqID, secMode, tokenID, lifetime uint32) *Config {
// 	return &Config{
// 		SecureChannelID:   chanID,
// 		SecurityPolicyURI: policyURI,
// 		Certificate:       cert,
// 		Thumbprint:        thumbprint,
// 		SequenceNumber:    seqNum,
// 		RequestID:         reqID,
// 		SecurityMode:      secMode,
// 		SecurityTokenID:   tokenID,
// 		Lifetime:          lifetime,
// 	}
// }

// NewClientConfig creates a new Config for Client.
//
// With all the parameter given, it is sufficient for client to open SecureChannel.
// If the secMode is None, cert and thumbprint is not required(can be nil).
func NewClientConfig(serverEndpoint *ua.EndpointDescription, localKey *rsa.PrivateKey, localCert []byte, lifetime uint32) *Config {
	var err error

	if serverEndpoint == nil || localKey == nil || localCert == nil {
		return NewClientConfigSecurityNone(lifetime)
	}

	remoteX509, err := x509.ParseCertificate(serverEndpoint.ServerCertificate)
	if err != nil {
		return nil // Bad Security Error instead?
	}
	_ = keyring.Add(remoteX509, nil)

	localX509, err := x509.ParseCertificate(localCert)
	if err != nil {
		return nil // Bad Security Error instead?
	}
	localThumb := keyring.Add(localX509, localKey)

	return &Config{
		ServerEndpoint: serverEndpoint,
		//		SecurityPolicyURI: policyURI,
		LocalCertificate: localCert,
		//		RemoteCertificate: remoteCert,
		LocalThumbprint: localThumb,
		//		RemoteThumbprint:  remoteThumb,
		RequestID: uint32(mrand.Int31()),
		//		SecurityMode:      secMode,
		Lifetime: lifetime,
	}
}

// NewClientConfigSecurityNone creates a new Config for Client, with SecurityMode=None.
func NewClientConfigSecurityNone(lifetime uint32) *Config {
	serverEndpoint := &ua.EndpointDescription{
		//EndpointURL:       "ep-url",
		SecurityPolicyURI: "http://opcfoundation.org/UA/SecurityPolicy#None",
		SecurityMode:      ua.MessageSecurityModeNone,
		SecurityLevel:     1,
		ServerCertificate: nil,
		UserIdentityTokens: []*ua.UserTokenPolicy{
			&ua.UserTokenPolicy{
				PolicyID:          "Anonymous",
				TokenType:         ua.UserTokenTypeAnonymous,
				IssuedTokenType:   "",
				IssuerEndpointURL: "",
				SecurityPolicyURI: "http://opcfoundation.org/UA/SecurityPolicy#None",
			},
		},
		TransportProfileURI: "http://opcfoundation.org/UA-Profile/Transport/uatcp-uasc-uabinary",
	}

	return &Config{
		//SecurityPolicyURI: "http://opcfoundation.org/UA/SecurityPolicy#None",
		ServerEndpoint: serverEndpoint,
		RequestID:      uint32(mrand.Int31()),
		//		SecurityMode:   ua.MessageSecurityModeNone,
		LocalThumbprint: nil,
		Lifetime:        lifetime,
	}
}

/* XXX - to be uncommented when encryption is
// NewClientConfigSignBasic256Sha256 creates a new Config for Client, with SecurityMode=Sign
// and SecurityPolicy=Basic256Sha256.
func NewClientConfigSignBasic256Sha256(cert, thumbprint []byte, reqID, lifetime uint32) *Config {
	return NewClientConfig(
		"http://opcfoundation.org/UA/SecurityPolicy#Basic256Sha256",
		cert, thumbprint, reqID, ua.MessageSecurityModeSign, lifetime,
	)
}

// NewClientConfigSignAndEncryptBasic256Sha256 creates a new Config for Client, with SecurityMode=SignAndEncrypt
// and SecurityPolicy=Basic256Sha256.
func NewClientConfigSignAndEncryptBasic256Sha256(cert, thumbprint []byte, reqID, lifetime uint32) *Config {
	return NewClientConfig(
		"http://opcfoundation.org/UA/SecurityPolicy#Basic256Sha256",
		cert, thumbprint, reqID, ua.MessageSecurityModeSignAndEncrypt, lifetime,
	)
}

// NewClientConfigSignAes128Sha256RsaOaep creates a new Config for Client, with SecurityMode=Sign
// and SecurityPolicy=Aes128_Sha256_RsaOaep.
func NewClientConfigSignAes128Sha256RsaOaep(cert, thumbprint []byte, reqID, lifetime uint32) *Config {
	return NewClientConfig(
		"http://opcfoundation.org/UA/SecurityPolicy#Aes128_Sha256_RsaOaep",
		cert, thumbprint, reqID, ua.MessageSecurityModeSign, lifetime,
	)
}

// NewClientConfigSignAndEncryptAes128Sha256RsaOaep creates a new Config for Client, with SecurityMode=SignAndEncrypt
// and SecurityPolicy=Aes128_Sha256_RsaOaep.
func NewClientConfigSignAndEncryptAes128Sha256RsaOaep(cert, thumbprint []byte, reqID, lifetime uint32) *Config {
	return NewClientConfig(
		"http://opcfoundation.org/UA/SecurityPolicy#Aes128_Sha256_RsaOaep",
		cert, thumbprint, reqID, ua.MessageSecurityModeSignAndEncrypt, lifetime,
	)
}

// NewClientConfigSignPubSubAes128CTR creates a new Config for Client, with SecurityMode=Sign
// and SecurityPolicy=PubSub_Aes128_CTR.
func NewClientConfigSignPubSubAes128CTR(cert, thumbprint []byte, reqID, lifetime uint32) *Config {
	return NewClientConfig(
		"http://opcfoundation.org/UA/SecurityPolicy#PubSub_Aes128_CTR",
		cert, thumbprint, reqID, ua.MessageSecurityModeSign, lifetime,
	)
}

// NewClientConfigSignAndEncryptPubSubAes128CTR creates a new Config for Client, with SecurityMode=SignAndEncrypt
// and SecurityPolicy=PubSub_Aes128_CTR.
func NewClientConfigSignAndEncryptPubSubAes128CTR(cert, thumbprint []byte, reqID, lifetime uint32) *Config {
	return NewClientConfig(
		"http://opcfoundation.org/UA/SecurityPolicy#PubSub_Aes128_CTR",
		cert, thumbprint, reqID, ua.MessageSecurityModeSignAndEncrypt, lifetime,
	)
}

// NewClientConfigSignPubSubAes256CTR creates a new Config for Client, with SecurityMode=Sign
// and SecurityPolicy=PubSub_Aes256_CTR.
func NewClientConfigSignPubSubAes256CTR(cert, thumbprint []byte, reqID, lifetime uint32) *Config {
	return NewClientConfig(
		"http://opcfoundation.org/UA/SecurityPolicy#PubSub_Aes256_CTR",
		cert, thumbprint, reqID, ua.MessageSecurityModeSign, lifetime,
	)
}

// NewClientConfigSignAndEncryptPubSubAes256CTR creates a new Config for Client, with SecurityMode=SignAndEncrypt
// and SecurityPolicy=PubSub_Aes256_CTR.
func NewClientConfigSignAndEncryptPubSubAes256CTR(cert, thumbprint []byte, reqID, lifetime uint32) *Config {
	return NewClientConfig(
		"http://opcfoundation.org/UA/SecurityPolicy#PubSub_Aes256_CTR",
		cert, thumbprint, reqID, ua.MessageSecurityModeSignAndEncrypt, lifetime,
	)
}
*/

// NewServerConfig creates a new Config for Server.
//
// With all the parameter given, it is sufficient for server to accept SecureChannel.
// If the secMode is None, cert and thumbprint is not required(can be nil).
func NewServerConfig(serverEndpoint *ua.EndpointDescription, localKey *rsa.PrivateKey, localCert []byte, chanID, tokenID, lifetime uint32) *Config {
	//func NewServerConfig(policyURI string, cert, thumbprint []byte, chanID uint32, secMode ua.MessageSecurityMode, tokenID, lifetime uint32) *Config {
	return &Config{
		//SecurityPolicyURI: policyURI,
		ServerEndpoint: serverEndpoint,
		//Certificate:       cert,
		//Thumbprint:        thumbprint,
		SecureChannelID: chanID,
		//SecurityMode:      secMode,
		SecurityTokenID: tokenID,
		Lifetime:        lifetime,
	}
}

// todo(dh): Temporarily disable validation of configs until the structure stops changing
// validate validates Config. This is just to avoid crash. Strange values would be accepted for flexibility.
//func (c *Config) validate(appType string) error {
//	switch appType {
//	case "client":
//		return c.validateClientConfig()
//	case "server":
//		return c.validateClientConfig()
//	default:
//		return errors.New("invalid type. should be client or server")
//	}
//}

//func (c *Config) validateClientConfig() error {
//	if c.SecurityMode == ua.MessageSecurityModeSignAndEncrypt && (c.Certificate == nil || c.Thumbprint == nil) {
//		return errors.New("Certificate, Thumbprint is required when using SignAndEncrypt")
//	}
//
//	if c.SecurityMode == ua.MessageSecurityModeNone {
//		c.Certificate = nil
//		c.Thumbprint = nil
//	}
//	return nil
//}

//func (c *Config) validateServerConfig() error {
//	if c.SecurityMode == ua.MessageSecurityModeNone {
//		c.Certificate = nil
//		c.Thumbprint = nil
//	}
//	return nil
//}

// SessionConfig is a set of common configurations used in Session.
type SessionConfig struct {
	// AuthenticationToken is the secret Session identifier used to verify that the request is
	// associated with the Session. The SessionAuthenticationToken type is defined in 7.31.
	AuthenticationToken *ua.NodeID

	// ClientDescription is the information that describes the Client application.
	// The type ApplicationDescription is defined in 7.1.
	ClientDescription *ua.ApplicationDescription

	// ServerEndpoints is the list of Endpoints that the Server supports.
	// The Server shall return a set of EndpointDescriptions available for the serverUri
	// specified in the request. The EndpointDescription type is defined in 7.10. The Client
	// shall verify this list with the list from a DiscoveryEndpoint if it used a
	// DiscoveryEndpoint to fetch the EndpointDescriptions.
	// It is recommended that Servers only include the server.applicationUri, endpointUrl,
	// securityMode, securityPolicyUri, userIdentityTokens, transportProfileUri and
	// securityLevel with all other parameters set to null. Only the recommended
	// parameters shall be verified by the client.
	ServerEndpoints []*ua.EndpointDescription

	// LocaleIDs is the list of locale ids in priority order for localized strings. The first
	// LocaleId in the list has the highest priority. If the Server returns a localized string
	// to the Client, the Server shall return the translation with the highest priority that
	// it can. If it does not have a translation for any of the locales identified in this list,
	// then it shall return the string value that it has and include the locale id with the
	// string. See Part 3 for more detail on locale ids. If the Client fails to specify at least
	// one locale id, the Server shall use any that it has.
	// This parameter only needs to be specified during the first call to ActivateSession during
	// a single application Session. If it is not specified the Server shall keep using the
	// current localeIds for the Session.
	LocaleIDs []string

	// UserIdentityToken is the credentials of the user associated with the Client application.
	// The Server uses these credentials to determine whether the Client should be allowed to
	// activate a Session and what resources the Client has access to during this Session.
	// The UserIdentityToken is an extensible parameter type defined in 7.36.
	// The EndpointDescription specifies what UserIdentityTokens the Server shall accept.
	// Null or empty user token shall always be interpreted as anonymous.
	UserIdentityToken interface{}

	// If the Client specified a user identity token that supports digital signatures, then it
	// shall create a signature and pass it as this parameter. Otherwise the parameter is null.
	// The SignatureAlgorithm depends on the identity token type.
	// The SignatureData type is defined in 7.32.
	UserTokenSignature *ua.SignatureData

	// If Session works as a client, SessionTimeout is the requested maximum number of milliseconds
	// that a Session should remain open without activity. If the Client fails to issue a Service
	// request within this interval, then the Server shall automatically terminate the Client Session.
	// If Session works as a server, SessionTimeout is an actual maximum number of milliseconds
	// that a Session shall remain open without activity. The Server should attempt to honour the
	// Client request for this parameter,but may negotiate this value up or down to meet its own constraints.
	SessionTimeout float64

	// mySignature is is the client/serverSignature expected to receive from the other endpoint.
	// This parameter is automatically calculated and kept temporarily until being used to verify
	// received client/serverSignature.
	// todo(fs): temp disable until the security code is resurrected. keep golangcibot happy
	// mySignature *ua.SignatureData

	// signatureToSend is the client/serverSignature defined in Part4, Table 15 and Table 17.
	// This parameter is automatically calculated and kept temporarily until it is sent in next message.
	// todo(fs): temp disable until the security code is resurrected. keep golangcibot happy
	// signatureToSend *ua.SignatureData
}

// NewClientSessionConfig creates a SessionConfig for client.
func NewClientSessionConfig(applicationURI string, locales []string, userToken interface{}) *SessionConfig {
	if applicationURI == "" {
		applicationURI = "urn:gopcua:client"
	}
	return &SessionConfig{
		SessionTimeout: 0xffff,
		ClientDescription: &ua.ApplicationDescription{
			ApplicationURI:  applicationURI,
			ProductURI:      "urn:gopcua",
			ApplicationName: &ua.LocalizedText{Text: "gopcua - OPC UA implementation in pure Go"},
			ApplicationType: ua.ApplicationTypeClient,
		},
		LocaleIDs:          locales,
		UserIdentityToken:  userToken,
		UserTokenSignature: &ua.SignatureData{},
	}
}

// NewServerSessionConfig creates a new SessionConfigServer for server.
func NewServerSessionConfig(secChan *SecureChannel) *SessionConfig {
	rawToken := make([]byte, 2)
	if _, err := rand.Read(rawToken); err != nil {
		binary.LittleEndian.PutUint16(rawToken, uint16(time.Now().UnixNano()))
	}
	return &SessionConfig{
		AuthenticationToken: ua.NewFourByteNodeID(0, binary.LittleEndian.Uint16(rawToken)),
		SessionTimeout:      0xffff,
		ServerEndpoints: []*ua.EndpointDescription{
			&ua.EndpointDescription{
				EndpointURL: secChan.LocalEndpoint(),
				Server: &ua.ApplicationDescription{
					ApplicationURI:  "urn:gopcua:client",
					ProductURI:      "urn:gopcua",
					ApplicationName: &ua.LocalizedText{Text: "gopcua - OPC UA implementation in pure Go"},
					ApplicationType: ua.ApplicationTypeServer,
				},
				ServerCertificate: secChan.cfg.ServerEndpoint.ServerCertificate,
				SecurityMode:      secChan.cfg.ServerEndpoint.SecurityMode,
				SecurityPolicyURI: secChan.cfg.ServerEndpoint.SecurityPolicyURI,
				// UserIdentityTokens: []*ua.UserTokenPolicy{&ua.UserTokenPolicy{}},
			},
		},
	}
}

// validate validates SessionConfig. This is just to avoid crash. Strange values would be accepted for flexibility.
func (s *SessionConfig) validate(appType string) error {
	switch appType {
	case "client":
		return s.validateClientSessionConfig()
	case "server":
		return s.validateClientSessionConfig()
	default:
		return errors.New("invalid type. should be client or server")
	}
}

func (s *SessionConfig) validateClientSessionConfig() error {
	return nil
}

func (s *SessionConfig) validateServerSessionConfig() error {
	return nil
}
