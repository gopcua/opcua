// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package changesink

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gopcua/opcua/server"
	"github.com/gopcua/opcua/ua"
)

// chTimeLayout is the DateTime64(3) layout accepted by ClickHouse JSONEachRow.
const chTimeLayout = "2006-01-02 15:04:05.000"

// Record is a flattened node change, written as one ClickHouse row and
// published as one MQTT JSON payload.
type Record struct {
	// Time is the server time of the change.
	Time time.Time `json:"-"`
	// Server is Config.ServerName.
	Server string `json:"server"`
	// NodeID is the string form of the node id, e.g. "ns=2;s=temp".
	NodeID string `json:"node_id"`
	// Namespace is the namespace index of NodeID.
	Namespace uint16 `json:"namespace"`
	// ValueStr is the value as a string (JSON for arrays and complex types).
	ValueStr string `json:"value_str"`
	// ValueNum is the numeric value for numeric and boolean types, else nil.
	ValueNum *float64 `json:"value_num"`
	// DataType is the OPC UA built-in type name, e.g. "Double".
	DataType string `json:"data_type"`
	// Status is the OPC UA status code of the value.
	Status uint32 `json:"status"`
	// SourceTime is the source timestamp of the value (zero if not set).
	SourceTime time.Time `json:"-"`
}

// NewRecord converts a server change event into a Record.
func NewRecord(serverName string, ev server.ChangeEvent) Record {
	r := Record{
		Time:   ev.Time.UTC(),
		Server: serverName,
	}
	if r.Time.IsZero() {
		r.Time = time.Now().UTC()
	}
	if ev.NodeID != nil {
		r.NodeID = ev.NodeID.String()
		r.Namespace = ev.NodeID.Namespace()
	}

	dv := ev.Value
	if dv == nil {
		r.Status = uint32(ua.StatusBad)
		r.DataType = "Null"
		return r
	}
	r.Status = uint32(dv.Status)
	if dv.EncodingMask&ua.DataValueSourceTimestamp != 0 {
		r.SourceTime = dv.SourceTimestamp.UTC()
	}
	if dv.Value == nil {
		r.DataType = "Null"
		return r
	}
	r.DataType = strings.TrimPrefix(dv.Value.Type().String(), "TypeID")
	if dv.Value.ArrayLength() > 0 || len(dv.Value.ArrayDimensions()) > 0 {
		r.DataType += "[]"
	}
	r.ValueStr, r.ValueNum = formatValue(dv.Value.Value())
	return r
}

func num(f float64) *float64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return nil
	}
	return &f
}

func formatValue(v any) (string, *float64) {
	switch x := v.(type) {
	case nil:
		return "", nil
	case bool:
		if x {
			return "true", num(1)
		}
		return "false", num(0)
	case int8:
		return fmt.Sprint(x), num(float64(x))
	case uint8:
		return fmt.Sprint(x), num(float64(x))
	case int16:
		return fmt.Sprint(x), num(float64(x))
	case uint16:
		return fmt.Sprint(x), num(float64(x))
	case int32:
		return fmt.Sprint(x), num(float64(x))
	case uint32:
		return fmt.Sprint(x), num(float64(x))
	case int64:
		return fmt.Sprint(x), num(float64(x))
	case uint64:
		return fmt.Sprint(x), num(float64(x))
	case float32:
		return fmt.Sprint(x), num(float64(x))
	case float64:
		return fmt.Sprint(x), num(x)
	case string:
		return x, nil
	case time.Time:
		return x.UTC().Format(time.RFC3339Nano), nil
	case []byte:
		b, _ := json.Marshal(x) // base64
		return strings.Trim(string(b), `"`), nil
	case fmt.Stringer:
		return x.String(), nil
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return fmt.Sprint(x), nil
		}
		return string(b), nil
	}
}

// clickhouseRow is the JSONEachRow representation of a Record.
type clickhouseRow struct {
	TS       string   `json:"ts"`
	Server   string   `json:"server"`
	NodeID   string   `json:"node_id"`
	NS       uint16   `json:"namespace"`
	ValueStr string   `json:"value_str"`
	ValueNum *float64 `json:"value_num"`
	DataType string   `json:"data_type"`
	Status   uint32   `json:"status"`
	SourceTS string   `json:"source_ts"`
}

func (r Record) clickhouseRow() clickhouseRow {
	src := r.SourceTime
	if src.IsZero() {
		src = r.Time
	}
	return clickhouseRow{
		TS:       r.Time.UTC().Format(chTimeLayout),
		Server:   r.Server,
		NodeID:   r.NodeID,
		NS:       r.Namespace,
		ValueStr: r.ValueStr,
		ValueNum: r.ValueNum,
		DataType: r.DataType,
		Status:   r.Status,
		SourceTS: src.UTC().Format(chTimeLayout),
	}
}

// MarshalJSON encodes the record as the MQTT payload.
func (r Record) MarshalJSON() ([]byte, error) {
	type plain Record // drop methods to avoid recursion
	p := struct {
		plain
		TS       string `json:"ts"`
		SourceTS string `json:"source_ts,omitempty"`
	}{plain: plain(r), TS: r.Time.UTC().Format(time.RFC3339Nano)}
	if !r.SourceTime.IsZero() {
		p.SourceTS = r.SourceTime.UTC().Format(time.RFC3339Nano)
	}
	return json.Marshal(p)
}
