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
// The byte sequences asserted here were checked against two reference stacks
// during the 2026-09-15 review of PR #901 and found identical in both:
// open62541's ua_types_encoding_binary.c (NodeId_encodeBinaryWithEncodingMask
// :625, Guid_encodeBinary:565, LocalizedText:801, DataValue:1300,
// Variant:1039, DiagnosticInfo:1381) and UA-.NETStandard's BinaryEncoder.cs
// (WriteNodeId:696, WriteGuid:580, WriteLocalizedText:823, WriteDataValue:880,
// WriteVariantValue:2386, WriteDiagnosticInfo:2122). Asserting the literal
// bytes here is what turns that one-off cross-check into something the suite
// re-runs; a test that compared ua.Encode(nil) against ua.Encode(&T{}) would
// keep agreeing with gopcua after its two sides - ua.Encode(nil) and
// ua.Encode(&T{}) - drifted away from the two reference stacks together.
//
// Each case recovers from a panic and reports it as that case's failure. A
// missing guard is a nil dereference, which takes the whole test binary with
// it, and then one missing guard hides the state of the other six - the
// difference between "one type is unguarded" and "six of the seven are" is
// not visible in the resulting output otherwise.

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
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. README.md says when to delete it. The
	// behaviour is closed by https://github.com/gopcua/opcua/pull/901.
	const knownBug = "a nil pointer to a type §5.1.2 Table 1 lists as nullable panics instead of encoding that type's null value, because its Encode method dereferences the nil receiver"

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

	// The named defect is an event - a panic on the nil receiver, or an
	// error returned for it - and an event is a presence, so the arms key
	// on it rather than on the absence of a correct encoding: while any
	// case still panics or fails, the behaviour is still missing and the
	// row stays green with the known bug logged; once no case does, a fix
	// has landed, and the row then demands the outcome be right - a
	// landed fix that still writes the wrong bytes is red, not green.
	// events is a count, not a boolean, because a fix can guard five of
	// the six unguarded types and the row must still say how many remain.
	events := 0
	var detail string
	encodes := true
	for _, c := range cases {
		got, err, panicked := encodeRecovering(c.nilp)
		ok := true
		var outcome string
		switch {
		case panicked != "":
			ok = false
			events++
			outcome = fmt.Sprintf("panicked (%s), where %s makes it % x", panicked, c.why, c.want)
		case err != nil:
			ok = false
			events++
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

	// Arm order: the two encodes arms come first, because a case that
	// panics or fails never encodes, so they cannot take a state the
	// events arm needs to see; the events arm comes before the wrong-fix
	// arm, because a fix that guards five of the six types leaves one
	// event and must stay green, not red; and the wrong-fix arm reads
	// the constant, so with the constant empty a missing behaviour lands
	// in the default arm below it.
	switch {
	case encodes && knownBug != "":
		t.Errorf("this now passes (a nil pointer to each of the %d nullable built-in types "+
			"this row checks encodes as that type's null value) - "+
			"delete the knownBug constant and every arm that reads it, in the commit that fixed it",
			len(cases))
	case encodes:
		t.Logf("a nil pointer to each of the %d nullable built-in types this row checks "+
			"encodes as that type's null value", len(cases))
	case events > 0 && knownBug != "":
		t.Logf("known bug: %s; %d of the %d cases still panic or fail on the nil receiver; "+
			"§5.1.2 Table 1 lists every type below as nullable and gives it a default value "+
			"to encode:%s", knownBug, events, len(cases), detail)
	case knownBug != "":
		t.Errorf("the nil receiver no longer panics or fails, but the row still measured "+
			"wrong bytes - a fix for this landed and is wrong:%s", detail)
	default:
		t.Errorf("§5.1.2 Table 1 lists every type below as nullable and gives it a default "+
			"value, so a nil pointer to one has a value to encode:%s", detail)
	}
}
