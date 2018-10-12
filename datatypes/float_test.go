// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package datatypes

import (
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Testing floating point numbers
// https://www.juliaferraioli.com/blog/2018/06/golang-testing-floats/

// Approximate equality for floats can be handled by defining a custom comparer on floats
// that determines two values to be equal if they are within some range of each other.
// https://godoc.org/github.com/google/go-cmp/cmp#example-Option--ApproximateFloats
var opt = cmp.Comparer(func(x, y float32) bool {
	a := float64(x)
	b := float64(y)
	delta := math.Abs(a - b)
	mean := math.Abs(a+b) / 2.0
	return delta/mean < 0.00001
})

var floatCases = []struct {
	description string
	structured  *Float
	serialized  []byte
}{
	{
		"normal",
		NewFloat(5.00078),
		[]byte{
			0x64, 0x06, 0xa0, 0x40,
		},
	},
}

func TestDecodeFloat(t *testing.T) {
	for _, c := range floatCases {
		got, err := DecodeFloat(c.serialized)
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.structured, opt); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestSerializeFloat(t *testing.T) {
	for _, c := range floatCases {
		got, err := c.structured.Serialize()
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(got, c.serialized); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}

func TestFloatLen(t *testing.T) {
	for _, c := range floatCases {
		got := c.structured.Len()

		if diff := cmp.Diff(got, len(c.serialized)); diff != "" {
			t.Errorf("%s failed\n%s", c.description, diff)
		}
	}
}
