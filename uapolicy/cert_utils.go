// Copyright 2018-2019 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uapolicy

import (
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"fmt"

	"github.com/gopcua/opcua/debug"
)

// Thumbprint returns the thumbprint of the first DER-encoded certificate.
// If c contains a certificate chain, only the first (leaf) certificate is used.
func Thumbprint(c []byte) []byte {
	certs, err := x509.ParseCertificates(c)
	if err != nil || len(certs) == 0 {
		// fallback: hash the raw bytes if parsing fails
		thumbprint := sha1.Sum(c)
		return thumbprint[:]
	}

	thumbprint := sha1.Sum(certs[0].Raw)
	return thumbprint[:]
}

// ParseCertificate parses the first DER-encoded certificate from c.
// It handles the case where a server returns a certificate chain
// (multiple concatenated DER-encoded certificates) by returning only
// the first (leaf) certificate.
func ParseCertificate(c []byte) (*x509.Certificate, error) {
	certs, err := x509.ParseCertificates(c)
	if err != nil {
		return nil, err
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("uapolicy: no certificates found")
	}

	debug.Printf("uapolicy: parsed certificate: Subject=%s, Issuer=%s, SerialNumber=%s", certs[0].Subject, certs[0].Issuer, certs[0].SerialNumber)
	return certs[0], nil
}

// PublicKey returns the RSA PublicKey from a DER-encoded certificate
func PublicKey(c []byte) (*rsa.PublicKey, error) {
	cert, err := ParseCertificate(c)
	if err != nil {
		return nil, err
	}

	return cert.PublicKey.(*rsa.PublicKey), nil
}
