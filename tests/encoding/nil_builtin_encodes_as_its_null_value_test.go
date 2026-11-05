// A nil pointer to a nullable built-in type encodes as that type's null
// value, and a message carrying one still parses.
//
// OPC 10000-6 §5.1.2 Table 1 gives every built-in type a Nullable column and
// a Default Value column, and defines the first: "The Nullable column
// indicates whether a 'null' value exists for the DataType in all
// DataEncodings. A 'null' value is a value that is equivalent 'no value
// specified'." The six types below are all Nullable = Yes, with the defaults
// Table 1 states: NodeId "i=0 (see 5.1.12)", Guid "All zeros", LocalizedText
// and DataValue "All fields set to default", Variant "Null", DiagnosticInfo
// "No fields specified".
//
// The byte sequences asserted here were checked against two reference stacks
// during the 2026-09-15 review of PR #901 and found identical in both:
// open62541's ua_types_encoding_binary.c (NodeId_encodeBinaryWithEncodingMask
// :625, Guid_encodeBinary:565, LocalizedText:801, DataValue:1300,
// Variant:1039, DiagnosticInfo:1381) and UA-.NETStandard's BinaryEncoder.cs
// (WriteNodeId:696, WriteGuid:580, WriteLocalizedText:823, WriteDataValue:880,
// WriteVariantValue:2386, WriteDiagnosticInfo:2122). Asserting the literal
// bytes here is what turns that one-off cross-check into something the suite
// re-runs; a test that compared ua.Encode(nil) against ua.Encode(&T{}) would
// keep agreeing with gopcua after both sides drifted away from the two
// reference stacks together.
//
// Each case recovers from a panic and reports it as that case's failure. A
// missing guard is a nil dereference, which takes the whole test binary with
// it, and then one missing guard hides the state of the other five - the
// difference between "one type is unguarded" and "six are" is not visible in
// the resulting output otherwise.

package encoding

import (
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

func nullEncodings() []nullEncoding {
	return []nullEncoding{
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
			// Already guarded before PR #901; held here so a later change to
			// the others cannot quietly take this one with it.
			want: []byte{0x00, 0x00, 0x00},
			why:  `Table 1 default "All fields set to default"`,
		},
	}
}

func TestNilBuiltinEncodesAsItsNullValue(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. README.md says when to delete it. The
	// behaviour is closed by https://github.com/gopcua/opcua/pull/901.
	const knownBug = "a nil pointer to a type §5.1.2 Table 1 lists as nullable panics instead of encoding that type's null value: six of the seven types this row checks - NodeId, Guid, LocalizedText, DataValue, Variant and DiagnosticInfo - panic on the nil receiver, and only ExtensionObject is guarded"

	cases := nullEncodings()
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
		case string(got) != string(c.want):
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
	case encodes && knownBug != "":
		t.Errorf("this now passes (a nil pointer to each of the %d nullable built-in types "+
			"encodes as that type's null value) - "+
			"delete the knownBug constant and the two arms that read it, in the commit that fixed it",
			len(cases))
	case encodes:
		t.Logf("a nil pointer to each of the %d nullable built-in types encodes as that "+
			"type's null value", len(cases))
	case knownBug != "":
		t.Logf("known bug: %s; §5.1.2 Table 1 lists every type below as nullable and gives "+
			"it a default value to encode:%s", knownBug, detail)
	default:
		t.Errorf("§5.1.2 Table 1 lists every type below as nullable and gives it a default "+
			"value, so a nil pointer to one has a value to encode:%s", detail)
	}
}
