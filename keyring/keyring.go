// Copyright 2018-2019 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// Package keyring implements a simple key-value store for certificates,
// public and private RSA keys, and the certificate thumbprints.
// It is meant to be an internal use only package to the gopcua library.
package keyring

import (
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"sync"
)

type privateKeys struct {
	thumbprint []byte
	cert       *x509.Certificate
	key        *rsa.PrivateKey
}

var localKeys sync.Map

// Add adds a new key to the keyring
func Add(c *x509.Certificate, k *rsa.PrivateKey) []byte {
	thumbprint := sha1.Sum(c.Raw)
	t := hex.EncodeToString(thumbprint[:])

	pk, _ := localKeys.LoadOrStore(t, privateKeys{
		thumbprint: thumbprint[:],
		cert:       c,
		key:        k,
	})

	return pk.(privateKeys).thumbprint
}

// Certificate returns the stored certificate for a given key thumbprint
func Certificate(thumbprint []byte) (*x509.Certificate, error) {
	t := hex.EncodeToString(thumbprint)

	pk, ok := localKeys.Load(t)
	if !ok {
		return nil, errors.New("could not fetch certificate, unknown thumbprint: " + t)
	}

	return pk.(privateKeys).cert, nil
}

// PrivateKey returns the stored RSA private key for a given thumbprint
func PrivateKey(thumbprint []byte) (*rsa.PrivateKey, error) {
	t := hex.EncodeToString(thumbprint)

	pk, ok := localKeys.Load(t)
	if !ok {
		return nil, errors.New("could not fetch private key, unknown thumbprint: " + t)
	}

	if pk.(privateKeys).key == nil {
		return nil, errors.New("thumbprint stored without private key: " + t)
	}
	return pk.(privateKeys).key, nil
}

// PublicKey returns the stored RSA public key for a given thumbprint
func PublicKey(thumbprint []byte) (*rsa.PublicKey, error) {
	t := hex.EncodeToString(thumbprint)

	pk, ok := localKeys.Load(t)
	if !ok {
		return nil, errors.New("could not fetch public key, unknown thumbprint")
	}

	return pk.(privateKeys).cert.PublicKey.(*rsa.PublicKey), nil
}

// Thumbprint returns the thumbprint of a DER-encoded certificate
func Thumbprint(c []byte) []byte {
	thumbprint := sha1.Sum(c)

	return thumbprint[:]
}

// Thumbprints returns a slice of thumbprints for all keys in the keyring
func Thumbprints() [][]byte {
	tt := make([][]byte, 0)

	localKeys.Range(func(k, v interface{}) bool {
		tt = append(tt, v.(privateKeys).thumbprint)
		return true
	})

	return tt
}
