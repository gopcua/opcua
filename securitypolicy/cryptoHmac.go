// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import (
	"crypto"
	"crypto/hmac"
	"errors"
)

func computeHmac(hash crypto.Hash, secret []byte) func(input []byte) ([]byte, error) {
	return func(input []byte) ([]byte, error) {
		h := hmac.New(hash.New, secret)
		h.Write(input)

		return h.Sum(nil), nil
	}
}

func verifyHmac(hash crypto.Hash, secret []byte) func(msg, signature []byte) error {
	signatureFunc := computeHmac(hash, secret)

	return func(msg, signature []byte) error {
		var err error

		sig, err := signatureFunc(msg)
		if err != nil {
			return err
		}

		if !hmac.Equal(sig, signature) {
			err = errors.New("signature validation failed")
		}

		return err
	}
}
