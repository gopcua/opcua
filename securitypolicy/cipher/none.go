package cipher

const (
	NoneBlockSize  = 1
	NoneMinPadding = 0
)

type None struct{}

func (c *None) Decrypt(src []byte) ([]byte, error) {
	var b []byte
	return append(b, src...), nil
}

func (c *None) Encrypt(src []byte) ([]byte, error) {
	var b []byte
	return append(b, src...), nil
}
