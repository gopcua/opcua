package uasc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/gopcua/opcua/keyring"
	"github.com/gopcua/opcua/securitypolicy"
	"github.com/gopcua/opcua/ua"
)

type Session struct {
	sechan *SecureChannel
	cfg    *SessionConfig

	maxRequestMessageSize uint32

	// mySignature is is the client/serverSignature expected to receive from the other endpoint.
	// This parameter is automatically calculated and kept temporarily until being used to verify
	// received client/serverSignature.
	mySignature *ua.SignatureData

	// signatureToSend is the client/serverSignature defined in Part4, Table 15 and Table 17.
	// This parameter is automatically calculated and kept temporarily until it is sent in next message.
	signatureToSend *ua.SignatureData
}

func NewSession(sechan *SecureChannel, cfg *SessionConfig) *Session {
	return &Session{sechan: sechan, cfg: cfg}
}

func (s *Session) Open() error {
	if err := s.createSession(); err != nil {
		return err
	}
	return s.activateSession()
}

func (s *Session) Close() error {
	return nil
}

func (s *Session) createSession() error {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	req := &ua.CreateSessionRequest{
		ClientDescription: s.cfg.ClientDescription,
		EndpointURL:       s.sechan.EndpointURL,
		SessionName:       fmt.Sprintf("gopcua-%d", time.Now().UnixNano()),
		ClientNonce:       nonce,
		ClientCertificate: s.sechan.cfg.LocalCertificate,
	}

	return s.sechan.Send(req, func(v interface{}) error {
		resp, ok := v.(*ua.CreateSessionResponse)
		if !ok {
			return fmt.Errorf("invalid response. Got %T, want CreateSessionResponse", v)
		}

		var sig []byte
		var sigAlg string
		if s.sechan.cfg.ServerEndpoint.SecurityMode != ua.MessageSecurityModeNone {
			localKey, err := keyring.PrivateKey(s.sechan.cfg.LocalThumbprint)
			if err != nil {
				return err
			}

			remoteCert, err := x509.ParseCertificate(resp.ServerCertificate)
			if err != nil {
				return err
			}
			remoteKey := remoteCert.PublicKey.(*rsa.PublicKey)

			enc, err := securitypolicy.Asymmetric(s.sechan.cfg.ServerEndpoint.SecurityPolicyURI, localKey, remoteKey)
			if err != nil {
				return err
			}
			err = enc.VerifySignature(append(s.sechan.cfg.LocalCertificate, nonce...), resp.ServerSignature.Signature)
			if err != nil {
				return err
			}

			sig, err = enc.Signature(append(resp.ServerCertificate, resp.ServerNonce...))
			if err != nil {
				return err
			}
			sigAlg = enc.SignatureURI()
		}
		s.sechan.reqhdr.AuthenticationToken = resp.AuthenticationToken
		s.cfg.ServerEndpoints = resp.ServerEndpoints
		s.cfg.SessionTimeout = resp.RevisedSessionTimeout

		s.signatureToSend = &ua.SignatureData{
			Algorithm: sigAlg,
			Signature: sig,
		}

		s.maxRequestMessageSize = resp.MaxRequestMessageSize

		// todo(fs): fix crypto
		// keep SignatureData to verify serverSignature in CreateSessionResponse.
		// s.mySignature = ua.NewSignatureDataFrom(s.sechan.cfg.Certificate, nonce)
		s.mySignature = &ua.SignatureData{}
		return nil
	})
}

func (s *Session) activateSession() error {
	req := &ua.ActivateSessionRequest{
		ClientSignature:            s.signatureToSend,
		ClientSoftwareCertificates: nil,
		LocaleIDs:                  s.cfg.LocaleIDs,
		UserIdentityToken:          ua.NewExtensionObject(s.cfg.UserIdentityToken),
		UserTokenSignature:         s.cfg.UserTokenSignature,
	}
	return s.sechan.Send(req, func(v interface{}) error {
		resp, ok := v.(*ua.ActivateSessionResponse)
		if !ok {
			return fmt.Errorf("invalid response. Got %T, want ActivateSessionResponse", v)
		}

		for _, result := range resp.Results {
			if result != 0 {
				return fmt.Errorf("rejected")
			}
		}
		return nil
	})
}
