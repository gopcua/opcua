//go:build conformance

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
// the shared encodeRecovering and the decode through the shared
// decodeRecovering, either of which turns the panic a missing guard
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
// The seven cases split six to one: NodeId, Guid, LocalizedText,
// DataValue, Variant and DiagnosticInfo had no nil guard, and
// ExtensionObject, the seventh, was the one case already guarded before
// PR #901, which closed the six.

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
// a typed nil goes through the same Encode call as a concrete nil pointer
// does - so the shape repeating seven times is a choice, not a necessity.
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
	// gopcua did not do this until https://github.com/gopcua/opcua/pull/901
	// merged; it passes since.

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

	var detail string
	survives := true
	for _, c := range cases {
		got, event, why := c.run()
		ok := true
		var outcome string
		switch {
		case event != "":
			ok = false
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

	switch {
	case survives:
		t.Logf("a struct with a nil field of each of the %d nullable built-in types this "+
			"row checks, followed by a uint32, survives encode and decode with the uint32 intact",
			len(cases))
	default:
		t.Errorf("§5.1.2 Table 1 lists every type below as nullable, so a nil field has a "+
			"value to encode and the uint32 behind it must come back as it went:%s", detail)
	}
}

// roundTrip encodes out and decodes the bytes into into. event carries a
// failure on the encode side - encoding panicked, or encoding failed - and is empty
// when neither happened. why carries the reason the trip did not survive
// once encoding succeeded - decoding panicked, decoding failed, or the
// count ua.Decode returned not matching the length of the bytes the encode
// produced, because ua.Decode does not report trailing bytes itself and a
// null encoding of the wrong length would otherwise leave them unseen - and
// is empty when nothing went wrong; the caller then reads its own marker
// field, which roundTrip has no static type for. Only the test function
// above turns any of the three into a failure.
func roundTrip(out, into interface{}) (event, why string) {
	b, err, panicked := encodeRecovering(out)
	if panicked != "" {
		return fmt.Sprintf("encoding panicked (%s)", panicked), ""
	}
	if err != nil {
		return fmt.Sprintf("encoding failed: %v", err), ""
	}
	n, err, panicked := decodeRecovering(b, into)
	if panicked != "" {
		return "", fmt.Sprintf("decoding the % x the encoder produced panicked (%s)", b, panicked)
	}
	if err != nil {
		return "", fmt.Sprintf("decoding the % x the encoder produced failed: %v", b, err)
	}
	if n != len(b) {
		return "", fmt.Sprintf("decode consumed %d of the %d bytes the encoder produced, "+
			"so the null encoding is the wrong length", n, len(b))
	}
	return "", ""
}
