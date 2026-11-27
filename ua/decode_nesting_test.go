// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

import (
	"bytes"
	"runtime/debug"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// specNestingLevels is the depth a decoder has to accept, OPC 10000-6 §5.1.8.
// It is a constant of its own so that lowering maxDecodeNesting below it
// fails the test below.
const specNestingLevels = 100

// Each nesting level costs a byte or two on the wire, so a short message
// nests deeper than any stack can hold. Decoding must report an error past
// maxDecodeNesting, see its comment.
func TestDecodeRejectsNestingDeeperThanTheLimit(t *testing.T) {
	smallStack(t, 64<<20)

	t.Run("variant", func(t *testing.T) {
		// a Variant holding a Variant holding a Variant, and so on
		b := append(bytes.Repeat([]byte{byte(TypeIDVariant)}, 1<<20), 0)

		_, err := new(Variant).Decode(b)
		require.ErrorIs(t, err, StatusBadEncodingLimitsExceeded)
	})

	t.Run("diagnostic info", func(t *testing.T) {
		// an inner DiagnosticInfo per level, OPC 10000-6 §5.2.2.12
		b := append(bytes.Repeat([]byte{DiagnosticInfoInnerDiagnosticInfo}, 1<<20), 0)

		_, err := new(DiagnosticInfo).Decode(b)
		require.ErrorIs(t, err, StatusBadEncodingLimitsExceeded)
	})

	t.Run("extension object body", func(t *testing.T) {
		// a Variant holding an ExtensionObject whose registered value
		// holds a Variant, nested past the limit: the value inside the
		// body is decoded by the generic decoder, without the ReadStruct
		// the built-in types read their parts with
		v := new(Variant)
		for i := 0; i < 200; i++ {
			v = MustVariant(NewExtensionObject(&KeyValuePair{Value: v}))
		}
		b, err := v.Encode()
		require.NoError(t, err)

		_, err = new(Variant).Decode(b)
		require.ErrorIs(t, err, StatusBadEncodingLimitsExceeded)
	})
}

// The limit has to keep the nesting levels the spec requires of a decoder.
func TestDecodeSupportsTheNestingLevelsTheSpecRequires(t *testing.T) {
	v := new(Variant)
	_, err := v.Decode(nestedVariantBytes(specNestingLevels))
	require.NoError(t, err)

	// count the Variants the outer one holds, one per nesting level
	depth := 0
	for val := v.Value(); ; {
		nested, ok := val.(*Variant)
		if !ok {
			break
		}
		depth++
		val = nested.Value()
	}
	require.Equal(t, specNestingLevels, depth)
}

// The nesting level belongs to the value being decoded, not to the process:
// decodes running at the same time must not count each other's nesting.
func TestDecodeNestingLimitIsNotSharedBetweenDecodes(t *testing.T) {
	shallow := nestedVariantBytes(3)
	deep := nestedVariantBytes(specNestingLevels + 1)

	var wg sync.WaitGroup
	var shallowErr, deepErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, shallowErr = new(Variant).Decode(shallow)
	}()
	go func() {
		defer wg.Done()
		_, deepErr = new(Variant).Decode(deep)
	}()
	wg.Wait()

	require.NoError(t, shallowErr)
	require.ErrorIs(t, deepErr, StatusBadEncodingLimitsExceeded)
}

// smallStack reaches the stack limit with a small message instead of the
// 1 GiB the runtime allows by default, and restores it when the test ends.
func smallStack(t *testing.T, limit int) {
	old := debug.SetMaxStack(limit)
	t.Cleanup(func() { debug.SetMaxStack(old) })
}

// nestedVariantBytes returns the encoding of a Variant that holds a Variant
// that holds a Variant, n times, with a null Variant at the bottom.
func nestedVariantBytes(n int) []byte {
	return append(bytes.Repeat([]byte{byte(TypeIDVariant)}, n), 0)
}
