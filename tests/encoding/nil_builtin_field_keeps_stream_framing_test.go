// A nil pointer to a nullable built-in type encodes as that type's null
// value, and a message carrying one still parses.
//
// OPC 10000-6 §5.1.2 Table 1 gives every built-in type a Nullable column,
// and defines it: "The Nullable column indicates whether a 'null' value
// exists for the DataType in all DataEncodings. A 'null' value is a value
// that is equivalent 'no value specified'."
//
// This is the failure the byte assertions in
// nil_builtin_encodes_as_its_null_value_test.go cannot see. They check one
// value in isolation; a null encoding of the wrong length is only harmful
// because everything after it in the same message shifts. A guard that wrote
// one byte too few for NodeId would still look like a plausible null value on
// its own, and would desynchronise every field behind it.
//
// The row runs against a struct whose first field is a nil pointer to a
// nullable built-in and whose second is a known marker: it encodes the
// struct, decodes the bytes back, and reads the marker out of the result.
// There is no scripted server in this package; the encoding goes through
// the shared encodeRecovering, which turns the panic a missing guard
// raises into a value the case then reports. See harness.go.
//
// What it asserts about the null bytes is that the marker survives behind
// them, and that gopcua's own decoder accepts them - of the two rows, only
// this one checks that, because encoding a null value no decoder reads is
// not an improvement on panicking. What it deliberately does not assert is
// the content of the null bytes: a *ua.NodeID guard writing {0x00, 0x01} -
// the right length, a valid but wrong TwoByte NodeId - would leave this
// row's check green, and the byte assertions in
// nil_builtin_encodes_as_its_null_value_test.go are the backstop for it.
//
// No reference stack was read for this row: it measures gopcua's encoder
// against gopcua's own decoder, so no other stack is exercised. The null
// byte lengths it rides on were checked against open62541 and
// UA-.NETStandard in nil_builtin_encodes_as_its_null_value_test.go.
//
// Each case recovers from a panic and reports it; while the knownBug
// constant stands, that report is logged as a green "known bug" line that
// still names the case, and it becomes that case's failure once the
// constant is deleted. A missing guard is a nil dereference, which takes
// the whole test binary with it, and then one missing guard hides the state
// of the other six - NodeId, Guid, LocalizedText, DataValue, Variant and
// DiagnosticInfo, with ExtensionObject, the seventh, the one case already
// guarded before PR #901 - so the difference between "one type is unguarded"
// and "six of them are" is not visible in the resulting output otherwise.

package encoding

import (
	"fmt"
	"testing"

	"github.com/gopcua/opcua/ua"
)

// marker is written after the nullable field in the framing test. The value
// is arbitrary; it has to be one that a short or long null encoding will not
// produce by accident, so it is not zero and not a small integer.
const marker = uint32(0xDEADBEEF)

// The framing structs. Each has a nullable built-in first and the same uint32
// marker behind it. The field's static type has to be the concrete pointer
// type because the decode half cannot work through an interface: ua.Decode
// into a struct with a ua.BinaryDecoder-typed field panics ("interface
// conversion: interface is nil, not ua.BinaryDecoder"), and a field whose
// interface type has no Decode method is rejected as an unsupported type.
// The encode side is not the constraint - a field of interface type holding
// a typed nil encodes just as the concrete value does - so the shape
// repeating seven times is a choice, not a necessity.
type (
	nodeIDFramed struct {
		Field  *ua.NodeID
		Marker uint32
	}
	guidFramed struct {
		Field  *ua.GUID
		Marker uint32
	}
	localizedTextFramed struct {
		Field  *ua.LocalizedText
		Marker uint32
	}
	dataValueFramed struct {
		Field  *ua.DataValue
		Marker uint32
	}
	variantFramed struct {
		Field  *ua.Variant
		Marker uint32
	}
	diagnosticInfoFramed struct {
		Field  *ua.DiagnosticInfo
		Marker uint32
	}
	extensionObjectFramed struct {
		Field  *ua.ExtensionObject
		Marker uint32
	}
)

