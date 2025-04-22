package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrors(t *testing.T) {
	t.Run("expand", func(t *testing.T) {
		err := Errorf("hello %s", "world")
		require.Error(t, err, errors.New("opcua: hello world"))
	})
	t.Run("simple", func(t *testing.T) {
		err := New("hello")
		require.Error(t, err, errors.New("opcua: hello"))
	})
	t.Run("parameter", func(t *testing.T) {
		err := New("hello %s")
		require.Error(t, err, errors.New("opcua: hello %s"))
	})
}
