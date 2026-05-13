package server

import (
	"context"
	mrand "math/rand"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/ualog"
)

type session struct {
	cfg sessionConfig

	ID                *ua.NodeID
	AuthTokenID       *ua.NodeID
	serverNonce       []byte
	remoteCertificate []byte

	PublishRequests chan PubReq
}

type sessionConfig struct {
	sessionTimeout time.Duration
}

type sessionBroker struct {
	// mu protects concurrent modification of s
	mu sync.Mutex

	// s contains all sessions watched by the session broker
	s map[string]*session
}

func newSessionBroker() *sessionBroker {
	return &sessionBroker{
		s: make(map[string]*session),
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

func (sb *sessionBroker) Close(ctx context.Context, authToken *ua.NodeID) error {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	if sb.s[authToken.String()] == nil {
		ualog.Warn(ctx, "unable to close session",
			ualog.String(ualog.ErrorKey, "error looking up session"),
			ualog.Any("token", authToken),
		)
	}
	delete(sb.s, authToken.String())

	return nil
}

func (sb *sessionBroker) Session(ctx context.Context, authToken *ua.NodeID) *session {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	s := sb.s[authToken.String()]
	if s == nil {
		ualog.Warn(ctx, "unable to lookup session", ualog.Any("token", authToken))
	}

	return s
}
