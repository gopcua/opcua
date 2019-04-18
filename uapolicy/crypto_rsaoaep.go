package uapolicy

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"

	// Force compilation of required hashing algorithms, although we don't directly use the packages
	_ "crypto/sha1"
	_ "crypto/sha256"
)

// messageLen = (keyLenBits / 8) - 2*(hashLenBits / 8) - 2
// paddingLen = keyLen - messageLen
//            = 2*hashLenBytes + 2
const (
	RSAOAEPMinPaddingSHA1   = (2 * 20) + 2
	RSAOAEPMinPaddingSHA256 = (2 * 64) + 2
)

type RSAOAEP struct {
	Hash       crypto.Hash
	PublicKey  *rsa.PublicKey
	PrivateKey *rsa.PrivateKey
}

func (a *RSAOAEP) Decrypt(src []byte) ([]byte, error) {
	rng := rand.Reader

	var plaintext []byte

	blockSize := a.PrivateKey.PublicKey.Size()
	srcRemaining := len(src)
	start := 0

	for srcRemaining > 0 {
		end := start + blockSize
		if end > len(src) {
			end = len(src)
		}

		p, err := rsa.DecryptOAEP(a.Hash.New(), rng, a.PrivateKey, src[start:end], nil)
		if err != nil {
			return nil, err
		}

		plaintext = append(plaintext, p...)
		start = end
		srcRemaining = len(src) - start
	}

	return plaintext, nil
}

func (a *RSAOAEP) Encrypt(src []byte) ([]byte, error) {
	rng := rand.Reader

	var ciphertext []byte

	maxBlock := a.PublicKey.Size() - RSAOAEPMinPadding(a.Hash)
	srcRemaining := len(src)
	start := 0
	for srcRemaining > 0 {
		end := start + maxBlock
		if end > len(src) {
			end = len(src)
		}

		c, err := rsa.EncryptOAEP(a.Hash.New(), rng, a.PublicKey, src[start:end], nil)
		if err != nil {
			return nil, err
		}

		ciphertext = append(ciphertext, c...)
		start = end
		srcRemaining = len(src) - start
	}

	return ciphertext, nil
}

func RSAOAEPMinPadding(hash crypto.Hash) int {
	// messageLen = (keyLenBits / 8) - 2*(hashLenBits / 8) - 2
	// paddingLen = keyLen - messageLen
	//            = 2*hashLenBytes + 2
	var hLen int
	switch hash {
	case crypto.SHA1:
		hLen = 20
	case crypto.SHA256:
		hLen = 64
	}

	return (2 * hLen) + 2
}
