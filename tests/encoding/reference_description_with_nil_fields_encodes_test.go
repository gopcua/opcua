// A ReferenceDescription whose pointer fields are nil encodes to the same
// bytes as one whose fields carry their null values: the 18 bytes below,
// OPC 10000-6 §5.1.2 Table 1 gives every built-in type a Nullable column and
// a Default Value column, and defines the first: "The Nullable column
// indicates whether a 'null' value exists for the DataType in all
// DataEncodings. A 'null' value is a value that is equivalent 'no value
// specified'."
//
// The two rows nearest this one are
// nil_builtin_encodes_as_its_null_value_test.go and
// nil_builtin_field_keeps_stream_framing_test.go: between them
// they cover a nullable built-in in isolation and a nullable built-in as a
// struct field. What separates this row from both is that it covers a
// concrete structure - ReferenceDescription - whose fields are nil, so the
// null values of several different built-ins have to sit in one message and
// frame each other correctly.
//
// The row runs against the shared helpers, with no scripted server in
// this package: the fully populated, all-nil and partially populated
// ReferenceDescriptions below are each encoded by encodeRecovering,
// which turns the panic an unguarded field raises into a value the case
// then reports. See harness.go.
//
// What the row asserts is that all three shapes encode to the same literal
// 18 bytes, given below field by field. What it deliberately does not
// assert: that any of the three messages decodes - no Decode call is made -
// and that a ReferenceDescription carrying non-null values (a real
// namespace index, a real browse name) encodes correctly; only the null
// forms are covered.
//
// The byte sequences asserted here were read out of two reference stacks'
// encoders and found identical in both:
// open62541's ua_types_encoding_binary.c (NodeId_encodeBinaryWithEncodingMask
// :662, TwoByte branch; ExpandedNodeId_encodeBinary:766, encoding mask 0 for
// a null ExpandedNodeID; QualifiedName_encodeBinary:832, with the null
// String's -1 length written by Array_encodeBinary:497) and UA-.NETStandard's
// BinaryEncoder.cs (WriteNodeId:722; WriteExpandedNodeId:750, whose null arm
// writes WriteUInt16(null, 0) at :755; WriteQualifiedName:822; WriteString
// :548, which writes -1 for a null string). Asserting the literal bytes here
// is what turns that one-off cross-check into something the suite re-runs; a
// row that compared gopcua's output only against gopcua's other output would
// keep agreeing with gopcua after both drifted away from the two reference
// stacks together.

package encoding

import (
	"testing"

	"github.com/gopcua/opcua/ua"
)

