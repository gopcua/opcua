// OPC 10000-6 §5.2.2.5: "A DateTime value shall be encoded as a 64-bit signed
// integer (see 5.2.2.2) which represents the number of 100 nanosecond
// intervals since January 1, 1601 (UTC)."
//
// That sentence is the whole of the valid-range contract, and it is a
// definition, not a range check: for any tick count in the range the clause
// permits, the decoded instant is determined.
//
// The expected tick count for an instant is computed here in math/big, from
// the clause's definition, with no int64 anywhere. That matters: the defect
// this file was written for is an int64 overflow in gopcua's own conversion,
// so an oracle that did the same arithmetic in int64 would overflow in the
// same place and agree with the bug. A big.Int oracle cannot.
//
// The range is swept, not sampled. Three boundary points would pass a fix
// that was right at the boundaries and wrong in between; the overflow being
// fixed has a cliff one tick wide, so a sweep is what finds a second one.
//
// The row runs against the shared helpers, with no scripted server in
// this package: validRangeInstants supplies the instants swept, ticksFor
// computes each one's expected tick count, wireBytes renders the count
// as the eight expected bytes, and each instant is encoded inside a
// wireDateTime so ua.Encode takes the same reflection path a service
// response does. See harness.go.
//
// The known bug has a signature, and the arms key on it: the instants inside
// roughly 1678-2262 encode rightly today, only those outside come back
// wrong, and each wrong one carries exactly the tick count the known
// conversion's arithmetic predicts - defectEncode below restates that
// arithmetic. Keying on the signature is what tells the known bug from a fix
// that landed and is wrong: a clamping fix leaves the same instants wrong,
// with different values, and an arm keyed on "something was wrong" cannot
// see the difference.

package encoding

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
)

// defectEncode restates the conversion ua/buffer.go's WriteTime runs today,
// the arithmetic the knownBug constant names. It is not the oracle the row
// asserts against - that is ticksFor, which computes the clause's answer in
// math/big, with no int64 anywhere. defectEncode is the known bug's
// fingerprint: the arms use it to tell the known bug, whose every wrong
// encoding is exactly the value this conversion predicts, from a fix that
// landed and is wrong, whose encodings carry values this conversion produces
// for no instant.
func defectEncode(v time.Time) []byte {
	d := make([]byte, 8)
	if !v.IsZero() {
		ts := uint64(v.UTC().UnixNano()/100 + 116444736000000000)
		binary.LittleEndian.PutUint64(d, ts)
	}
	return d
}

func TestDateTimeEncodesEveryInstantInTheValidRange(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. README.md says when to delete it. The
	// behaviour is closed by https://github.com/gopcua/opcua/pull/906.
	const knownBug = "an instant inside the range §5.2.2.5 defines does not encode to the tick count that clause names, outside roughly 1678-2262: ua/buffer.go's WriteTime does v.UTC().UnixNano()/100, and UnixNano is undefined outside that window, so the wire does not carry the number of 100 nanosecond intervals since 1601-01-01T00:00:00Z"

	instants := validRangeInstants()
	if len(instants) == 0 {
		t.Fatal("the sweep is empty - this row would report a pass having " +
			"checked nothing")
	}

	mismatches := 0
	// unexplained counts the cases that match neither the clause's tick
	// count nor the known bug's predicted one.
	unexplained := 0
	var first, firstUnexplained string
	for _, in := range instants {
		want := wireBytes(ticksFor(in))

		got, err := ua.Encode(&wireDateTime{V: in})
		if err != nil {
			mismatches++
			if first == "" {
				first = fmt.Sprintf("%s, which §5.2.2.5 makes tick count %s and % x on "+
					"the wire, failed to encode: %v",
					in.Format(time.RFC3339Nano), ticksFor(in), want, err)
			}
			unexplained++
			if firstUnexplained == "" {
				firstUnexplained = fmt.Sprintf("%s, which §5.2.2.5 makes tick count %s "+
					"and % x on the wire, failed to encode, where the known bug's "+
					"arithmetic makes it % x",
					in.Format(time.RFC3339Nano), ticksFor(in), want, defectEncode(in))
			}
			continue
		}
		if !bytes.Equal(got, want) {
			mismatches++
			if first == "" {
				first = fmt.Sprintf("%s, which §5.2.2.5 makes tick count %s and % x on "+
					"the wire, encoded as % x",
					in.Format(time.RFC3339Nano), ticksFor(in), want, got)
			}
			if !bytes.Equal(got, defectEncode(in)) {
				unexplained++
				if firstUnexplained == "" {
					firstUnexplained = fmt.Sprintf("%s, which §5.2.2.5 makes tick count %s "+
						"and % x on the wire and the known bug's arithmetic makes % x, "+
						"encoded as % x, matching neither",
						in.Format(time.RFC3339Nano), ticksFor(in), want, defectEncode(in), got)
				}
			}
		}
	}
	encodes := mismatches == 0
	signatureHolds := unexplained == 0

	// Arm order: the two encodes arms come first, because a sweep that
	// encodes every instant rightly satisfies the signature vacuously and
	// must reach the arm that says so; the signature arm comes before the
	// wrong-fix arm, because the wrong-fix arm is defined by the
	// signature's absence; and the wrong-fix arm reads the constant, so
	// with the constant empty a missing behaviour lands in the default arm
	// below it.
	switch {
	case encodes && knownBug != "":
		t.Errorf("this now passes (every one of the %d instants in the sweep encoded as the "+
			"tick count §5.2.2.5 defines for it) - "+
			"delete the knownBug constant and every arm that reads it, in the commit that fixed it",
			len(instants))
	case encodes:
		t.Logf("every one of the %d instants in the sweep encoded as the tick count "+
			"§5.2.2.5 defines for it", len(instants))
	case signatureHolds && knownBug != "":
		t.Logf("known bug: %s; %d of the %d instants in the sweep encoded as the wrong tick "+
			"count, each exactly the one the known bug's arithmetic predicts, and the "+
			"first is %s",
			knownBug, mismatches, len(instants), first)
	case knownBug != "":
		t.Errorf("%d of the %d instants in the sweep encoded as the wrong tick count, and %d "+
			"of them match neither the tick count §5.2.2.5 defines nor the one the known "+
			"bug's arithmetic predicts - a fix for this landed and is wrong; the first "+
			"case matching neither is %s",
			mismatches, len(instants), unexplained, firstUnexplained)
	default:
		t.Errorf("§5.2.2.5 defines an instant as a number of 100 nanosecond intervals since "+
			"1601-01-01T00:00:00Z, so each one encodes as the tick count that names it; %d of "+
			"the %d instants in the sweep encoded as the wrong tick count, and the first is %s",
			mismatches, len(instants), first)
	}
}
