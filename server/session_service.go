package server

import (
	"crypto/rand"
	"log"
	"strings"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

const (
	sessionTimeoutMin     = 100            // 100ms
	sessionTimeoutMax     = 30 * 60 * 1000 // 30 minutes
	sessionTimeoutDefault = 60 * 1000      // 60s

	sessionNonceLength = 32
)

// SessionService implements the Session Service Set.
//
// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.6
type SessionService struct {
	srv *Server
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.6.2
func (s *SessionService) CreateSession(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.CreateSessionRequest](r)
	if err != nil {
		return nil, err
	}

	// New session
	sess := s.srv.sb.NewSession()

	// Ensure session timeout is reasonable
	sess.cfg.sessionTimeout = time.Duration(req.RequestedSessionTimeout) * time.Millisecond
	if sess.cfg.sessionTimeout > sessionTimeoutMax || sess.cfg.sessionTimeout < sessionTimeoutMin {
		sess.cfg.sessionTimeout = sessionTimeoutDefault
	}

	nonce := make([]byte, sessionNonceLength)
	if _, err := rand.Read(nonce); err != nil {
		log.Printf("error creating session nonce")
		return nil, ua.StatusBadInternalError
	}
	sess.serverNonce = nonce
	sess.remoteCertificate = req.ClientCertificate

	sig, alg, err := sc.NewSessionSignature(req.ClientCertificate, req.ClientNonce)
	if err != nil {
		log.Printf("error creating session signature")
		return nil, ua.StatusBadInternalError
	}

	matching_endpoints := make([]*ua.EndpointDescription, 0)
	reqTrimmedURL, _ := strings.CutSuffix(req.EndpointURL, "/")
	for i := range s.srv.endpoints {
		ep := s.srv.endpoints[i]
		epTrimmedURL, _ := strings.CutSuffix(ep.EndpointURL, "/")
		if epTrimmedURL == reqTrimmedURL {
			matching_endpoints = append(matching_endpoints, ep)
		}
	}

	response := &ua.CreateSessionResponse{
		ResponseHeader:        responseHeader(req.RequestHeader.RequestHandle, ua.StatusOK),
		SessionID:             sess.ID,
		AuthenticationToken:   sess.AuthTokenID,
		RevisedSessionTimeout: float64(sess.cfg.sessionTimeout / time.Millisecond),
		MaxRequestMessageSize: 0, // Not used
		ServerSignature: &ua.SignatureData{
			Signature: sig,
			Algorithm: alg,
		},
		ServerCertificate: s.srv.cfg.certificate,
		ServerNonce:       nonce,
		ServerEndpoints:   matching_endpoints,
	}

	return response, nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.6.3
func (s *SessionService) ActivateSession(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.ActivateSessionRequest](r)
	if err != nil {
		return nil, err
	}

	sess := s.srv.sb.Session(req.RequestHeader.AuthenticationToken)
	if sess == nil {
		return nil, ua.StatusBadSessionIDInvalid
	}

	err = sc.VerifySessionSignature(sess.remoteCertificate, sess.serverNonce, req.ClientSignature.Signature)
	if err != nil {
		if s.srv.cfg.logger != nil {
			s.srv.cfg.logger.Warn("error verifying session signature with nonce: %s", err)
		}
		return nil, ua.StatusBadSecurityChecksFailed
	}

	nonce := make([]byte, sessionNonceLength)
	if _, err := rand.Read(nonce); err != nil {
		log.Printf("error creating session nonce")
		return nil, ua.StatusBadInternalError
	}
	sess.serverNonce = nonce

	response := &ua.ActivateSessionResponse{
		ResponseHeader: responseHeader(req.RequestHeader.RequestHandle, ua.StatusOK),
		ServerNonce:    nonce,
		// Results:         []ua.StatusCode{},
		// DiagnosticInfos: []*ua.DiagnosticInfo{},
	}

	return response, nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.6.4
func (s *SessionService) CloseSession(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.CloseSessionRequest](r)
	if err != nil {
		return nil, err
	}

	err = s.srv.sb.Close(req.RequestHeader.AuthenticationToken)
	if err != nil {
		return nil, ua.StatusBadSessionIDInvalid
	}

	//TODO: deal with 'delete subscriptions' field in request
	response := &ua.CloseSessionResponse{
		ResponseHeader: responseHeader(req.RequestHeader.RequestHandle, ua.StatusOK),
	}

	return response, nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.6.5
func (s *SessionService) Cancel(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.CancelRequest](r)
	if err != nil {
		return nil, err
	}
	return serviceUnsupported(req.RequestHeader), nil
}
