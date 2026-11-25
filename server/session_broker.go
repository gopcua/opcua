package server

import (
	mrand "math/rand"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/gopcua/opcua/ua"
)

type session struct {
	cfg sessionConfig

	ID                *ua.NodeID
	AuthTokenID       *ua.NodeID
	serverNonce       []byte
	remoteCertificate []byte

	// activateMu serializes ActivateSession calls for this session, which run
	// concurrently to the message loop. It guards serverNonce after creation.
	activateMu sync.Mutex

	// mu guards activated and identity
	mu        sync.Mutex
	activated bool
	identity  *Identity

	PublishRequests chan PubReq
}

// Activated reports whether the session was successfully activated.
func (s *session) Activated() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activated
}

// Identity returns the user identity the session was activated with, or nil.
func (s *session) Identity() *Identity {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.identity
}

func (s *session) activate(id *Identity) {
	s.mu.Lock()
	s.activated = true
	s.identity = id
	s.mu.Unlock()
}

type sessionConfig struct {
	sessionTimeout time.Duration
}

type sessionBroker struct {
	// mu protects concurrent modification of s
	mu sync.Mutex

	// s contains all sessions watched by the session broker
	s      map[string]*session
	logger Logger
}

func newSessionBroker(logger Logger) *sessionBroker {
	return &sessionBroker{
		s:      make(map[string]*session),
		logger: logger,
	}
}

func (sb *sessionBroker) NewSession() *session {
	s := &session{
		ID:              ua.NewGUIDNodeID(1, uuid.New().String()),
		AuthTokenID:     ua.NewNumericNodeID(0, uint32(mrand.Int31())),
		PublishRequests: make(chan PubReq, 100),
	}

	sb.mu.Lock()
	sb.s[s.AuthTokenID.String()] = s
	sb.mu.Unlock()

	return s
}

func (sb *sessionBroker) Close(authToken *ua.NodeID) error {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	if sb.s[authToken.String()] == nil {
		if sb.logger != nil {
			sb.logger.Warn("sessionBroker.Close: error looking up session %v", authToken)
		}
	}
	delete(sb.s, authToken.String())

	return nil
}

func (sb *sessionBroker) Session(authToken *ua.NodeID) *session {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	s := sb.s[authToken.String()]
	if s == nil {
		if sb.logger != nil {
			sb.logger.Warn("sessionBroker.Session: error looking up session %v", authToken)
		}
	}

	return s
}
