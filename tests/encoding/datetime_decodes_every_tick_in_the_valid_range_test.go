// A DateTime on the wire decodes to the instant its tick count names, for
// every instant the specification allows.
//
// OPC 10000-6 §5.2.2.5: "A DateTime value shall be encoded as a 64-bit signed
// integer (see 5.2.2.2) which represents the number of 100 nanosecond
// intervals since January 1, 1601 (UTC)."
//
// That sentence is the whole of the valid-range contract, and it is a
// definition, not a range check: for any tick count in the range the clause
// permits, the decoded instant is determined.
//
// The neighbouring rows are
// datetime_encodes_every_instant_in_the_valid_range_test.go, the same
// clause read in the other direction - an instant going onto the wire as
// the tick count that names it - and
// datetime_out_of_range_saturates_test.go, the clause's rules for tick
// counts outside the range this row sweeps.
//
// The row runs against the shared helpers, with no scripted server in
// this package: validRangeInstants supplies the instants swept, ticksFor
// computes each one's expected tick count, and wireBytes renders the
// count as the eight bytes ua.Decode reads back into a wireDateTime, so
// the decode takes the same reflection path a service response does,
// rather than Buffer.ReadTime directly; the reported defect is only
// interesting if it reaches a caller. See harness.go.
//
// The expected tick count for an instant is computed here in math/big, from
// the clause's definition, with no int64 anywhere. That matters: the defect
// this file was written for is an int64 overflow in gopcua's own conversion,
// so an oracle that did the same arithmetic in int64 would overflow in the
// same place and agree with the bug. A big.Int oracle cannot.
//
// The sweep crosses the range at a step of one instant a year, plus four
// probes. Three boundary points would pass a fix that was right at the
// boundaries and wrong in between; the overflow being fixed has a cliff
// one tick wide, so a sweep that crosses the range is what can find a
// second one.

//
// The known bug has a signature, and the arms key on it: the instants inside
// roughly 1678-2262 decode rightly today, only those outside come back
// wrong, and each wrong one carries exactly the instant the known
// conversion's arithmetic predicts - defectDecode below restates that
// arithmetic. Keying on the signature is what tells the known bug from a fix
// that landed and is wrong: a clamping fix leaves the same instants wrong,
// with different values, and an arm keyed on "something was wrong" cannot
// see the difference.
//
// What the row deliberately does not assert: the sweep above is a
// sample, not every tick the clause permits, so a one-tick cliff away
// from a sampled point is invisible to it. And
// tick 0, the clause's own low endpoint, 1601-01-01T00:00:00Z itself, is
// not swept: gopcua reserves tick 0 for the zero time, which is the
// library's own convention and not something the clause requires.
//
// No reference stack was read for this row; the math/big oracle above,
// which runs the clause's algebra in a type that cannot overflow and so
// shares none of gopcua's int64 arithmetic, is the whole of the
// evidence.

package encoding

import (
	"fmt"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
)

// defectDecode restates the conversion ua/buffer.go's ReadTime runs today,
// the arithmetic the knownBug constant names. It is not the oracle the row
// asserts against - that is ticksFor, which computes the clause's answer in
// math/big, with no int64 anywhere. defectDecode is the known bug's
// fingerprint: the arms use it to tell the known bug, whose every wrong
// decode is exactly the value this conversion predicts, from a fix that
// landed and is wrong, whose decodes carry values this conversion produces
// for no tick count.
func defectDecode(ts uint64) time.Time {
	if ts == 0 {
		return time.Time{}
	}
	return time.Unix(0, int64((ts-116444736000000000)*100)).UTC()
}

