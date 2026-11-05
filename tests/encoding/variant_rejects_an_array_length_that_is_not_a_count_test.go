// A flattened array length that is not a count of anything - any negative
// length other than -1 - is rejected, not allocated for.
//
// OPC 10000-6 §5.2.5 Arrays: "If an Array is null, then its length is
// encoded as -1." OPC 10000-6 §5.2.2.16 Variant: "Multi-dimensional arrays
// are encoded as a one-dimensional array and this field specifies the total
// number of elements." Together those two make -1 the only negative length
// the encoding defines: it means null, it is not a count. A length of -2 is
// therefore not a length at all, so a decoder has no number of elements to
// make a slice for and nothing to read; the only conforming move is to
// reject the packet.
//
// The neighbouring rows are
// variant_rejects_more_dimensions_than_the_message_can_hold_test.go, which
// is the dimension-count field rather than the array-length field, and
// variant_rejects_dimensions_inconsistent_with_the_array_length_test.go,
// which is the relation between the dimensions and the length this row
// decodes. The three are neighbours of each other and of nothing else in
// this directory: the datetime and nil-builtin rows read §5.2.2.5 and
// §5.1.2 Table 1, clauses about a different field of a different type, and
// none of them touches a Variant array field.
//
// The row runs against the shared helpers, with no scripted server in
// this package: every packet, including the null-array control, is
// decoded by decodeVariantRecovering, which turns the panic a bad length
// can raise into a value the case then reports instead of taking the
// binary down. See harness.go.
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
// No reference stack was read for this row; the two clauses and the
// measured behaviour are the whole of the evidence.

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
	const knownBug = "a negative flattened array length other than -1 is not rejected but passed to reflect.MakeSlice, which panics with \"reflect.MakeSlice: negative len\" and, since Decode has no recover, takes the process with it"

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
	case nullV.Value() == nil:
		t.Fatalf("could not set the test up: the null array, length -1, decoded to " +
			"no value at all rather than to a nil slice")
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

	// The arms, in the order the rule in tests/README.md asks for: the presence
	// with the constant standing is the green known bug; the presence gone but
	// the outcome still wrong is a fix that landed wrong, and red; the outcome
	// right with the constant standing asks for the constant to be deleted,
	// and is red; the outcome right with the constant gone is the row's
	// ordinary pass. What fits none of those - the panic with the constant
	// deleted - falls to default, so deleting the constant without the fix
	// is red rather than silent.
	switch {
	case thePanic && knownBug != "":
		t.Logf("known bug: %s; §5.2.5 gives -1 the meaning null and gives no negative "+
			"length another, so none of these is a count and each must be rejected, "+
			"not allocated for:%s", knownBug, detail)
	case !thePanic && !rejected:
		t.Errorf("the \"reflect.MakeSlice: negative len\" panic is gone, so a change has "+
			"landed on this decode, but the outcome is still wrong: §5.2.5 gives -1 the "+
			"meaning null and gives no negative length another, so none of these is a "+
			"count and each must be rejected, and the decoder still does not reject "+
			"it - the change that removed the panic did not fix the behaviour:%s", detail)
	case rejected && knownBug != "":
		t.Errorf("this now passes (a flattened array length other than -1 is rejected, " +
			"and the null array still decodes) - " +
			"delete the knownBug constant, every arm that reads it, and anything only those arms used, in the commit that fixed it")
	case rejected:
		t.Logf("a flattened array length other than -1 is rejected, and the null array still decodes")
	default:
		t.Errorf("§5.2.5 gives -1 the meaning null and gives no negative length "+
			"another, so none of these is a count and each must be rejected, "+
			"not allocated for:%s", detail)
	}
}
