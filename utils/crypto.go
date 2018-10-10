// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package utils

import (
	"crypto/sha1"
	"io/ioutil"
)

// NewThumbprintFromCert returns a thumbprint(SHA-1 hash) in byte slice from the DER-formatted X509 certificate file given.
func NewThumbprintFromCert(myCert string) ([]byte, error) {
	cert, err := ioutil.ReadFile(myCert)
	if err != nil {
		return nil, err
	}

	hash := sha1.Sum(cert)
	return hash[:], nil
}
