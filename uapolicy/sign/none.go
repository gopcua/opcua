package sign

type None struct{}

func (s *None) Signature(msg []byte) ([]byte, error) {
	return make([]byte, 0), nil
}

func (s *None) Verify(msg, signature []byte) error {
	return nil
}
