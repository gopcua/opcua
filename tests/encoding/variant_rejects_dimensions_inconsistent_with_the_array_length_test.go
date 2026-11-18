//go:build conformance

// Dimensions whose product is not the array length are rejected, whatever
// the product's size.
//
// OPC 10000-6 §5.2.2.16 Variant: "Multi-dimensional arrays are encoded as
// a one-dimensional array and this field specifies the total number of
// elements", "The length of each dimension encoded as a sequence of Int32
// values", "All dimensions shall be specified and shall be greater than
// zero", and: "If ArrayDimensions are inconsistent with the ArrayLength
// then the decoder shall stop and raise a Bad_DecodingError."
//
// The last sentence is exact and says what must happen. The packet below
// declares an array length of 4 with dimensions [4, 1073741825]: both are
// greater than zero, so the "greater than zero" clause leaves them alone,
// and four elements are present, so the element decode succeeds. The
// product is 2^32+4, not 4, so the dimensions are inconsistent with the
// length and the decoder must stop. gopcua multiplied the dimensions in an
// int32, so the product wrapped to exactly 4, the check passed, and the
// decode succeeded with dimensions a four-element variant cannot have, until
// the fix for https://github.com/gopcua/opcua/pull/953 made the decoder stop.
//
// The neighbouring rows are
// variant_rejects_an_array_length_that_is_not_a_count_test.go, which is the
// array-length field alone, and
// variant_rejects_more_dimensions_than_the_message_can_hold_test.go, which
// is the dimension-count field. This row is the relation between the two
// fields: the length and the dimensions are each individually well-formed,
// and it is their product that is wrong. The three are neighbours of each
// other and of nothing else in this directory: the datetime rows read
// §5.2.2.5 and the three nil-value rows,
// reference_description_with_nil_fields_encodes_test.go among them, read
// §5.1.2 Table 1 - clauses about a different field of a different type.
// None of the six touches a Variant array field; where a nil row names a
// Variant at all it is a null scalar, not an array.
//
// The row runs against ua.Decode, with no scripted server in this
// package: both of its decodes go through the decodeVariantRecovering its
// two siblings share, which lives in harness.go. See harness.go.
//
// The row asserts that decoding the packet stops with an error, and that a
// consistent control - the same four elements with dimensions [2, 2], whose
// product is the length - keeps decoding, so a fix that rejected every
// dimensioned array cannot pass. It deliberately does not require the
// error to be exactly Bad_DecodingError: the clause names it, but requiring
// gopcua's error taxonomy rather than the stopping would keep a fix that
// stops and raises some other error red for the wrong reason; the detail
// line reports what the decoder did raise. Both decodes go through the
// shared recover, so a panic in the decode becomes this row's outcome
// instead of ending the run; the clause's required outcome is an error,
// not a crash, and a panic is scored as neither - it lands in the red arm
// with its reason in the detail line.
//
// All three reference stacks on disk stop on this packet, so none of them
// fails this row and no exception under ../README.md's sixth question is
// needed. open62541 accumulates the product in a size_t, which does not
// wrap where gopcua's int32 does, and compares it against the array length
// (ua_types_encoding_binary.c:1407-1415, UA_STATUSCODE_BADDECODINGERROR).
// UA-.NETStandard reaches MatrixOf.IsValidMatrix, which multiplies in a
// long and gives up once the product passes int.MaxValue (MatrixOf.cs:828
// and :829), and the caller raises BadDecodingError
// (BinaryDecoder.cs:1830, :1832). Milo stops for a different reason than
// either: rather than checking the product, calculateMatrixLength checks
// each dimension on its own as it goes (OpcUaBinaryDecoder.java:1425), and
// 1073741825 is past its default two-megabyte maximum message size
// (EncodingLimits.java:23), so checkArrayLength raises
// Bad_EncodingLimitsExceeded (:1406) on the second dimension. The
// Math.multiplyExact below it runs once, on the harmless first dimension,
// and never sees the one that overflows.
//
// Milo naming Bad_EncodingLimitsExceeded where the clause names
// Bad_DecodingError is the reason this row requires the decode to stop
// rather than requiring a particular status: all three stacks stop, and
// they do not agree on what to call it.