func TestDateTimeDecodesEveryTickInTheValidRange(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. ../README.md says when to delete it. The
	// behaviour is closed by https://github.com/gopcua/opcua/pull/906.
	const knownBug = "a tick count inside the range §5.2.2.5 defines does not decode to the instant that clause names, outside roughly 1678-2262: ua/buffer.go's ReadTime does time.Unix(0, int64((ts-116444736000000000)*100)), so the unsigned subtraction underflows below 1970 and the int64 nanosecond multiply overflows above 2262"

	instants := validRangeInstants()
	if len(instants) == 0 {
		t.Fatal("the sweep is empty - this row would report a pass having " +
			"checked nothing")
	}

	mismatches := 0
	// unexplained counts the cases that match neither the clause's instant
	// nor the known bug's predicted one.
	unexplained := 0
	var first, firstUnexplained string
	for _, want := range instants {
		ticks := ticksFor(want)
		wire := wireBytes(ticks)

		var got wireDateTime
		if _, err := ua.Decode(wire, &got); err != nil {
			mismatches++
			if first == "" {
				first = fmt.Sprintf("tick count %s (wire % x), which §5.2.2.5 defines as %s, "+
					"failed to decode: %v",
					ticks, wire, want.Format(time.RFC3339Nano), err)
			}
			unexplained++
			if firstUnexplained == "" {
				firstUnexplained = fmt.Sprintf("tick count %s, which §5.2.2.5 defines as %s, "+
					"failed to decode, where the known bug's arithmetic decodes it to %s",
					ticks, want.Format(time.RFC3339Nano),
					defectDecode(ticks.Uint64()).Format(time.RFC3339Nano))
			}
			continue
		}
		if got.V.Equal(want) {
			continue
		}
		mismatches++
		if first == "" {
			first = fmt.Sprintf("tick count %s, which §5.2.2.5 defines as %s, "+
				"and it decoded as %s",
				ticks, want.Format(time.RFC3339Nano), got.V.UTC().Format(time.RFC3339Nano))
		}
		if !got.V.Equal(defectDecode(ticks.Uint64())) {
			unexplained++
			if firstUnexplained == "" {
				firstUnexplained = fmt.Sprintf("tick count %s, which §5.2.2.5 defines as %s "+
					"and the known bug's arithmetic decodes to %s, decoded as %s, "+
					"matching neither",
					ticks, want.Format(time.RFC3339Nano),
					defectDecode(ticks.Uint64()).Format(time.RFC3339Nano),
					got.V.UTC().Format(time.RFC3339Nano))
			}
		}
	}
	decodes := mismatches == 0
	signatureHolds := unexplained == 0

	// Arm order: the two decodes arms come first, because a sweep that
	// decodes every instant rightly satisfies the signature vacuously and
	// must reach the arm that says so; the signature arm comes before the
	// wrong-fix arm, because the wrong-fix arm is defined by the
	// signature's absence; and the wrong-fix arm reads the constant, so
	// with the constant empty a missing behaviour lands in the default arm
	// below it.
	switch {
	case decodes && knownBug != "":
		t.Errorf("this now passes (every one of the %d instants in the sweep decoded to "+
			"the instant its tick count names) - "+
			"delete the knownBug constant, every arm that reads it, and "+
			"anything only those arms used, in the commit that fixed it",
			len(instants))
	case decodes:
		t.Logf("every one of the %d instants in the sweep decoded to the instant its "+
			"tick count names", len(instants))
	case signatureHolds && knownBug != "":
		t.Logf("known bug: %s; %d of the %d instants in the sweep came back as the wrong "+
			"instant, each exactly the one the known bug's arithmetic predicts, and the "+
			"first is %s",
			knownBug, mismatches, len(instants), first)
	case knownBug != "":
		t.Errorf("%d of the %d instants in the sweep came back as the wrong instant, and %d "+
			"of them match neither the instant §5.2.2.5 defines nor the one the known "+
			"bug's arithmetic predicts - a fix for this landed and is wrong; the first "+
			"case matching neither is %s",
			mismatches, len(instants), unexplained, firstUnexplained)
	default:
		t.Errorf("§5.2.2.5 defines a tick count as a number of 100 nanosecond intervals since "+
			"1601-01-01T00:00:00Z, so each one decodes to the instant it names; %d of the %d "+
			"instants in the sweep came back as the wrong instant, and the first is %s",
			mismatches, len(instants), first)
	}
}
