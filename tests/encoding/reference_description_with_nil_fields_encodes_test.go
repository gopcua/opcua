// A nil pointer to a nullable built-in type encodes as that type's null
// value, and a message carrying one still parses.
//
// OPC 10000-6 §5.1.2 Table 1 gives every built-in type a Nullable column and
// a Default Value column, and defines the first: "The Nullable column
// indicates whether a 'null' value exists for the DataType in all
// DataEncodings. A 'null' value is a value that is equivalent 'no value
// specified'."
//
// The row runs against the shared helpers, with no scripted server in
// this package: the fully populated, all-nil and partially populated
// ReferenceDescriptions below are each encoded by encodeRecovering,
// which turns the panic an unguarded field raises into a value the case
// then reports. See harness.go.

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
	// in place of a failure. README.md says when to delete it.
	const knownBug = "a ReferenceDescription with nil fields does not encode: its two *ExpandedNodeID fields return an error on a nil receiver, and with those set its nil *QualifiedName BrowseName silently contributes no bytes, so the message is six bytes short"

	// Every pointer field given the explicit null value its type has by
	// Table 1. This is the length a conformant encoder produces, and the
	// yardstick the partially populated ReferenceDescription below is
	// measured against.
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
	_, allNilErr, allNilPanic := encodeRecovering(&ua.ReferenceDescription{})

	// Only the two *ExpandedNodeID fields set, which is as far as #901's
	// guards can carry it.
	partial, partialErr, partialPanic := encodeRecovering(&ua.ReferenceDescription{
		NodeID:         ua.NewTwoByteExpandedNodeID(0),
		TypeDefinition: ua.NewTwoByteExpandedNodeID(0),
	})

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
	framed := partialErr == nil && partialPanic == "" && len(partial) == len(full)

	// Arm order: the two passes arms come first, because an all-nil message
	// that fails on the nil receiver never encodes, so they cannot take a
	// state the event arm needs to see; the event arm comes before the
	// wrong-fix arm, because a fix that guards only some of the fields
	// leaves the event and must stay green, not red; and the wrong-fix arm
	// reads the constant, so with the constant empty a missing behaviour
	// lands in the default arm below it.
	switch {
	case encodes && framed && knownBug != "":
		t.Errorf("this now passes (a ReferenceDescription with all fields nil encodes, and one "+
			"with only its ExpandedNodeIDs set is %d bytes, the same as a fully populated one) - "+
			"delete the knownBug constant and every arm that reads it, in the commit that fixed it",
			len(partial))
	case encodes && framed:
		t.Logf("a ReferenceDescription with nil fields encodes to the same %d bytes as a fully "+
			"populated one", len(full))
	case event && knownBug != "":
		t.Logf("known bug: %s (all fields nil: err %v, panic %q; only the ExpandedNodeIDs set: "+
			"%d bytes where a fully populated one is %d)",
			knownBug, allNilErr, allNilPanic, len(partial), len(full))
	case knownBug != "":
		t.Errorf("the nil-receiver failure is gone, but the outcome is still wrong - a fix "+
			"for this landed and is wrong: with all fields nil the message now encodes, and "+
			"with only the ExpandedNodeIDs set it gave err %v and panic %q and %d bytes, "+
			"where a fully populated one is %d",
			partialErr, partialPanic, len(partial), len(full))
	default:
		t.Errorf("§5.1.2 Table 1 lists every pointer field of ReferenceDescription as nullable, so "+
			"one with nil fields has a value to encode; all fields nil gave err %v and panic %q, "+
			"and with only the ExpandedNodeIDs set it was %d bytes where a fully populated one is %d",
			allNilErr, allNilPanic, len(partial), len(full))
	}
}