package encoding

import (
	"fmt"
	"testing"
)

func TestVariantRejectsDimensionsInconsistentWithTheArrayLength(t *testing.T) {
	// gopcua did not do this until https://github.com/gopcua/opcua/pull/953
	// merged; it passes since.

	// The control: the same four Boolean elements with dimensions [2, 2],
	// whose product is the array length. It decodes today and the row
	// requires it to keep decoding.
	control := []byte{0xc1, 0x04, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x01,
		0x02, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00}
	if _, err, panicked := decodeVariantRecovering(control); err != nil || panicked != "" {
		t.Errorf("gopcua's own behaviour made this run unscoreable: dimensions [2, 2] with an "+
			"array length of 4 are consistent and no longer decode (err %v, panicked %s)", err, panicked)
		return
	}

	// The probe: array length 4, four Boolean elements, dimensions
	// [4, 1073741825]. The product is 2^32+4 and wraps to 4 in an int32.
	packet := []byte{0xc1, 0x04, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x01,
		0x02, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x40}
	v, err, panicked := decodeVariantRecovering(packet)
	if panicked != "" {
		// A panic is neither the stop the clause requires nor a decode: it
		// is scored as neither, and lands in the red arm with its reason in
		// the detail line.
		err = fmt.Errorf("panicked: %s", panicked)
	}

	stops := err != nil && panicked == ""
	const (
		// inconsistent names the packet's defect, in the words the detail line
		// and the wrong-fix arm below share.
		inconsistent = "dimensions [4, 1073741825] have the product 2^32+4 " +
			"against an array length of 4"
		// shallStop is the clause's requirement, in the words the detail
		// line's arm and the wrong-fix arm below share.
		shallStop = "§5.2.2.16 says the decoder shall stop"
		// fixed is what the row says once the decoder stops, in the words the
		// pass line below uses.
		fixed = "dimensions whose product is not the array length are " +
			"rejected, and consistent dimensions still decode"
	)
	// The presence the wrong-fix arm keys on, per the rule in
	// tests/README.md, is the decode succeeding with the packet's own
	// inconsistent dimensions still reported: a change that reaches this
	// decode but still accepts the packet no longer shows the presence,
	// which is what lets the wrong-fix arm below fire.
	dims := v.ArrayDimensions()
	dimsReported := len(dims) == 2 && dims[0] == 4 && dims[1] == 1073741825
	detail := fmt.Sprintf("\n    decoding % x, whose "+inconsistent+
		", returned err %v, panicked (%s) and ArrayDimensions() %v", packet, err, panicked, dims)

	// The arms: the outcome right is the row's ordinary pass; a change that
	// reaches this decode but leaves the outcome wrong - the packet still
	// accepted, but no longer with its own dimensions on it - is a fix that
	// landed wrong, and red. What fits neither - the decode succeeding with
	// its dimensions - falls to default, red.
	switch {
	case err == nil && !dimsReported:
		t.Errorf("the decode no longer reports the packet's own dimensions, so a "+
			"change has landed on this decode, but the outcome is still wrong: the "+
			inconsistent+", "+shallStop+", and the decoder still accepts the "+
			"packet - the change that altered the dimensions did "+
			"not make the decoder stop:%s", detail)
	case stops:
		t.Logf(fixed)
	default:
		t.Errorf("§5.2.2.16 says \"If ArrayDimensions are inconsistent with the "+
			"ArrayLength then the decoder shall stop and raise a Bad_DecodingError\", and "+
			"the dimensions here are inconsistent with the length:%s", detail)
	}
}
