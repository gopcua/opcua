// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
)

func hmacSha1(secret []byte) func(input []byte) []byte {
	return func(input []byte) []byte {
		h := hmac.New(sha1.New, secret)
		h.Write(input)

		return h.Sum(nil)
	}
}

func verifyHmacSha1(secret []byte) func(msg, signature []byte) error {
	signatureFunc := hmacSha1(secret)
	return func(msg, signature []byte) error {
		var err error

		if !hmac.Equal(signatureFunc(msg), signature) {
			err = errors.New("signature validation failed")
		}

		return err
	}
}

func hmacSha256(secret []byte) func(input []byte) []byte {
	return func(input []byte) []byte {
		h := hmac.New(sha256.New, secret)
		h.Write(input)

		return h.Sum(nil)
	}
}

func verifyHmacSha256(secret []byte) func(msg, signature []byte) error {
	signatureFunc := hmacSha256(secret)
	return func(msg, signature []byte) error {
		var err error

		if !hmac.Equal(signatureFunc(msg), signature) {
			err = errors.New("signature validation failed")
		}

		return err
	}
}
