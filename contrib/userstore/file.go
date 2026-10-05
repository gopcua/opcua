// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package userstore

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// DefaultCost is the bcrypt cost used for new password hashes.
const DefaultCost = 12

// MaxPasswordLength is the longest password bcrypt can hash without silently
// truncating it.
const MaxPasswordLength = 72

// File is the on-disk format of users.yaml.
type File struct {
	// Users keyed by username.
	Users map[string]*User `yaml:"users"`
	// Certificates are trusted user certificates, identified by thumbprint.
	Certificates []*Certificate `yaml:"certificates,omitempty"`
}

// User is a username/password account.
type User struct {
	// Password is a bcrypt hash. Never a plain text password.
	Password string `yaml:"password"`
	// Roles are free-form role names (e.g. viewer, operator, admin).
	Roles []string `yaml:"roles,omitempty,flow"`
	// Disabled blocks the login but keeps the entry.
	Disabled bool `yaml:"disabled,omitempty"`
}

// Certificate is a trusted X509 user certificate.
type Certificate struct {
	// Thumbprint is the lowercase hex SHA-1 of the DER certificate.
	Thumbprint string `yaml:"thumbprint"`
	// Name is a human readable label.
	Name string `yaml:"name,omitempty"`
	// Roles are free-form role names.
	Roles []string `yaml:"roles,omitempty,flow"`
	// Disabled blocks the login but keeps the entry.
	Disabled bool `yaml:"disabled,omitempty"`
}

// Thumbprint returns the lowercase hex SHA-1 thumbprint of a DER certificate,
// the identifier OPC UA uses for certificates.
func Thumbprint(der []byte) string {
	sum := sha1.Sum(der)
	return hex.EncodeToString(sum[:])
}

// normalizeThumbprint accepts upper case and colon/space separated forms.
func normalizeThumbprint(s string) (string, error) {
	t := strings.ToLower(strings.NewReplacer(":", "", " ", "").Replace(s))
	if len(t) != 2*sha1.Size {
		return "", fmt.Errorf("thumbprint %q: want %d hex characters", s, 2*sha1.Size)
	}
	if _, err := hex.DecodeString(t); err != nil {
		return "", fmt.Errorf("thumbprint %q: not hex", s)
	}
	return t, nil
}

// ValidateUsername checks that name can be used as a username.
func ValidateUsername(name string) error {
	if name == "" {
		return errors.New("username must not be empty")
	}
	if len(name) > 256 {
		return errors.New("username too long")
	}
	if strings.TrimSpace(name) != name {
		return errors.New("username must not start or end with whitespace")
	}
	for _, r := range name {
		if r < 0x20 || r == 0x7f {
			return errors.New("username must not contain control characters")
		}
	}
	return nil
}

// HashPassword returns a bcrypt hash of password. cost <= 0 uses DefaultCost.
// Empty passwords and passwords longer than MaxPasswordLength bytes are
// rejected.
func HashPassword(password string, cost int) (string, error) {
	if password == "" {
		return "", errors.New("password must not be empty")
	}
	if len(password) > MaxPasswordLength {
		return "", fmt.Errorf("password longer than %d bytes", MaxPasswordLength)
	}
	if cost <= 0 {
		cost = DefaultCost
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// Validate checks the file for invalid entries and normalizes thumbprints.
func (f *File) Validate() error {
	if f.Users == nil {
		f.Users = map[string]*User{}
	}
	for name, u := range f.Users {
		if err := ValidateUsername(name); err != nil {
			return fmt.Errorf("user %q: %w", name, err)
		}
		if u == nil {
			return fmt.Errorf("user %q: empty entry", name)
		}
		if _, err := bcrypt.Cost([]byte(u.Password)); err != nil {
			return fmt.Errorf("user %q: password is not a bcrypt hash", name)
		}
	}
	seen := map[string]bool{}
	for i, c := range f.Certificates {
		if c == nil {
			return fmt.Errorf("certificate %d: empty entry", i)
		}
		t, err := normalizeThumbprint(c.Thumbprint)
		if err != nil {
			return fmt.Errorf("certificate %d: %w", i, err)
		}
		if seen[t] {
			return fmt.Errorf("certificate %d: duplicate thumbprint %s", i, t)
		}
		seen[t] = true
		c.Thumbprint = t
	}
	return nil
}

// Certificate returns the entry with the given thumbprint, or nil.
func (f *File) Certificate(thumbprint string) *Certificate {
	t, err := normalizeThumbprint(thumbprint)
	if err != nil {
		return nil
	}
	for _, c := range f.Certificates {
		if c.Thumbprint == t {
			return c
		}
	}
	return nil
}

// Usernames returns the sorted list of usernames.
func (f *File) Usernames() []string {
	names := make([]string, 0, len(f.Users))
	for n := range f.Users {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Parse decodes and validates a users.yaml document.
func Parse(b []byte) (*File, error) {
	f := &File{}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	// io.EOF: an empty document is an empty file
	if err := dec.Decode(f); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("parse users file: %w", err)
	}
	if err := f.Validate(); err != nil {
		return nil, err
	}
	return f, nil
}

// Load reads and validates a users file.
func Load(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f, err := Parse(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return f, nil
}

const fileHeader = `# gopcua users file. Manage with: opcua-users -f <file> add|passwd|del|list ...
# Passwords are bcrypt hashes. Do not put plain text passwords here.
`

// Save validates f and writes it atomically (temp file + rename) with mode 0600.
func (f *File) Save(path string) error {
	if err := f.Validate(); err != nil {
		return err
	}
	b, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(fileHeader); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// lockTimeout is how long Update waits for another writer.
var lockTimeout = 5 * time.Second

// Update locks path, loads it (an empty File if it does not exist yet), calls
// fn and saves the result. Concurrent Update calls on the same file, including
// from other processes, are serialized with a <path>.lock file.
func Update(path string, fn func(*File) error) error {
	unlock, err := lockFile(path + ".lock")
	if err != nil {
		return err
	}
	defer unlock()

	f, err := Load(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		f = &File{Users: map[string]*User{}}
	case err != nil:
		return err
	}
	if err := fn(f); err != nil {
		return err
	}
	return f.Save(path)
}

func lockFile(path string) (func(), error) {
	deadline := time.Now().Add(lockTimeout)
	for {
		lf, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			fmt.Fprintf(lf, "%d\n", os.Getpid())
			lf.Close()
			return func() { os.Remove(path) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("users file is locked by another process (remove %s if it is stale)", path)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
