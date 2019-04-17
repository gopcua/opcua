package sign

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
)

type PKCS1v15 struct {
	Hash       crypto.Hash
	PublicKey  *rsa.PublicKey
	PrivateKey *rsa.PrivateKey
}

func (s *PKCS1v15) Signature(msg []byte) ([]byte, error) {
	rng := rand.Reader

	h := s.Hash.New()
	h.Write(msg)
	hashed := h.Sum(nil)

	return rsa.SignPKCS1v15(rng, s.PrivateKey, s.Hash, hashed[:])
}

func (s *PKCS1v15) Verify(msg, signature []byte) error {
	h := s.Hash.New()
	h.Write(msg)
	hashed := h.Sum(nil)
	return rsa.VerifyPKCS1v15(s.PublicKey, s.Hash, hashed[:], signature)
}
