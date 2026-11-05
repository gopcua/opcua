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
// Everything here goes through ua.Encode and ua.Decode, the path an
// application's service responses take, rather than through Buffer.ReadTime
// directly. The reported defect is only interesting if it reaches a caller.

package encoding

import (
	"fmt"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
)

func TestDateTimeDecodesEveryTickInTheValidRange(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. README.md says when to delete it. The
	// behaviour is closed by https://github.com/gopcua/opcua/pull/906.
	const knownBug = "a tick count inside the range §5.2.2.5 defines does not decode to the instant that clause names, outside roughly 1678-2262: ua/buffer.go's ReadTime does time.Unix(0, int64((ts-116444736000000000)*100)), so the unsigned subtraction underflows below 1970 and the int64 nanosecond multiply overflows above 2262"

	instants := validRangeInstants()
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
		if !got.V.Equal(want) {
			mismatches++
			if first == "" {
				first = fmt.Sprintf("tick count %s, which §5.2.2.5 defines as %s, "+
					"and it decoded as %s",
					ticks, want.Format(time.RFC3339Nano), got.V.UTC().Format(time.RFC3339Nano))
			}
		}
	}
	decodes := mismatches == 0

	switch {
	case decodes && knownBug != "":
		t.Errorf("this now passes (every one of the %d instants in the sweep decoded to "+
			"the instant its tick count names) - "+
			"delete the knownBug constant and the two arms that read it, in the commit that fixed it",
			len(instants))
	case decodes:
		t.Logf("every one of the %d instants in the sweep decoded to the instant its "+
			"tick count names", len(instants))
	case knownBug != "":
		t.Logf("known bug: %s; %d of the %d instants in the sweep came back as the wrong "+
			"instant, and the first is %s",
			knownBug, mismatches, len(instants), first)
	default:
		t.Errorf("§5.2.2.5 defines a tick count as a number of 100 nanosecond intervals since "+
			"1601-01-01T00:00:00Z, so each one decodes to the instant it names; %d of the %d "+
			"instants in the sweep came back as the wrong instant, and the first is %s",
			mismatches, len(instants), first)
	}
}
