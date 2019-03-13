// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

func blockSizeAES() int {
	return aes.BlockSize
}

func minPaddingAES() int {
	return 0
}

func decryptAES(keyLength int, iv, secret []byte) func(src []byte) ([]byte, error) {
	return func(src []byte) ([]byte, error) {
		paddedKey := make([]byte, keyLength/8)
		copy(paddedKey, secret)

		block, err := aes.NewCipher(secret)
		if err != nil {
			return nil, err
		}

		if len(src) < aes.BlockSize {
			return nil, errors.New("ciphertext too short")
		}

		// CBC mode always works in whole blocks.
		if len(src)%aes.BlockSize != 0 {
			return nil, errors.New("ciphertext is not a multiple of the block size")
		}

		mode := cipher.NewCBCDecrypter(block, iv)

		dst := make([]byte, len(src))

		mode.CryptBlocks(dst, src)

		return dst, nil
	}
}

func encryptAES(keyLength int, iv, secret []byte) func(src []byte) ([]byte, error) {
	return func(src []byte) ([]byte, error) {
		paddedKey := make([]byte, keyLength/8)
		copy(paddedKey, secret)

		// CBC mode always works in whole blocks.
		if len(src)%aes.BlockSize != 0 {
			return nil, errors.New("plaintext is not a multiple of the block size")
		}

		block, err := aes.NewCipher(paddedKey)
		if err != nil {
			return nil, err
		}

		mode := cipher.NewCBCEncrypter(block, iv)

		dst := make([]byte, len(src))
		mode.CryptBlocks(dst, src)

		return dst, nil
	}
}
