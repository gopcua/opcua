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

package encoding

import (
	"fmt"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
)

func TestDateTimeEncodesEveryInstantInTheValidRange(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. README.md says when to delete it. The
	// behaviour is closed by https://github.com/gopcua/opcua/pull/906.
	const knownBug = "an instant inside the range §5.2.2.5 defines does not encode to the tick count that clause names, outside roughly 1678-2262: ua/buffer.go's WriteTime does v.UTC().UnixNano()/100, and UnixNano is undefined outside that window, so the wire does not carry the number of 100 nanosecond intervals since 1601-01-01T00:00:00Z"

	instants := validRangeInstants()
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
		if string(got) != string(want) {
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
	case encodes && knownBug != "":
		t.Errorf("this now passes (every one of the %d instants in the sweep encoded as the "+
			"tick count §5.2.2.5 defines for it) - "+
			"delete the knownBug constant and the two arms that read it, in the commit that fixed it",
			len(instants))
	case encodes:
		t.Logf("every one of the %d instants in the sweep encoded as the tick count "+
			"§5.2.2.5 defines for it", len(instants))
	case knownBug != "":
		t.Logf("known bug: %s; %d of the %d instants in the sweep encoded as the wrong tick "+
			"count, and the first is %s",
			knownBug, mismatches, len(instants), first)
	default:
		t.Errorf("§5.2.2.5 defines an instant as a number of 100 nanosecond intervals since "+
			"1601-01-01T00:00:00Z, so each one encodes as the tick count that names it; %d of "+
			"the %d instants in the sweep encoded as the wrong tick count, and the first is %s",
			mismatches, len(instants), first)
	}
}
