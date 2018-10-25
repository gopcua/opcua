package codectest

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

type S interface {
	Serialize() ([]byte, error)
	Len() int
}

// Case describes a test case for a encoding and decoding an
// object from bytes.
type Case struct {
	Name   string
	Struct S
	Bytes  []byte
}

// DecoderFunc creates an object from bytes.
type DecoderFunc func([]byte) (S, error)

// Run tests encoding, decoding and length calclulation for the given
// object.
func Run(t *testing.T, cases []Case, decode DecoderFunc) {
	t.Helper()

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Run("decode", func(t *testing.T) {
				v, err := decode(c.Bytes)
				if err != nil {
					t.Fatal(err)
				}

				if got, want := v, c.Struct; !cmp.Equal(got, want) {
					t.Fatal(cmp.Diff(got, want))
				}
			})

			t.Run("encode", func(t *testing.T) {
				b, err := c.Struct.Serialize()
				if err != nil {
					t.Fatal(err)
				}

				if got, want := b, c.Bytes; !cmp.Equal(got, want) {
					t.Fatal(cmp.Diff(got, want))
				}
			})

			t.Run("len", func(t *testing.T) {
				if got, want := c.Struct.Len(), len(c.Bytes); got != want {
					t.Fatalf("got %v want %v", got, want)
				}
			})
		})
	}
}
