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
// length and the decoder must stop. gopcua multiplies the dimensions in an
// int32, the product wraps to exactly 4, the check passes, and the decode
// succeeds with dimensions a four-element variant cannot have.
//
// The neighbouring rows are
// variant_rejects_an_array_length_that_is_not_a_count_test.go, which is the
// array-length field alone, and
// variant_rejects_more_dimensions_than_the_message_can_hold_test.go, which
// is the dimension-count field. This row is the relation between the two
// fields: the length and the dimensions are each individually well-formed,
// and it is their product that is wrong. The three are neighbours of each
// other and of nothing else in this directory: the datetime and nil-builtin
// rows read §5.2.2.5 and §5.1.2 Table 1, clauses about a different field of
// a different type, and none of them touches a Variant array field.
//
// The row runs against ua.Decode alone, with no scripted server in this
// package: it needs no helper, and the decodeVariantRecovering its two
// siblings share lives in harness.go. See harness.go.
//
// The row asserts that decoding the packet stops with an error, and that a
// consistent control - the same four elements with dimensions [2, 2], whose
// product is the length - keeps decoding, so a fix that rejected every
// dimensioned array cannot pass. It deliberately does not require the
// error to be exactly Bad_DecodingError: the clause names it, but requiring
// gopcua's error taxonomy rather than the stopping would keep a fix that
// stops and raises some other error red for the wrong reason; the detail
// line reports what the decoder did raise. The row drives ua.Decode
// directly, without the recover the two sibling rows share a helper for:
// nothing in this packet's path can take the process down in the current
// code - it decodes - and the clause's required outcome is an error, not a
// crash.
//
// All three reference stacks on disk stop on this packet, so none of them
// fails this row and no exception under ../README.md's sixth question is
// needed. open62541 accumulates the product in a size_t, which does not
// wrap where gopcua's int32 does, and compares it against the array length
// (ua_types_encoding_binary.c:1407-1415, UA_STATUSCODE_BADDECODINGERROR).
// UA-.NETStandard reaches MatrixOf.IsValidMatrix, which multiplies in a
// long and gives up once the product passes int.MaxValue (MatrixOf.cs:828
// and :829), and the caller raises BadDecodingError
// (BinaryDecoder.cs:1830, :1832). Milo stops earlier than the other two and
// for a different reason: calculateMatrixLength checks each dimension on
// its own before multiplying anything (OpcUaBinaryDecoder.java:1425), and
// 1073741825 is past its default two-megabyte maximum message size, so
// checkArrayLength raises Bad_EncodingLimitsExceeded (:1406) and the
// Math.multiplyExact below it is never reached.
//
// Milo naming Bad_EncodingLimitsExceeded where the clause names
// Bad_DecodingError is the reason this row requires the decode to stop
// rather than requiring a particular status: all three stacks stop, and
// they do not agree on what to call it.

package encoding

import (
	"fmt"
	"testing"

	"github.com/gopcua/opcua/ua"
)

func TestVariantRejectsDimensionsInconsistentWithTheArrayLength(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. ../README.md says when to delete it. No open pull
	// request closes this behaviour: neither
	// https://github.com/gopcua/opcua/pull/892 nor
	// https://github.com/gopcua/opcua/pull/893 nor
	// https://github.com/gopcua/opcua/pull/936 fixes it, so the constant
	// stands until whichever change makes the decoder stop does.
	const knownBug = "dimensions whose product is not the array length are accepted when the product overflows: the consistency check multiplies the dimensions in an int32, so dimensions [4, 1073741825] wrap to exactly the array length 4 and the decode succeeds"

	// The control: the same four Boolean elements with dimensions [2, 2],
	// whose product is the array length. It decodes today and the row
	// requires it to keep decoding.
	control := []byte{0xc1, 0x04, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x01,
		0x02, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00}
	var cv ua.Variant
	if _, err := ua.Decode(control, &cv); err != nil {
		t.Fatalf("could not set the test up: dimensions [2, 2] with an array length "+
			"of 4 are consistent and no longer decode: %v", err)
	}

	// The probe: array length 4, four Boolean elements, dimensions
	// [4, 1073741825]. The product is 2^32+4 and wraps to 4 in an int32.
	packet := []byte{0xc1, 0x04, 0x00, 0x00, 0x00, 0x01, 0x01, 0x01, 0x01,
		0x02, 0x00, 0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x40}
	v := new(ua.Variant)
	_, err := ua.Decode(packet, v)

	stops := err != nil
	const (
		// inconsistent names the packet's defect, in the words the detail line
		// and the wrong-fix arm below share.
		inconsistent = "dimensions [4, 1073741825] have the product 2^32+4 " +
			"against an array length of 4"
		// shallStop is the clause's requirement, in the words the known-bug
		// line and the wrong-fix arm below share.
		shallStop = "§5.2.2.16 says the decoder shall stop"
		// fixed is what the row logs once the decoder stops.
		fixed = "dimensions whose product is not the array length are " +
			"rejected, and consistent dimensions still decode"
	)
	// The presence the knownBug arm keys on, per the rule in tests/README.md,
	// is the decode succeeding with the packet's own inconsistent dimensions
	// still reported. A fixed library stops, so it never shows this presence;
	// a change that accepts the packet some other way shows the acceptance
	// without the presence, which is what lets the wrong-fix arm below fire.
	dims := v.ArrayDimensions()
	dimsReported := len(dims) == 2 && dims[0] == 4 && dims[1] == 1073741825
	decoded := err == nil && dimsReported
	detail := fmt.Sprintf("\n    decoding % x, whose "+inconsistent+
		", returned err %v and ArrayDimensions() %v", packet, err, dims)

	// The arms, keyed as the rule asks for: the presence with the
	// constant standing is the green known bug; the presence gone but the
	// outcome still wrong - the packet still accepted, but no longer with
	// its own dimensions on it, so a change has reached this decode - is a
	// fix that landed wrong, and red; the outcome right with the constant
	// standing asks for the constant to be deleted, and is red; the outcome
	// right with the constant gone is the row's ordinary pass. What fits none
	// of those - the decode still succeeding with its dimensions, and the
	// constant deleted - falls to default, so deleting the constant without
	// the fix is red rather than silent.
	switch {
	case decoded && knownBug != "":
		t.Logf("known bug: %s; "+shallStop+" when the dimensions are "+
			"inconsistent with the array length:%s", knownBug, detail)
	case err == nil && !dimsReported:
		t.Errorf("the decode no longer reports the packet's own dimensions, so a "+
			"change has landed on this decode, but the outcome is still wrong: the "+
			inconsistent+", "+shallStop+", and the decoder still accepts the "+
			"packet - the change that altered the dimensions did "+
			"not make the decoder stop:%s", detail)
	case stops && knownBug != "":
		t.Errorf("this now passes (" + fixed + ") - " +
			"delete the knownBug constant, every arm that reads it, and " +
			"anything only those arms used, in the commit that fixed it")
	case stops:
		t.Logf(fixed)
	default:
		t.Errorf("§5.2.2.16 says \"If ArrayDimensions are inconsistent with the "+
			"ArrayLength then the decoder shall stop and raise a Bad_DecodingError\", and "+
			"the dimensions here are inconsistent with the length:%s", detail)
	}
}
