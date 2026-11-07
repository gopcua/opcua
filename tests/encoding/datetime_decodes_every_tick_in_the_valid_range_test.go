//go:build conformance

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
// rather than Buffer.ReadTime directly; a defect here is only
// interesting if it reaches a caller. See harness.go.
//
// The expected tick count for an instant is computed here in math/big, from
// the clause's definition, with no int64 anywhere. That matters: the defect
// this file was written for is an int64 overflow in gopcua's own conversion,
// so an oracle that did the same arithmetic in int64 would overflow in the
// same place and agree with the bug. A big.Int oracle cannot.
//
// The sweep crosses the range at a step of one instant a year, plus six
// probes. Three boundary points would pass a fix that was right at the
// boundaries and wrong in between; the overflow #906 fixed had a cliff
// one tick wide, so a sweep that crosses the range is what can find a
// second one.
//
// What the row deliberately does not assert: the sweep above is a
// sample, not every tick the clause permits, so a one-tick cliff away
// from a sampled point is invisible to it. And
// tick 0, the clause's own low endpoint, 1601-01-01T00:00:00Z itself, is
// not swept: gopcua reserves tick 0 for the zero time, which is the
// library's own convention and not something the clause requires.
//
// No reference stack was read for this row; the evidence is the math/big
// oracle above, which runs the clause's algebra in a type that cannot
// overflow and so shares none of gopcua's int64 arithmetic, pinned by the
// hand-written byte anchors below: instants whose eight bytes are written
// out from the clause's definition and checked before the sweep, so a
// defect in ticksFor or wireBytes cannot take the sweep's verdict with it.

package encoding

import (
	"fmt"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
)

func TestDateTimeDecodesEveryTickInTheValidRange(t *testing.T) {
	// gopcua did not do this until https://github.com/gopcua/opcua/pull/906
	// merged; it passes since.

	// The literal anchors, written out by hand from the clause's definition:
	// a tick count is the number of 100 nanosecond intervals since
	// 1601-01-01T00:00:00Z, so one tick past that endpoint is tick 1,
	// 1970-01-01T00:00:00Z is 11644473600 seconds after the 1601 epoch and so
	// tick 116444736000000000, and 9999-12-31T23:59:59Z is tick
	// 2650467743990000000. Tick 0 itself, the endpoint instant, is not an
	// anchor, for the reason the header gives above. Each anchor's bytes are
	// decoded and compared against its instant before the sweep, so the
	// anchors hold even if the sweep never runs.
	anchors := []struct {
		wire []byte
		want time.Time
	}{
		{[]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			time.Date(1601, time.January, 1, 0, 0, 0, 100, time.UTC)},
		{[]byte{0x00, 0x80, 0x3e, 0xd5, 0xde, 0xb1, 0x9d, 0x01},
			time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)},
		{[]byte{0x80, 0xa9, 0x27, 0xd1, 0x5e, 0x5a, 0xc8, 0x24},
			time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC)},
	}
	for _, a := range anchors {
		var got wireDateTime
		if _, err := ua.Decode(a.wire, &got); err != nil {
			t.Fatalf("could not set the test up: the literal anchor % x failed "+
				"to decode: %v", a.wire, err)
		}
		if !got.V.Equal(a.want) {
			t.Fatalf("could not set the test up: the literal anchor % x decodes as "+
				"%s where the hand-written anchor says %s", a.wire,
				got.V.UTC().Format(time.RFC3339Nano), a.want.Format(time.RFC3339Nano))
		}
	}

	instants := validRangeInstants()
	if len(instants) == 0 {
		t.Fatal("the sweep is empty - this row would report a pass having " +
			"checked nothing")
	}

	mismatches := 0
	var first string
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
	}
	decodes := mismatches == 0

	switch {
	case decodes:
		t.Logf("every one of the %d instants in the sweep decoded to the instant its "+
			"tick count names", len(instants))
	default:
		t.Errorf("§5.2.2.5 defines a tick count as a number of 100 nanosecond intervals since "+
			"1601-01-01T00:00:00Z, so each one decodes to the instant it names; %d of the %d "+
			"instants in the sweep came back as the wrong instant, and the first is %s",
			mismatches, len(instants), first)
	}
}