func TestNilBuiltinFieldKeepsStreamFraming(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. README.md says when to delete it. The
	// behaviour is closed by https://github.com/gopcua/opcua/pull/901.
	const knownBug = "a message whose field is a nil pointer to a nullable built-in does not survive encode and decode, because encoding it panics"

	cases := []struct {
		name string
		run  func() (uint32, string, string)
	}{
		{"NodeId", func() (uint32, string, string) {
			var in nodeIDFramed
			event, why := roundTrip(&nodeIDFramed{Marker: marker}, &in)
			return in.Marker, event, why
		}},
		{"Guid", func() (uint32, string, string) {
			var in guidFramed
			event, why := roundTrip(&guidFramed{Marker: marker}, &in)
			return in.Marker, event, why
		}},
		{"LocalizedText", func() (uint32, string, string) {
			var in localizedTextFramed
			event, why := roundTrip(&localizedTextFramed{Marker: marker}, &in)
			return in.Marker, event, why
		}},
		{"DataValue", func() (uint32, string, string) {
			var in dataValueFramed
			event, why := roundTrip(&dataValueFramed{Marker: marker}, &in)
			return in.Marker, event, why
		}},
		{"Variant", func() (uint32, string, string) {
			var in variantFramed
			event, why := roundTrip(&variantFramed{Marker: marker}, &in)
			return in.Marker, event, why
		}},
		{"DiagnosticInfo", func() (uint32, string, string) {
			var in diagnosticInfoFramed
			event, why := roundTrip(&diagnosticInfoFramed{Marker: marker}, &in)
			return in.Marker, event, why
		}},
		{"ExtensionObject", func() (uint32, string, string) {
			var in extensionObjectFramed
			event, why := roundTrip(&extensionObjectFramed{Marker: marker}, &in)
			return in.Marker, event, why
		}},
	}
	if len(cases) == 0 {
		t.Fatal("the case table is empty - this row would report a pass " +
			"having checked nothing")
	}

	// The named defect is an event - encoding panics, or encoding fails on
	// the nil receiver - and an event is a presence, so the arms key on it
	// rather than on the absence of a surviving round trip: while any case
	// still panics or fails on encode, the behaviour is still missing and
	// the row stays green with the known bug logged; once no case does, a
	// fix has landed, and the row then demands the message survive - a
	// landed fix whose bytes still shift the marker, or whose bytes its
	// own decoder rejects, is red, not green. events is a count, not a
	// boolean, because a fix can guard five of the six unguarded types
	// and the row must still say how many remain.
	events := 0
	var detail string
	survives := true
	for _, c := range cases {
		got, event, why := c.run()
		ok := true
		var outcome string
		switch {
		case event != "":
			ok = false
			events++
			outcome = fmt.Sprintf("did not survive encode and decode: %s", event)
		case why != "":
			ok = false
			outcome = fmt.Sprintf("did not survive encode and decode: %s", why)
		case got != marker:
			ok = false
			outcome = fmt.Sprintf("decoded the uint32 behind it as %#x where %#x went in, "+
				"so the null encoding is the wrong length and everything behind it has shifted",
				got, marker)
		default:
			outcome = "survived encode and decode with the uint32 behind it intact"
		}
		if !ok {
			survives = false
		}
		detail += fmt.Sprintf("\n    %-16s %s", c.name, outcome)
	}

	// Arm order: the two survives arms come first, because a case that
	// panics or fails on encode never survives, so they cannot take a
	// state the events arm needs to see; the events arm comes before the
	// wrong-fix arm, because a fix that guards five of the six types
	// leaves one event and must stay green, not red; and the wrong-fix
	// arm reads the constant, so with the constant empty a missing
	// behaviour lands in the default arm below it.
	switch {
	case survives && knownBug != "":
		t.Errorf("this now passes (a struct with a nil field of each of the %d nullable "+
			"built-in types this row checks, followed by a uint32, survives encode and "+
			"decode with the uint32 intact) - "+
			"delete the knownBug constant and every arm that reads it, in the commit that fixed it",
			len(cases))
	case survives:
		t.Logf("a struct with a nil field of each of the %d nullable built-in types this "+
			"row checks, followed by a uint32, survives encode and decode with the uint32 intact",
			len(cases))
	case events > 0 && knownBug != "":
		t.Logf("known bug: %s; %d of the %d cases still panic or fail on encode; "+
			"§5.1.2 Table 1 lists every type below as nullable, so a nil field has a value "+
			"to encode and the uint32 behind it must come back as it went:%s",
			knownBug, events, len(cases), detail)
	case knownBug != "":
		t.Errorf("encoding no longer panics or fails, but the message still does not "+
			"survive encode and decode - a fix for this landed and is wrong:%s", detail)
	default:
		t.Errorf("§5.1.2 Table 1 lists every type below as nullable, so a nil field has a "+
			"value to encode and the uint32 behind it must come back as it went:%s", detail)
	}
}

// roundTrip encodes out and decodes the bytes into into. event carries the
// named defect's event - encoding panicked, or encoding failed - and is empty
// when neither happened. why carries the reason the trip did not survive
// once encoding succeeded, and is empty when nothing went wrong; the caller
// then reads its own marker field, which roundTrip cannot read for it,
// because it holds the decoded value as an interface{}. The count ua.Decode
// returns is checked against the length of the bytes the encode produced,
// because ua.Decode does not report trailing bytes itself, and a null
// encoding of the wrong length would otherwise leave them unseen. Only the
// test function above turns any of the three into a failure.
func roundTrip(out, into interface{}) (event, why string) {
	b, err, panicked := encodeRecovering(out)
	if panicked != "" {
		return fmt.Sprintf("encoding panicked (%s)", panicked), ""
	}
	if err != nil {
		return fmt.Sprintf("encoding failed: %v", err), ""
	}
	n, err := ua.Decode(b, into)
	if err != nil {
		return "", fmt.Sprintf("decoding the % x it produced failed: %v", b, err)
	}
	if n != len(b) {
		return "", fmt.Sprintf("decode consumed %d of the %d bytes it produced, "+
			"so the null encoding is the wrong length", n, len(b))
	}
	return "", ""
}
