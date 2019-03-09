package services

func flatten(b ...[]byte) []byte {
	var x []byte
	for _, buf := range b {
		x = append(x, buf...)
	}
	return x
}
