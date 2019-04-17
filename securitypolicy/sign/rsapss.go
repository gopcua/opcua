package sign

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"

	// Force compilation of required hashing algorithms, although we don't directly use the packages
	_ "crypto/sha1"
	_ "crypto/sha256"
)

type RSAPSS struct {
	Hash       crypto.Hash
	PublicKey  *rsa.PublicKey
	PrivateKey *rsa.PrivateKey
}

func (s *RSAPSS) Signature(msg []byte) ([]byte, error) {
	rng := rand.Reader

	h := s.Hash.New()
	h.Write(msg)
	hashed := h.Sum(nil)

	return rsa.SignPSS(rng, s.PrivateKey, s.Hash, hashed[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
}

func (s *RSAPSS) Verify(msg, signature []byte) error {
	h := s.Hash.New()
	h.Write(msg)
	hashed := h.Sum(nil)
	return rsa.VerifyPSS(s.PublicKey, s.Hash, hashed[:], signature, nil)
}
