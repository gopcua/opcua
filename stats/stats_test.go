package stats

import (
	"errors"
	"expvar"
	"io"
	"testing"

	"github.com/gopcua/opcua/ua"
	"github.com/pascaldekloe/goe/verify"
)

func newExpVarInt(i int64) *expvar.Int {
	v := &expvar.Int{}
	v.Set(i)
	return v
}

func TestConvienienceFuncs(t *testing.T) {
	Reset()

	Client().Add("a", 1)
	verify.Values(t, "", Client().Get("a"), newExpVarInt(1))

	Error().Add("b", 2)
	verify.Values(t, "", Error().Get("b"), newExpVarInt(2))

	Subscription().Add("c", 3)
	verify.Values(t, "", Subscription().Get("c"), newExpVarInt(3))
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
			got, want := s.Error.Get(tt.key), newExpVarInt(1)
			verify.Values(t, "", got, want)
		})
	}
}
