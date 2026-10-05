// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// Package userstore is a file-backed user store for the gopcua server.
//
// Users and trusted user certificates are kept in a YAML file (users.yaml).
// Passwords are stored as bcrypt hashes. The file is managed with the
// opcua-users command and reloaded automatically when it changes, so users
// can be added, disabled or have their password changed without a restart.
//
//	store, err := userstore.Open("users.yaml")
//	if err != nil { ... }
//	defer store.Close()
//
//	srv := server.New(
//		...,
//		server.UserNameAuth(store.UserNameAuthenticator()),
//		server.X509Auth(store.X509Authenticator()),
//	)
package userstore

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/gopcua/opcua/server"
)

// Options configure a Store. The zero value uses the defaults.
type Options struct {
	// ReloadInterval is how often the file is checked for changes.
	// Default 2s. Negative disables automatic reloading (Reload still works).
	ReloadInterval time.Duration

	// CacheTTL is how long a successful username/password check is cached in
	// memory, so reconnecting clients do not pay the bcrypt cost every time.
	// Default 5m. Negative disables the cache.
	CacheTTL time.Duration

	// MaxFailures is the number of failed logins per user within
	// FailureWindow after which the user is locked out for LockoutDuration.
	// Defaults 5, 1m and 1m. A failure shortly after a lockout locks the
	// user again for twice as long, up to 15m. Negative MaxFailures disables
	// throttling.
	MaxFailures     int
	FailureWindow   time.Duration
	LockoutDuration time.Duration

	// Logf logs reload errors and lockouts. Default log.Printf.
	Logf func(format string, args ...any)

	now func() time.Time
}

const maxLockout = 15 * time.Minute

// ErrLockedOut is returned while a user is locked out after too many failed
// logins. The server reports it as Bad_UserAccessDenied like any other
// failure, so clients cannot tell a lockout from a wrong password.
var ErrLockedOut = errors.New("too many failed logins, try again later")

// Store is a users.yaml backed user store. It is safe for concurrent use.
type Store struct {
	path string
	opts Options

	mu      sync.RWMutex
	file    *File
	modTime time.Time
	size    int64

	// cache of successful logins: hmac(user, password) -> expiry
	cacheMu  sync.Mutex
	cacheKey []byte
	cache    map[[32]byte]time.Time // expiry

	failMu   sync.Mutex
	failures map[string]*failState

	// dummyHash is compared against for unknown users, so a lookup takes
	// about as long as for a real user.
	dummyHash []byte

	stop chan struct{}
	done chan struct{}
}

type failState struct {
	count       int
	first       time.Time
	lockedUntil time.Time
	lockout     time.Duration
}

