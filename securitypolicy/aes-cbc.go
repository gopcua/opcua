// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package securitypolicy

import (
	"crypto/aes"
	"crypto/cipher"
)

func blockSizeAES() int {
	return aes.BlockSize
}

func decryptAES(keyLength int, iv, secret []byte) func(src []byte) []byte {
	return func(src []byte) []byte {
		paddedKey := make([]byte, keyLength/8)
		copy(paddedKey, secret)

		block, err := aes.NewCipher(secret)
		if err != nil {
			panic(err)
		}

		if len(src) < aes.BlockSize {
			panic("ciphertext too short") //FIXME
		}

		// CBC mode always works in whole blocks.
		if len(src)%aes.BlockSize != 0 {
			panic("ciphertext is not a multiple of the block size") //FIXME
		}

		mode := cipher.NewCBCDecrypter(block, iv)

		dst := make([]byte, len(src))

		mode.CryptBlocks(dst, src)
		return dst

	}
}

func encryptAES(keyLength int, iv, secret []byte) func(src []byte) []byte {
	return func(src []byte) []byte {
		paddedKey := make([]byte, keyLength/8)
		copy(paddedKey, secret)

		if len(src)%aes.BlockSize != 0 {
			panic("plaintext is not a multiple of the block size") //FIXME
		}

		block, err := aes.NewCipher(paddedKey)
		if err != nil {
			panic(err) //FIXME
		}

		mode := cipher.NewCBCEncrypter(block, iv)

		dst := make([]byte, len(src))
		mode.CryptBlocks(dst, src)

		return dst
	}
}
