// A flattened array length that is not a count of anything - any negative
// length other than -1 - is rejected, not allocated for.
//
// OPC 10000-6 §5.2.5 Arrays: "If an Array is null, then its length is
// encoded as -1." OPC 10000-6 §5.2.2.16 Variant: "Multi-dimensional arrays
// are encoded as a one-dimensional array and this field specifies the total
// number of elements." Together those two make -1 the only negative length
// the encoding defines: it means null, it is not a count. A length of -2 is
// therefore not a length at all, so a decoder has no number of elements to
// make a slice for and nothing to read. What it should do instead the two
// clauses do not say, and the reference stacks split on it. This row
// requires the packet rejected, which is the stricter of the two readings
// and the one gopcua's own pull requests below implement; the split is
// declared at the end of this header.
//
// The neighbouring rows are
// variant_rejects_more_dimensions_than_the_message_can_hold_test.go, which
// is the dimension-count field rather than the array-length field, and
// variant_rejects_dimensions_inconsistent_with_the_array_length_test.go,
// which is the relation between the dimensions and the length this row
// decodes. The three are neighbours of each other and of nothing else in
// this directory: the datetime rows read §5.2.2.5 and the three nil-value
// rows, reference_description_with_nil_fields_encodes_test.go among them,
// read §5.1.2 Table 1 - clauses about a different field of a different
// type. None of the six touches a Variant array field; where a nil row
// names a Variant at all it is a null scalar, not an array.
//
// The row runs against the shared helpers, with no scripted server in
// this package: every packet, including the null-array control, is decoded
// by decodeVariantRecovering. See harness.go.
//
// The row decodes two packets, length -2 and the Int32 minimum, and
// requires each to come back rejected: an error, not a panic. It also
// decodes the null array, length -1, and requires that to keep working,
// because a fix that rejected every negative length would break the one
// negative length §5.2.5 defines: the null array must still decode to a
// nil slice, and this row must not let a fix through that breaks it. The
// row does not require a particular error - gopcua's own guards in this
// decode return StatusBadEncodingLimitsExceeded and the clauses here name
// no error - and it does not cover lengths above MaxVariantArrayLength,
// which gopcua already rejects before the slice type is reached.
//
// The row's verdict rests on the literal packets written out below;
// nothing here encodes anything, so gopcua is never its own yardstick.
//
// The reference stacks split on what this row requires, and ../README.md's
// sixth question asks a row that a known-good implementation would fail to
// name the stacks it fails. Two of the three fail it, both by treating
// every negative length as the null array: open62541's Array_decodeBinary
// returns UA_STATUSCODE_GOOD with a NULL array for any signed_length <= 0
// (ua_types_encoding_binary.c:530), and UA-.NETStandard's ReadArrayLength
// maps any negative length to -1 (BinaryDecoder.cs:2358). Milo rejects, and
// rejects in this row's own shape: its Variant path returns the null value
// for -1 first, then checkArrayLength throws Bad_DecodingError for whatever
// negative length is left (OpcUaBinaryDecoder.java:414, :421, :1400). The
// -1 control below is what keeps this row from licensing the lumping move
// Milo avoids - rejecting every negative length, the null array with it.
//
// Why this row takes the stricter reading, given two stacks take the other:
// the length is a framing field, so nulling a length the sender did not mean
// as null leaves the read position short by however many elements it did
// write, and every later field in the message is then read from the wrong
// offset. Measured on a two-Variant message whose second Variant is the
// UInt32 7, with the first carrying a length of -2 and two booleans behind
// it: the lenient decode returns no error and the second Variant comes back
// false. Rejecting costs one response; nulling can hand the caller a wrong
// value for a field that was never malformed, and say nothing.

package encoding

import (
	"fmt"
	"testing"
)