// Open loads the users file at path and, unless disabled, starts watching it
// for changes. The file must exist; create it with
// `opcua-users -f <path> add <user>`.
func Open(path string, opts ...Options) (*Store, error) {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.ReloadInterval == 0 {
		o.ReloadInterval = 2 * time.Second
	}
	if o.CacheTTL == 0 {
		o.CacheTTL = 5 * time.Minute
	}
	if o.MaxFailures == 0 {
		o.MaxFailures = 5
	}
	if o.FailureWindow == 0 {
		o.FailureWindow = time.Minute
	}
	if o.LockoutDuration == 0 {
		o.LockoutDuration = time.Minute
	}
	if o.Logf == nil {
		o.Logf = log.Printf
	}
	if o.now == nil {
		o.now = time.Now
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	dummy, err := bcrypt.GenerateFromPassword(key[:16], DefaultCost)
	if err != nil {
		return nil, err
	}

	s := &Store{
		path:      path,
		opts:      o,
		cacheKey:  key,
		cache:     map[[32]byte]time.Time{},
		failures:  map[string]*failState{},
		dummyHash: dummy,
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
	}
	if err := s.Reload(); err != nil {
		return nil, err
	}
	if o.ReloadInterval > 0 {
		go s.watch()
	} else {
		close(s.done)
	}
	return s, nil
}

// Close stops watching the file.
func (s *Store) Close() error {
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
	<-s.done
	return nil
}

// Path returns the users file path.
func (s *Store) Path() string { return s.path }

// Reload re-reads the users file. If the file is invalid the previous
// version stays active and the error is returned.
func (s *Store) Reload() error {
	fi, err := os.Stat(s.path)
	if err != nil {
		return err
	}
	f, err := Load(s.path)
	if err != nil {
		return err
	}
	if fi.Mode().Perm()&0o077 != 0 {
		s.opts.Logf("userstore: %s is accessible by other users (mode %v), chmod 600 recommended", s.path, fi.Mode().Perm())
	}
	s.mu.Lock()
	s.file = f
	s.modTime = fi.ModTime()
	s.size = fi.Size()
	s.mu.Unlock()

	// passwords, roles or disabled flags may have changed
	s.cacheMu.Lock()
	clear(s.cache)
	s.cacheMu.Unlock()
	return nil
}

func (s *Store) watch() {
	defer close(s.done)
	t := time.NewTicker(s.opts.ReloadInterval)
	defer t.Stop()
	var lastErr string
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
		}
		fi, err := os.Stat(s.path)
		if err != nil {
			if err.Error() != lastErr {
				s.opts.Logf("userstore: %s, keeping the last loaded users", err)
				lastErr = err.Error()
			}
			continue
		}
		s.mu.RLock()
		changed := !fi.ModTime().Equal(s.modTime) || fi.Size() != s.size
		s.mu.RUnlock()
		if !changed {
			continue
		}
		if err := s.Reload(); err != nil {
			if err.Error() != lastErr {
				s.opts.Logf("userstore: reload failed, keeping the last loaded users: %s", err)
				lastErr = err.Error()
			}
			continue
		}
		lastErr = ""
		s.opts.Logf("userstore: reloaded %s", s.path)
	}
}

// User returns a copy of the user entry, or nil if it does not exist.
func (s *Store) User(name string) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u := s.file.Users[name]
	if u == nil {
		return nil
	}
	c := *u
	c.Roles = slices.Clone(u.Roles)
	return &c
}

// Roles returns the roles of a server identity: a username, a certificate
// listed in the users file, or nil for anonymous and unknown identities.
func (s *Store) Roles(id *server.Identity) []string {
	if id == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if id.UserName != "" {
		if u := s.file.Users[id.UserName]; u != nil && !u.Disabled {
			return slices.Clone(u.Roles)
		}
		return nil
	}
	if id.Certificate != nil {
		if c := s.file.Certificate(Thumbprint(id.Certificate.Raw)); c != nil && !c.Disabled {
			return slices.Clone(c.Roles)
		}
	}
	return nil
}

// Authenticate checks a username and password.
func (s *Store) Authenticate(username, password string) error {
	now := s.opts.now()
	if err := s.checkLockout(username, now); err != nil {
		return err
	}

	ck := s.cacheKeyFor(username, password)
	if s.opts.CacheTTL > 0 {
		s.cacheMu.Lock()
		exp, ok := s.cache[ck]
		s.cacheMu.Unlock()
		if ok && now.Before(exp) {
			s.clearFailures(username)
			return nil
		}
	}

	s.mu.RLock()
	u := s.file.Users[username]
	var hash []byte
	disabled := false
	if u != nil {
		hash = []byte(u.Password)
		disabled = u.Disabled
	}
	s.mu.RUnlock()

	if u == nil {
		// same work as for a real user, so usernames can't be probed by timing
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		s.recordFailure(username, now)
		return server.ErrAccessDenied
	}
	if err := bcrypt.CompareHashAndPassword(hash, []byte(password)); err != nil {
		s.recordFailure(username, now)
		return server.ErrAccessDenied
	}
	if disabled {
		return fmt.Errorf("user is disabled: %w", server.ErrAccessDenied)
	}

	s.clearFailures(username)
	if s.opts.CacheTTL > 0 {
		s.cacheMu.Lock()
		if len(s.cache) > 10000 {
			for k, exp := range s.cache {
				if !now.Before(exp) {
					delete(s.cache, k)
				}
			}
		}
		s.cache[ck] = now.Add(s.opts.CacheTTL)
		s.cacheMu.Unlock()
	}
	return nil
}

