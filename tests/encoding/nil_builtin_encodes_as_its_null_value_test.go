// A nil pointer to a nullable built-in type encodes as that type's null
// value. That a message carrying one still parses is not asserted here -
// this row encodes each value in isolation and decodes nothing; it is what
// nil_builtin_field_keeps_stream_framing_test.go checks, by decoding the
// message the null value sits in.
//
// OPC 10000-6 §5.1.2 Table 1 gives every built-in type a Nullable column and
// a Default Value column, and defines the first: "The Nullable column
// indicates whether a 'null' value exists for the DataType in all
// DataEncodings. A 'null' value is a value that is equivalent 'no value
// specified'." The seven types below are all Nullable = Yes, with the
// defaults Table 1 states: NodeId "i=0 (see 5.1.12)", Guid "All zeros",
// LocalizedText and DataValue "All fields set to default", Variant "Null",
// DiagnosticInfo "No fields specified", ExtensionObject "All fields set to
// default". ExtensionObject is the one case already guarded before PR #901;
// it is held here so a later change to the others cannot quietly take this
// one with it.
//
// The row runs against the shared helpers, with no scripted server in
// this package: each nil pointer is encoded by encodeRecovering, which
// turns the panic a missing guard raises into a value the case then
// reports. See harness.go.
//
// The byte sequences asserted here were read out of two reference stacks'
// encoders: open62541's ua_types_encoding_binary.c
// (NodeId_encodeBinaryWithEncodingMask:638, Guid:602, LocalizedText:858,
// Variant:1119, DataValue:1424, DiagnosticInfo:1509) and UA-.NETStandard's
// BinaryEncoder.cs, at the line each writes the null byte rather than at
// its public entry point (WriteNodeId:727, WriteLocalizedText:841,
// WriteDataValue:916, the Variant path reached through WriteVariant at
// :1710, DiagnosticInfo's inner overload at :2394). WriteGuid:615 is the
// exception: it takes a value type, so it has no null arm at all and
// writes the sixteen zero bytes unconditionally.
//
// Asserting the literal bytes here is what turns that cross-check into
// something the suite re-runs; a test that compared ua.Encode(nil)
// against ua.Encode(&T{}) would keep agreeing with gopcua after its two
// sides drifted away from the two reference stacks together.
//
// Each case recovers from a panic and records it as that case's outcome,
// which fails the row. A missing guard is a nil dereference, which takes
// the whole test binary with it, and then one missing guard hides the
// state of the other six - the difference between "one type is
// unguarded" and "six of the seven are" is not visible in the resulting
// output otherwise.

package encoding

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/gopcua/opcua/ua"
)

// nullEncoding pairs a typed nil pointer with the bytes §5.1.2 Table 1 gives
// as that type's default value.
type nullEncoding struct {
	name string
	nilp interface{}
	want []byte
	// why names the Default Value column entry the bytes come from.
	why string
}

func TestNilBuiltinEncodesAsItsNullValue(t *testing.T) {
	// gopcua did not do this until https://github.com/gopcua/opcua/pull/901
	// merged; it passes since.

	cases := []nullEncoding{
		{
			name: "NodeId",
			nilp: (*ua.NodeID)(nil),
			// Two-byte encoding: one encoding byte, one identifier byte.
			want: []byte{0x00, 0x00},
			why:  `Table 1 default "i=0 (see 5.1.12)"`,
		},
		{
			name: "Guid",
			nilp: (*ua.GUID)(nil),
			want: make([]byte, 16),
			why:  `Table 1 default "All zeros"`,
		},
		{
			name: "LocalizedText",
			nilp: (*ua.LocalizedText)(nil),
			want: []byte{0x00},
			why:  `Table 1 default "All fields set to default", i.e. an empty encoding mask`,
		},
		{
			name: "DataValue",
			nilp: (*ua.DataValue)(nil),
			want: []byte{0x00},
			why:  `Table 1 default "All fields set to default", i.e. an empty encoding mask`,
		},
		{
			name: "Variant",
			nilp: (*ua.Variant)(nil),
			want: []byte{0x00},
			why:  `Table 1 default "Null", i.e. an empty encoding mask`,
		},
		{
			name: "DiagnosticInfo",
			nilp: (*ua.DiagnosticInfo)(nil),
			want: []byte{0x00},
			why:  `Table 1 default "No fields specified", i.e. an empty encoding mask`,
		},
		{
			name: "ExtensionObject",
			nilp: (*ua.ExtensionObject)(nil),
			// TypeId i=0 as a two-byte NodeId, then an empty encoding mask.
			want: []byte{0x00, 0x00, 0x00},
			why:  `Table 1 default "All fields set to default"`,
		},
	}
	if len(cases) == 0 {
		t.Fatal("the case table is empty - this row would report a pass " +
			"having checked nothing")
	}

	var detail string
	encodes := true
	for _, c := range cases {
		got, err, panicked := encodeRecovering(c.nilp)
		ok := true
		var outcome string
		switch {
		case panicked != "":
			ok = false
			outcome = fmt.Sprintf("panicked (%s), where %s makes it % x", panicked, c.why, c.want)
		case err != nil:
			ok = false
			outcome = fmt.Sprintf("failed with %v, where %s makes it % x", err, c.why, c.want)
		case !bytes.Equal(got, c.want):
			ok = false
			outcome = fmt.Sprintf("encoded as % x, where %s makes it % x", got, c.why, c.want)
		default:
			outcome = fmt.Sprintf("encoded as % x, as %s makes it", got, c.why)
		}
		if !ok {
			encodes = false
		}
		detail += fmt.Sprintf("\n    %-16s %s", c.name, outcome)
	}

	switch {
	case encodes:
		t.Logf("a nil pointer to each of the %d nullable built-in types this row checks "+
			"encodes as that type's null value", len(cases))
	default:
		t.Errorf("§5.1.2 Table 1 lists every type below as nullable and gives it a default "+
			"value, so a nil pointer to one has a value to encode:%s", detail)
	}
}
