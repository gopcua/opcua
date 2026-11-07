//go:build conformance

// An instant goes onto the wire as the tick count that names it, for every
// instant the specification allows.
//
// OPC 10000-6 §5.2.2.5: "A DateTime value shall be encoded as a 64-bit signed
// integer (see 5.2.2.2) which represents the number of 100 nanosecond
// intervals since January 1, 1601 (UTC)."
//
// That sentence is the whole of the valid-range contract, and it is a
// definition, not a range check: for any instant in the range the clause
// permits, the tick count is determined.
//
// The neighbouring rows are
// datetime_decodes_every_tick_in_the_valid_range_test.go, the same
// clause read in the other direction - a tick count coming off the wire
// as the instant it names - and
// datetime_out_of_range_saturates_test.go, the clause's rules for tick
// counts outside the range this row sweeps.
//
// The row runs against the shared helpers, with no scripted server in
// this package: validRangeInstants supplies all but one of the instants
// swept - the exception is this row's own and is named at it - ticksFor
// computes each one's expected tick count, wireBytes renders the count
// as the eight expected bytes, and each instant is encoded inside a
// wireDateTime so ua.Encode takes the same reflection path a service
// response does. See harness.go.
//
// The expected tick count for an instant is computed here in math/big, from
// the clause's definition, with no int64 anywhere. That matters: the defect
// this file was written for is an int64 overflow in gopcua's own conversion,
// so an oracle that did the same arithmetic in int64 would overflow in the
// same place and agree with the bug. A big.Int oracle cannot.
//
// The sweep crosses the range at a step of one instant a year, plus the
// shared sweep's six probes and one more this row adds for itself.
// Three boundary points would pass a fix that was right at the
// boundaries and wrong in between; the overflow #906 fixed had a cliff
// one tick wide, so a sweep that crosses the range is what can find a
// second one.
//
// What the row deliberately does not assert: the sweep above is a
// sample, not every tick the clause permits, so a one-tick cliff away
// from a sampled point is invisible to it. And
// the instant tick 0 names, 1601-01-01T00:00:00Z itself, is not swept: the
// sweep is shared with the decodes row, and it leaves that instant out
// because gopcua reserves tick 0 for the zero time, which is the
// library's own convention and not something the clause requires.
//
// No reference stack was read for this row; the evidence is the math/big
// oracle above, which runs the clause's algebra in a type that cannot
// overflow and so shares none of gopcua's int64 arithmetic, pinned by the
// hand-written byte anchors below: instants whose eight bytes are written
// out from the clause's definition and checked before the sweep,
// so a defect in ticksFor or wireBytes cannot take the sweep's verdict
// with it.

package encoding

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
)

func TestDateTimeEncodesEveryInstantInTheValidRange(t *testing.T) {
	// gopcua did not do this until https://github.com/gopcua/opcua/pull/906
	// merged; it passes since.

	// The literal anchors, written out by hand from the clause's definition:
	// a tick count is the number of 100 nanosecond intervals since
	// 1601-01-01T00:00:00Z, so 1601-01-01T00:00:00Z is tick 0 (the instant is
	// not gopcua's zero time, and encodes as that tick), one tick later is
	// tick 1, 1970-01-01T00:00:00Z is 11644473600 seconds after the 1601 epoch
	// and so tick 116444736000000000, and 9999-12-31T23:59:59Z is tick
	// 2650467743990000000. Each is encoded and compared against its bytes
	// before the sweep, so the anchors hold even if the sweep never runs.
	anchors := []struct {
		instant time.Time
		want    []byte
	}{
		{time.Date(1601, time.January, 1, 0, 0, 0, 0, time.UTC),
			[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{time.Date(1601, time.January, 1, 0, 0, 0, 100, time.UTC),
			[]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC),
			[]byte{0x00, 0x80, 0x3e, 0xd5, 0xde, 0xb1, 0x9d, 0x01}},
		{time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC),
			[]byte{0x80, 0xa9, 0x27, 0xd1, 0x5e, 0x5a, 0xc8, 0x24}},
	}
	for _, a := range anchors {
		got, err := ua.Encode(&wireDateTime{V: a.instant})
		if err != nil {
			t.Fatalf("could not set the test up: the literal anchor for %s failed "+
				"to encode: %v", a.instant.Format(time.RFC3339Nano), err)
		}
		if !bytes.Equal(got, a.want) {
			t.Fatalf("could not set the test up: %s encodes as % x where the "+
				"hand-written anchor says % x",
				a.instant.Format(time.RFC3339Nano), got, a.want)
		}
	}

	// subTick is this row's own, not part of the shared sweep: every instant
	// in validRangeInstants is a whole number of ticks, which is what blinds
	// the sweep to the truncation half of the defect #906 fixed. It cannot
	// be added there, because ticksFor floors and the decode row would then
	// score an instant no tick count names.
	subTick := time.Date(1900, time.June, 15, 13, 45, 30, 123456789, time.UTC)
	shared := validRangeInstants()
	if len(shared) == 0 {
		t.Fatal("the sweep is empty - this row would report a pass having " +
			"checked nothing")
	}
	instants := append(shared, subTick)

	mismatches := 0
	var first string
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
			continue
		}
		if !bytes.Equal(got, want) {
			mismatches++
			if first == "" {
				first = fmt.Sprintf("%s, which §5.2.2.5 makes tick count %s and % x on "+
					"the wire, encoded as % x",
					in.Format(time.RFC3339Nano), ticksFor(in), want, got)
			}
		}
	}
	encodes := mismatches == 0

	switch {
	case encodes:
		t.Logf("every one of the %d instants in the sweep encoded as the tick count "+
			"§5.2.2.5 defines for it", len(instants))
	default:
		t.Errorf("§5.2.2.5 defines an instant as a number of 100 nanosecond intervals since "+
			"1601-01-01T00:00:00Z, so each one encodes as the tick count that names it; %d of "+
			"the %d instants in the sweep encoded as the wrong tick count, and the first is %s",
			mismatches, len(instants), first)
	}
}
