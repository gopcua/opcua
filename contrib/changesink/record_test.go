package changesink

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

func f64(f float64) *float64 { return &f }

func TestNewRecord(t *testing.T) {
	ts := time.Date(2026, 10, 5, 10, 11, 12, 345_000_000, time.FixedZone("TR", 3*3600))
	src := time.Date(2026, 10, 5, 7, 0, 0, 0, time.UTC)
	nid := ua.NewStringNodeID(2, "line1.temp")

	tests := []struct {
		name     string
		value    any
		wantStr  string
		wantNum  *float64
		wantType string
	}{
		{"bool true", true, "true", f64(1), "Boolean"},
		{"bool false", false, "false", f64(0), "Boolean"},
		{"int8", int8(-3), "-3", f64(-3), "SByte"},
		{"uint8", uint8(3), "3", f64(3), "Byte"},
		{"int16", int16(-300), "-300", f64(-300), "Int16"},
		{"uint16", uint16(300), "300", f64(300), "Uint16"},
		{"int32", int32(-70000), "-70000", f64(-70000), "Int32"},
		{"uint32", uint32(70000), "70000", f64(70000), "Uint32"},
		{"int64", int64(-1 << 40), "-1099511627776", f64(-1 << 40), "Int64"},
		{"uint64", uint64(1 << 40), "1099511627776", f64(1 << 40), "Uint64"},
		{"float", float32(1.5), "1.5", f64(1.5), "Float"},
		{"double", 21.25, "21.25", f64(21.25), "Double"},
		{"nan", math.NaN(), "NaN", nil, "Double"},
		{"string", "hello", "hello", nil, "String"},
		{"datetime", src, "2026-10-05T07:00:00Z", nil, "DateTime"},
		{"bytestring", []byte{1, 2, 3}, "AQID", nil, "ByteString"},
		{"int array", []int32{1, 2, 3}, "[1,2,3]", nil, "Int32[]"},
		{"node id", ua.NewNumericNodeID(0, 85), "i=85", nil, "NodeID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := &ua.DataValue{
				EncodingMask:    ua.DataValueValue | ua.DataValueSourceTimestamp,
				Value:           ua.MustVariant(tt.value),
				SourceTimestamp: src,
			}
			r := NewRecord("plant-a", server.ChangeEvent{NodeID: nid, Value: dv, Time: ts})
			require.Equal(t, "plant-a", r.Server)
			require.Equal(t, "ns=2;s=line1.temp", r.NodeID)
			require.Equal(t, uint16(2), r.Namespace)
			require.Equal(t, tt.wantStr, r.ValueStr)
			require.Equal(t, tt.wantNum, r.ValueNum)
			require.Equal(t, tt.wantType, r.DataType)
			require.Equal(t, uint32(ua.StatusOK), r.Status)
			require.Equal(t, ts.UTC(), r.Time)
			require.Equal(t, src, r.SourceTime)
		})
	}
}

func TestNewRecordNilAndBad(t *testing.T) {
	nid := ua.NewNumericNodeID(1, 7)

	r := NewRecord("s", server.ChangeEvent{NodeID: nid})
	require.Equal(t, uint32(ua.StatusBad), r.Status)
	require.Equal(t, "Null", r.DataType)
	require.False(t, r.Time.IsZero(), "zero event time defaults to now")

	r = NewRecord("s", server.ChangeEvent{NodeID: nid, Value: &ua.DataValue{
		EncodingMask: ua.DataValueStatusCode, Status: ua.StatusBadNodeIDUnknown,
	}, Time: time.Now()})
	require.Equal(t, uint32(ua.StatusBadNodeIDUnknown), r.Status)
	require.Equal(t, "Null", r.DataType)
	require.Nil(t, r.ValueNum)
	require.True(t, r.SourceTime.IsZero())
}

func TestRecordEncodings(t *testing.T) {
	ts := time.Date(2026, 10, 5, 7, 8, 9, 123_000_000, time.UTC)
	r := Record{Time: ts, Server: "s", NodeID: "ns=1;s=a", Namespace: 1, ValueStr: "1.5", ValueNum: f64(1.5), DataType: "Double"}

	row := r.clickhouseRow()
	require.Equal(t, "2026-10-05 07:08:09.123", row.TS)
	require.Equal(t, "2026-10-05 07:08:09.123", row.SourceTS, "source_ts falls back to ts")

	b, err := json.Marshal(r)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	require.Equal(t, "2026-10-05T07:08:09.123Z", m["ts"])
	require.Equal(t, "ns=1;s=a", m["node_id"])
	require.Equal(t, 1.5, m["value_num"])
	require.Equal(t, "Double", m["data_type"])
	require.NotContains(t, m, "source_ts")

	r.ValueNum = nil
	r.SourceTime = ts.Add(-time.Second)
	b, err = json.Marshal(r)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &m))
	require.Nil(t, m["value_num"])
	require.Equal(t, "2026-10-05T07:08:08.123Z", m["source_ts"])
}

func TestSanitizeTopicLevel(t *testing.T) {
	require.Equal(t, "ns=2;s=a_b_c_d", sanitizeTopicLevel("ns=2;s=a/b+c#d"))
	require.Equal(t, "_", sanitizeTopicLevel(""))
}
