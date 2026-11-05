// A nil pointer to a nullable built-in type encodes as that type's null
// value, and a message carrying one still parses.
//
// OPC 10000-6 §5.1.2 Table 1 gives every built-in type a Nullable column and
// a Default Value column, and defines the first: "The Nullable column
// indicates whether a 'null' value exists for the DataType in all
// DataEncodings. A 'null' value is a value that is equivalent 'no value
// specified'."
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

// marker is written after the nullable field in the framing test. The value
// is arbitrary; it has to be one that a short or long null encoding will not
// produce by accident, so it is not zero and not a small integer.
const marker = uint32(0xDEADBEEF)

// The framing structs. Each has a nullable built-in first and the same uint32
// marker behind it. They are named types at package scope because the encoder
// dispatches on each field's static type, so the field cannot be an interface
// and the shape cannot be shared.
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

// TestNilBuiltinFieldKeepsStreamFraming encodes a struct whose first field is
// a nil pointer to a nullable built-in and whose second is a known marker,
// then decodes the result and reads the marker back.
//
// This is the failure the byte assertions in
// nil_builtin_encodes_as_its_null_value_test.go cannot see. They check one
// value in isolation; a null encoding of the wrong length is only harmful
// because everything after it in the same message shifts. A guard that wrote
// one byte too few for NodeId would still look like a plausible null value on
// its own, and would desynchronise every field behind it.
//
// It is also the only check here that the bytes a guard writes are bytes
// gopcua's own decoder accepts. Encoding a null value no decoder reads is not
// an improvement on panicking.
func TestNilBuiltinFieldKeepsStreamFraming(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. README.md says when to delete it. The
	// behaviour is closed by https://github.com/gopcua/opcua/pull/901.
	const knownBug = "a message whose field is a nil pointer to a nullable built-in does not survive encode and decode, because encoding it panics"

	cases := []struct {
		name string
		run  func() (uint32, string)
	}{
		{"NodeId", func() (uint32, string) {
			var in nodeIDFramed
			why := roundTrip(&nodeIDFramed{Marker: marker}, &in)
			return in.Marker, why
		}},
		{"Guid", func() (uint32, string) {
			var in guidFramed
			why := roundTrip(&guidFramed{Marker: marker}, &in)
			return in.Marker, why
		}},
		{"LocalizedText", func() (uint32, string) {
			var in localizedTextFramed
			why := roundTrip(&localizedTextFramed{Marker: marker}, &in)
			return in.Marker, why
		}},
		{"DataValue", func() (uint32, string) {
			var in dataValueFramed
			why := roundTrip(&dataValueFramed{Marker: marker}, &in)
			return in.Marker, why
		}},
		{"Variant", func() (uint32, string) {
			var in variantFramed
			why := roundTrip(&variantFramed{Marker: marker}, &in)
			return in.Marker, why
		}},
		{"DiagnosticInfo", func() (uint32, string) {
			var in diagnosticInfoFramed
			why := roundTrip(&diagnosticInfoFramed{Marker: marker}, &in)
			return in.Marker, why
		}},
		{"ExtensionObject", func() (uint32, string) {
			var in extensionObjectFramed
			why := roundTrip(&extensionObjectFramed{Marker: marker}, &in)
			return in.Marker, why
		}},
	}

	var detail string
	survives := true
	for _, c := range cases {
		got, why := c.run()
		ok := true
		var outcome string
		switch {
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
	case survives && knownBug != "":
		t.Errorf("this now passes (a struct with a nil field of each of the %d nullable "+
			"built-in types, followed by a uint32, survives encode and decode with the "+
			"uint32 intact) - "+
			"delete the knownBug constant and the two arms that read it, in the commit that fixed it",
			len(cases))
	case survives:
		t.Logf("a struct with a nil field of each of the %d nullable built-in types, "+
			"followed by a uint32, survives encode and decode with the uint32 intact",
			len(cases))
	case knownBug != "":
		t.Logf("known bug: %s; §5.1.2 Table 1 lists every type below as nullable, so a nil "+
			"field has a value to encode and the uint32 behind it must come back as it went:%s",
			knownBug, detail)
	default:
		t.Errorf("§5.1.2 Table 1 lists every type below as nullable, so a nil field has a "+
			"value to encode and the uint32 behind it must come back as it went:%s", detail)
	}
}

// roundTrip encodes out and decodes the bytes into into. It returns the
// reason it could not, empty when nothing went wrong; the caller then reads
// its own marker field, which it alone has the concrete type for. Only the
// test function above turns either into a failure.
func roundTrip(out, into interface{}) string {
	b, err, panicked := encodeRecovering(out)
	if panicked != "" {
		return fmt.Sprintf("encoding panicked (%s)", panicked)
	}
	if err != nil {
		return fmt.Sprintf("encoding failed: %v", err)
	}
	if _, err := ua.Decode(b, into); err != nil {
		return fmt.Sprintf("decoding the % x it produced failed: %v", b, err)
	}
	return ""
}
