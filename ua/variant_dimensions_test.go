// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// The dimensions of a multi-dimensional array only have to be greater than
// zero and consistent with the array length (OPC 10000-6 §5.2.2.16).
// Decoding refuses more dimensions than MaxVariantArrayDimensions allows.
func TestDecodeRejectsMoreDimensionsThanTheLimit(t *testing.T) {
	smallStack(t, 8<<20)

	t.Run("a dimension per four bytes of message", func(t *testing.T) {
		const dims = 1 << 20
		b := multiDimensionalArrayBytes(TypeIDBoolean, []byte{1}, 1, dims)

		v := new(Variant)
		_, err := v.Decode(b)
		require.ErrorIs(t, err, StatusBadEncodingLimitsExceeded)
		require.Empty(t, v.ArrayDimensions())
	})

	t.Run("one dimension too many", func(t *testing.T) {
		b := multiDimensionalArrayBytes(TypeIDBoolean, []byte{1}, 1, MaxVariantArrayDimensions+1)

		_, err := new(Variant).Decode(b)
		require.ErrorIs(t, err, StatusBadEncodingLimitsExceeded)
	})
}

// An array with as many dimensions as the limit still decodes, to the same
// value as before the limit.
func TestDecodeSupportsAsManyDimensionsAsTheLimit(t *testing.T) {
	dims := MaxVariantArrayDimensions
	b := multiDimensionalArrayBytes(TypeIDBoolean, []byte{1}, 1, dims)

	v := new(Variant)
	_, err := v.Decode(b)
	require.NoError(t, err)

	expected := make([]int32, dims)
	for i := range expected {
		expected[i] = 1
	}
	require.Equal(t, expected, v.ArrayDimensions())

	// every dimension has a single element, so the value is one slice per
	// dimension, nested, with the array element at the bottom
	val := reflect.ValueOf(v.Value())
	depth := 0
	for val.Kind() == reflect.Slice {
		require.Equal(t, 1, val.Len())
		val = val.Index(0)
		depth++
	}
	require.Equal(t, dims, depth)
	require.Equal(t, true, val.Interface())
}

// A multi-dimensional array decodes to the value it encoded.
func TestDecodeMultiDimensionalArray(t *testing.T) {
	// dimensions [2,2,3], the values serialized with the lower rank
	// dimensions first, OPC 10000-6 §5.2.2.16
	val := [][][]uint32{
		{{1, 2, 3}, {4, 5, 6}},
		{{7, 8, 9}, {10, 11, 12}},
	}
	b, err := MustVariant(val).Encode()
	require.NoError(t, err)

	v := new(Variant)
	_, err = v.Decode(b)
	require.NoError(t, err)
	require.Equal(t, val, v.Value())
	require.Equal(t, []int32{2, 2, 3}, v.ArrayDimensions())
}

// multiDimensionalArrayBytes returns the encoding of an array Variant of the
// given built-in type with count dimensions of size elements each: the
// flattened values, then the array dimensions, OPC 10000-6 §5.2.2.16. The
// product of the dimensions must equal the number of values.
func multiDimensionalArrayBytes(typ TypeID, values []byte, size int32, count int) []byte {
	b := []byte{byte(typ) | VariantArrayValues | VariantArrayDimensions}
	b = binary.LittleEndian.AppendUint32(b, uint32(len(values)))
	b = append(b, values...)
	b = binary.LittleEndian.AppendUint32(b, uint32(count))
	for i := 0; i < count; i++ {
		b = binary.LittleEndian.AppendUint32(b, uint32(size))
	}
	return b
}
