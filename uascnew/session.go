package uascnew

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/services"
	"github.com/wmnsk/gopcua/uasc"
)

type Session struct {
	sechan *SecureChannel
	cfg    *uasc.SessionConfig

	maxRequestMessageSize uint32

	// mySignature is is the client/serverSignature expected to receive from the other endpoint.
	// This parameter is automatically calculated and kept temporarily until being used to verify
	// received client/serverSignature.
	mySignature *services.SignatureData

	// signatureToSend is the client/serverSignature defined in Part4, Table 15 and Table 17.
	// This parameter is automatically calculated and kept temporarily until it is sent in next message.
	signatureToSend *services.SignatureData
}

func NewSession(sechan *SecureChannel, cfg *uasc.SessionConfig) *Session {
	return &Session{sechan: sechan, cfg: cfg}
}

func (s *Session) Open() error {
	if err := s.sendCreateSessionRequest(); err != nil {
		return err
	}
	if err := s.handleCreateSessionResponse(); err != nil {
		return err
	}
	if err := s.sendActivateSessionRequest(); err != nil {
		return err
	}
	if err := s.handleActivateSessionResponse(); err != nil {
		return err
	}
	go s.sechan.monitor()
	return nil
}

func (s *Session) Close() error {
	return nil
}

func (s *Session) sendCreateSessionRequest() error {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	req := &services.CreateSessionRequest{
		ClientDescription: s.cfg.ClientDescription,
		EndpointURL:       s.sechan.endpointURL,
		SessionName:       fmt.Sprintf("gopcua-%d", time.Now().UnixNano()),
		ClientNonce:       nonce,
		ClientCertificate: s.sechan.cfg.Certificate,
	}

	if err := s.sechan.send(req, nil); err != nil {
		return err
	}

	// keep SignatureData to verify serverSignature in CreateSessionResponse.
	s.mySignature = services.NewSignatureDataFrom(s.sechan.cfg.Certificate, nonce)
	return nil
}

func (s *Session) handleCreateSessionResponse() error {
	svc, err := s.sechan.recv()
	if err != nil {
		return err
	}
	resp, ok := svc.(*services.CreateSessionResponse)
	if !ok {
		return fmt.Errorf("invalid response. Got %T, want CreateSessionResponse", svc)
	}

	s.sechan.reqHeader.AuthenticationToken = resp.AuthenticationToken
	s.cfg.ServerEndpoints = resp.ServerEndpoints
	s.cfg.SessionTimeout = resp.RevisedSessionTimeout
	s.signatureToSend = services.NewSignatureDataFrom(resp.ServerCertificate, resp.ServerNonce)
	s.maxRequestMessageSize = resp.MaxRequestMessageSize
	return nil
}

func (s *Session) sendActivateSessionRequest() error {
	req := &services.ActivateSessionRequest{
		ClientSignature:            s.signatureToSend,
		ClientSoftwareCertificates: nil,
		LocaleIDs:                  s.cfg.LocaleIDs,
		UserIdentityToken:          datatypes.NewExtensionObject(1, s.cfg.UserIdentityToken),
		UserTokenSignature:         s.cfg.UserTokenSignature,
	}
	return s.sechan.send(req, nil)
}

func (s *Session) handleActivateSessionResponse() error {
	svc, err := s.sechan.recv()
	if err != nil {
		return err
	}
	resp, ok := svc.(*services.ActivateSessionResponse)
	if !ok {
		return fmt.Errorf("invalid response. Got %T, want ActivateSessionResponse", svc)
	}

	for _, result := range resp.Results {
		if result != 0 {
			return fmt.Errorf("rejected")
		}
	}
	return nil
}