// UserNameAuthenticator returns a server.UserNameAuthenticator backed by the
// store.
func (s *Store) UserNameAuthenticator() server.UserNameAuthenticator {
	return s.Authenticate
}

// AuthenticateCertificate accepts a user certificate whose thumbprint is
// listed in the users file and not disabled. The server has already verified
// that the client holds the certificate's private key. The validity period is
// checked here.
func (s *Store) AuthenticateCertificate(cert *x509.Certificate) error {
	if cert == nil {
		return server.ErrAccessDenied
	}
	now := s.opts.now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return fmt.Errorf("certificate not valid at %s: %w", now.Format(time.RFC3339), server.ErrAccessDenied)
	}
	s.mu.RLock()
	c := s.file.Certificate(Thumbprint(cert.Raw))
	s.mu.RUnlock()
	if c == nil {
		return fmt.Errorf("certificate %s is not trusted: %w", Thumbprint(cert.Raw), server.ErrAccessDenied)
	}
	if c.Disabled {
		return fmt.Errorf("certificate %s is disabled: %w", c.Thumbprint, server.ErrAccessDenied)
	}
	return nil
}

// X509Authenticator returns a server.X509Authenticator backed by the
// certificates section of the users file.
func (s *Store) X509Authenticator() server.X509Authenticator {
	return s.AuthenticateCertificate
}

func (s *Store) cacheKeyFor(user, pass string) [32]byte {
	m := hmac.New(sha256.New, s.cacheKey)
	// length-prefix the user so "ab"+"c" and "a"+"bc" differ
	fmt.Fprintf(m, "%d:%s", len(user), user)
	m.Write([]byte(pass))
	var k [32]byte
	copy(k[:], m.Sum(nil))
	return k
}

func (s *Store) checkLockout(user string, now time.Time) error {
	if s.opts.MaxFailures < 0 {
		return nil
	}
	s.failMu.Lock()
	defer s.failMu.Unlock()
	f := s.failures[user]
	if f != nil && now.Before(f.lockedUntil) {
		return ErrLockedOut
	}
	return nil
}

func (s *Store) recordFailure(user string, now time.Time) {
	if s.opts.MaxFailures < 0 {
		return
	}
	s.failMu.Lock()
	defer s.failMu.Unlock()
	f := s.failures[user]
	switch {
	case f == nil:
		f = &failState{first: now}
		s.failures[user] = f
	case f.lockout > 0 && !now.After(f.lockedUntil.Add(maxLockout)):
		// failing again shortly after a lockout: lock again, twice as long
		f.lockout = min(2*f.lockout, maxLockout)
		f.lockedUntil = now.Add(f.lockout)
		s.opts.Logf("userstore: user %q locked out for %s after repeated failed logins", user, f.lockout)
		return
	case f.lockout > 0 || now.Sub(f.first) > s.opts.FailureWindow:
		*f = failState{first: now}
	}
	f.count++
	if f.count < s.opts.MaxFailures {
		return
	}
	f.lockout = s.opts.LockoutDuration
	f.lockedUntil = now.Add(f.lockout)
	s.opts.Logf("userstore: user %q locked out for %s after %d failed logins", user, f.lockout, f.count)

	// bound memory when many unknown usernames are tried
	if len(s.failures) > 10000 {
		for k, v := range s.failures {
			if now.After(v.lockedUntil.Add(maxLockout)) && now.Sub(v.first) > s.opts.FailureWindow {
				delete(s.failures, k)
			}
		}
	}
}

func (s *Store) clearFailures(user string) {
	s.failMu.Lock()
	delete(s.failures, user)
	s.failMu.Unlock()
}