func TestVariantRejectsAnArrayLengthThatIsNotACount(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. ../README.md says when to delete it. The
	// behaviour is closed by https://github.com/gopcua/opcua/pull/892 and
	// https://github.com/gopcua/opcua/pull/936.
	const knownBug = "a negative flattened array length other than -1 is not rejected but passed to reflect.MakeSlice, which panics and, since Decode has no recover, takes the process with it"

	// mask 0x81 is Boolean | array values; the four bytes behind it are the
	// flattened array length §5.2.2.16 defines. Both lengths below are
	// negative and neither is -1, so §5.2.5 gives neither a meaning.
	probes := []struct {
		name   string
		packet []byte
	}{
		{"length -2", []byte{0x81, 0xfe, 0xff, 0xff, 0xff}},
		{"the Int32 minimum", []byte{0x81, 0x00, 0x00, 0x00, 0x80}},
	}

	// The control: -1 is the null array, the one negative length the clause
	// defines. It decodes today and the row requires it to keep decoding,
	// so a fix that rejects every negative length cannot pass.
	nullV, err, panicked := decodeVariantRecovering([]byte{0x81, 0xff, 0xff, 0xff, 0xff})
	switch {
	case panicked != "" || err != nil:
		t.Fatalf("could not set the test up: the null array, length -1, no longer "+
			"decodes (err %v, panic %q)", err, panicked)
	default:
		if s, ok := nullV.Value().([]bool); !ok || s != nil {
			t.Fatalf("could not set the test up: the null array, length -1, decoded to "+
				"%#v rather than to a nil slice", nullV.Value())
		}
	}

	// The presence the knownBug arm keys on is the panic itself, "reflect.MakeSlice:
	// negative len", per the rule in tests/README.md: the panic is what says
	// the decoder reached this stimulus and did the wrong thing, so the arm
	// below logs green only while that panic still occurs. Keying it on the
	// absence of the rejection instead would log a decoder that no longer
	// panics but still does not reject - a fix that landed wrong - green
	// beside the unfixed decoder, and the arm that asks for the constant to
	// be deleted could never fire on a wrong fix.
	const makeSlicePanic = "reflect.MakeSlice: negative len"

	var detail string
	rejected := true
	thePanic := false
	for _, p := range probes {
		_, err, panicked := decodeVariantRecovering(p.packet)
		var outcome string
		switch {
		case panicked == makeSlicePanic:
			thePanic = true
			rejected = false
			outcome = fmt.Sprintf("panicked (%s), where -1 being the only negative "+
				"length means there is nothing here to allocate for", panicked)
		case panicked != "":
			rejected = false
			outcome = fmt.Sprintf("panicked (%s), which is not the panic the known bug "+
				"names, so a change has reached this decode", panicked)
		case err == nil:
			rejected = false
			outcome = "decoded, where -1 being the only negative length means this is not a count of anything"
		default:
			outcome = fmt.Sprintf("rejected with %v", err)
		}
		detail += fmt.Sprintf("\n    %-16s %s", p.name, outcome)
	}

	// notACount is the inference the three arms below share: it is the reason
	// each outcome is a failure, and it is one sentence in one place so a
	// change to the reading cannot land in two of the three.
	const notACount = "§5.2.5 gives -1 the meaning null and gives no negative " +
		"length another, so none of these is a count and each must be rejected"

	// The arms, keyed as the rule in tests/README.md asks: the presence
	// with the constant standing is the green known bug; the presence gone but
	// the outcome still wrong is a fix that landed wrong, and red; the outcome
	// right with the constant standing asks for the constant to be deleted,
	// and is red; the outcome right with the constant gone is the row's
	// ordinary pass. What fits none of those - the panic with the constant
	// deleted - falls to default, so deleting the constant without the fix
	// is red rather than silent.
	switch {
	case thePanic && knownBug != "":
		t.Logf("known bug: %s; %s, not allocated for:%s", knownBug, notACount, detail)
	case !thePanic && !rejected:
		t.Errorf("the %q panic is gone, so a change has landed on this decode, but the "+
			"outcome is still wrong: %s, and the decoder still does not reject it - "+
			"the change that removed the panic did not fix the behaviour:%s",
			makeSlicePanic, notACount, detail)
	case rejected && knownBug != "":
		t.Errorf("this now passes (a flattened array length other than -1 is rejected, " +
			"and the null array still decodes) - " +
			"delete the knownBug constant, every arm that reads it, and " +
			"anything only those arms used, in the commit that fixed it")
	case rejected:
		t.Logf("a flattened array length other than -1 is rejected, and the null array still decodes")
	default:
		t.Errorf("%s, not allocated for:%s", notACount, detail)
	}
}
