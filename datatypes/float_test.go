// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"bytes"
	"math"
	"testing"

	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestFloat(t *testing.T) {
	cases := []codectest.Case{
		{
			Name:   "Normal",
			Struct: NewFloat(5.00078),
			Bytes:  []byte{0x64, 0x06, 0xa0, 0x40},
		},
	}
	codectest.Run(t, cases, func(b []byte) (codectest.S, error) {
		return DecodeFloat(b)
	})
}

// compare NaN in a separate test since the
// decode test will always fail with a NaN value
// since f != f for a NaN float.
func TestFloatNaN(t *testing.T) {
	qnan32 := []byte{0x00, 0x00, 0xc0, 0xff}
	t.Run("decode silent nan", func(t *testing.T) {
		f, err := DecodeFloat(qnan32)
		if err != nil {
			t.Fatal(err)
		}
		if !math.IsNaN(float64(f.Value)) {
			t.Fatal("should be NaN")
		}
	})
	t.Run("encode nan", func(t *testing.T) {
		f := &Float{float32(math.NaN())}
		b, err := f.Serialize()
		if err != nil {
			t.Fatal(err)
		}
		if got, want := b, qnan32; !bytes.Equal(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})
}
