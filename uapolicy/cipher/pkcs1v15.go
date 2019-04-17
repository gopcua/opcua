package cipher

import (
	"crypto/rand"
	"crypto/rsa"

	// Force compilation of required hashing algorithms, although we don't directly use the packages
	_ "crypto/sha1"
	_ "crypto/sha256"
)

const PKCS1v15MinPadding = 11

type PKCS1v15 struct {
	PublicKey  *rsa.PublicKey
	PrivateKey *rsa.PrivateKey
}

func (c *PKCS1v15) Decrypt(src []byte) ([]byte, error) {
	rng := rand.Reader

	var plaintext []byte

	blockSize := c.PrivateKey.PublicKey.Size()
	srcRemaining := len(src)
	start := 0

	for srcRemaining > 0 {
		end := start + blockSize
		if end > len(src) {
			end = len(src)
		}

		p, err := rsa.DecryptPKCS1v15(rng, c.PrivateKey, src[start:end])
		if err != nil {
			return nil, err
		}

		plaintext = append(plaintext, p...)
		start = end
		srcRemaining = len(src) - start
	}

	return plaintext, nil
}

func (c *PKCS1v15) Encrypt(src []byte) ([]byte, error) {
	rng := rand.Reader

	var ciphertext []byte

	maxBlock := c.PublicKey.Size() - PKCS1v15MinPadding
	srcRemaining := len(src)
	start := 0
	for srcRemaining > 0 {
		end := start + maxBlock
		if end > len(src) {
			end = len(src)
		}

		c, err := rsa.EncryptPKCS1v15(rng, c.PublicKey, src[start:end])
		if err != nil {
			return nil, err
		}

		ciphertext = append(ciphertext, c...)
		start = end
		srcRemaining = len(src) - start
	}

	return ciphertext, nil
}
