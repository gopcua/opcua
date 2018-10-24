package codectest

import (
	"reflect"
	"testing"

	"github.com/wmnsk/gopcua"

	"github.com/pascaldekloe/goe/verify"
)

// Case describes a test case for a encoding and decoding an
// object from bytes.
type Case struct {
	Name   string
	Struct interface{}
	Bytes  []byte
}

// Run tests encoding, decoding and length calclulation for the given
// object.
func Run(t *testing.T, cases []Case) {
	t.Helper()

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Run("decode", func(t *testing.T) {
				// create a new instance of the same type as c.Struct
				typ := reflect.ValueOf(c.Struct).Type()
				var v reflect.Value
				switch typ.Kind() {
				case reflect.Ptr:
					v = reflect.New(typ.Elem()) // typ: *struct, v: *struct
				case reflect.Slice:
					v = reflect.New(typ) // typ: []x, v: *[]x
				default:
					t.Fatalf("%T is not a pointer or a slice", c.Struct)
				}

				if err := gopcua.Decode(c.Bytes, v.Interface()); err != nil {
					t.Fatal(err)
				}

				// if v is a *[]x we need to dereference it before comparing it.
				if typ.Kind() == reflect.Slice {
					v = v.Elem()
				}
				verify.Values(t, "", v.Interface(), c.Struct)
			})

			t.Run("encode", func(t *testing.T) {
				b, err := gopcua.Encode(c.Struct)
				if err != nil {
					t.Fatal(err)
				}
				verify.Values(t, "", b, c.Bytes)
			})
		})
	}
}