// TestReferenceDescriptionWithNilFieldsEncodes is the case PR #901's own
// description names - "a server-built ReferenceDescription" - and the case
// that PR does not reach. It stays measured here rather than being left to a
// review document, because the two fields it turns on are both nullable by
// §5.1.2 Table 1 and neither is covered.
//
// ReferenceDescription has five pointer fields. #901 guards ReferenceTypeID
// (*NodeID) and DisplayName (*LocalizedText). It leaves:
//
//   - NodeID and TypeDefinition, both *ExpandedNodeID. Table 1 lists
//     ExpandedNodeId as nullable with default "i=0", and gopcua returns an
//     error for a nil receiver instead.
//   - BrowseName, a *QualifiedName. Table 1 lists QualifiedName as nullable.
//     It is not a BinaryEncoder, so a nil one takes the generic pointer arm
//     in ua/encode.go - "case reflect.Ptr: if val.IsNil() { return nil, nil }"
//   - and contributes no bytes at all. The message is six bytes short and
//     nothing reports it.
//
// Both are pre-existing: the same measurement on an unpatched tree gives the
// same answer, so #901 introduces neither. What #901 does change is reach.
// Before it, a ReferenceDescription with nil fields panicked, loudly. After
// it, the panic is gone and a short, silently misframed message takes its
// place. That is still the better trade, and it is the reason this row is
// worth keeping after #901 merges rather than before.
func TestReferenceDescriptionWithNilFieldsEncodes(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. ../README.md says when to delete it.
	const knownBug = "a ReferenceDescription with nil fields does not encode: with every field nil it panics on the nil *NodeID ReferenceTypeID before any *ExpandedNodeID is reached; with that set, its *ExpandedNodeID fields return an error on a nil receiver; and a nil *QualifiedName BrowseName silently contributes no bytes, leaving the message six bytes short"

	// Every pointer field given the explicit null value its type has by
	// Table 1. This encodes to the 18 bytes in want below - the bytes the
	// two reference stacks named in the header produce for the same value -
	// and those bytes, not this call's own output, are the yardstick the
	// partially populated and all-nil ReferenceDescriptions are measured
	// against.
	full, err, panicked := encodeRecovering(&ua.ReferenceDescription{
		ReferenceTypeID: ua.NewTwoByteNodeID(0),
		NodeID:          ua.NewTwoByteExpandedNodeID(0),
		BrowseName:      &ua.QualifiedName{},
		DisplayName:     &ua.LocalizedText{},
		TypeDefinition:  ua.NewTwoByteExpandedNodeID(0),
	})
	if panicked != "" || err != nil {
		t.Fatalf("could not set the test up: encoding a fully populated ReferenceDescription "+
			"failed (err %v, panic %q)", err, panicked)
	}

	// All five left nil, as a server can build it.
	allNil, allNilErr, allNilPanic := encodeRecovering(&ua.ReferenceDescription{})

	// Only the two *ExpandedNodeID fields set, which is as far as #901's
	// guards can carry it.
	partial, partialErr, partialPanic := encodeRecovering(&ua.ReferenceDescription{
		NodeID:         ua.NewTwoByteExpandedNodeID(0),
		TypeDefinition: ua.NewTwoByteExpandedNodeID(0),
	})

	// The bytes both reference stacks produce for a ReferenceDescription
	// whose nullable fields all carry their null value. Under a correct fix
	// the fully populated, all-nil and ExpandedNodeIDs-only encodings are
	// byte-identical at these 18 bytes, so one literal covers all three.
	const want = "\x00\x00" + // ReferenceTypeID, null NodeID: encoding 0x00, identifier 0x00
		"\x00" + // IsForward, false
		"\x00\x00" + // NodeID, null ExpandedNodeID: encoding 0x00 (two-byte NodeId, no flags), identifier 0x00
		"\x00\x00\xff\xff\xff\xff" + // BrowseName, null QualifiedName: namespace index 0, name length -1
		"\x00" + // DisplayName, null LocalizedText: encoding mask 0x00, neither string present
		"\x00\x00\x00\x00" + // NodeClass 0, Int32
		"\x00\x00" // TypeDefinition, null ExpandedNodeID: as NodeID above

	// The named defect is an event - the error the two *ExpandedNodeID
	// fields return for a nil receiver, or a panic from one of the fields
	// #901 does not guard - and an event is a presence, so the arms key on
	// it rather than on the absence of a right outcome: while the all-nil
	// message still fails on the nil receiver, the behaviour is still
	// missing and the row stays green with the known bug logged; once it
	// encodes, a fix has landed, and the row then demands the framing be
	// right - a landed fix that still leaves the message short is red,
	// not green.
	event := allNilErr != nil || allNilPanic != ""

	encodes := allNilErr == nil && allNilPanic == ""
	framed := string(full) == want && string(allNil) == want && string(partial) == want

	// Arm order: the two passes arms come first, because an all-nil message
	// that fails on the nil receiver never encodes, so they cannot take a
	// state the event arm needs to see; the event arm comes before the
	// wrong-fix arm, because a fix that guards only some of the fields
	// leaves the event and must stay green, not red; and the wrong-fix arm
	// reads the constant, so with the constant empty a missing behaviour
	// lands in the default arm below it.
	switch {
	case encodes && framed && knownBug != "":
		t.Errorf("this now passes (a ReferenceDescription with all fields nil encodes to the 18 "+
			"bytes the reference stacks produce, the same as a fully populated one and one with "+
			"only its ExpandedNodeIDs set: % x) - delete the knownBug constant, every arm that "+
			"reads it, and anything only those arms used, in the commit that fixed it",
			allNil)
	case encodes && framed:
		t.Logf("a ReferenceDescription with nil fields encodes to the 18 bytes the reference "+
			"stacks produce, byte-identical to a fully populated one: % x", allNil)
	case event && knownBug != "":
		t.Logf("known bug: %s (all fields nil: err %v, panic %q; only the ExpandedNodeIDs set: "+
			"%d bytes where the reference stacks produce %d)",
			knownBug, allNilErr, allNilPanic, len(partial), len(want))
	case knownBug != "":
		t.Errorf("the nil-receiver failure is gone, but the outcome is still wrong - a fix "+
			"for this landed and is wrong: with all fields nil the message now encodes, but the "+
			"bytes are not the 18 the reference stacks produce (full % x, all-nil % x, "+
			"ExpandedNodeIDs-only err %v panic %q % x, want % x)",
			full, allNil, partialErr, partialPanic, partial, want)
	default:
		t.Errorf("§5.1.2 Table 1 lists every pointer field of ReferenceDescription as nullable, so "+
			"one with nil fields has a value to encode; all fields nil gave err %v and panic %q, "+
			"and with only the ExpandedNodeIDs set it was %d bytes where the reference stacks "+
			"produce %d",
			allNilErr, allNilPanic, len(partial), len(want))
	}
}
