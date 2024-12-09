package stats

import (
	"errors"
	"expvar"
	"io"
	"testing"

	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

func newExpVarInt(i int64) *expvar.Int {
	v := &expvar.Int{}
	v.Set(i)
	return v
}

func TestConvienienceFuncs(t *testing.T) {
	Reset()

	Client().Add("a", 1)
	require.Equal(t, newExpVarInt(1), Client().Get("a"))

	Error().Add("b", 2)
	require.Equal(t, newExpVarInt(2), Error().Get("b"))

	Subscription().Add("c", 3)
	require.Equal(t, newExpVarInt(3), Subscription().Get("c"))
}

func TestRecordError(t *testing.T) {
	tests := []struct {
		err error
		key string
	}{
		{io.EOF, "io.EOF"},
		{ua.StatusOK, "ua.StatusOK"},
		{ua.StatusBad, "ua.StatusBad"},
		{ua.StatusUncertain, "ua.StatusUncertain"},
		{ua.StatusBadAlreadyExists, "ua.StatusBadAlreadyExists"},
		{errors.New("hello"), "*errors.errorString"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			s := NewStats()
			s.RecordError(tt.err)
			require.Equal(t, newExpVarInt(1), s.Error.Get(tt.key))
		})
	}
}
