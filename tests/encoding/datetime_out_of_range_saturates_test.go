// The clause's remaining rules cover what happens outside that range and
// are a separate behaviour from the definition, tested in
// datetime_decodes_every_tick_in_the_valid_range_test.go and
// datetime_encodes_every_instant_in_the_valid_range_test.go.
//
// TestDateTimeOutOfRangeSaturates checks the four rules §5.2.2.5 states for
// values outside the range the definition covers. gopcua implements none of
// them: it computes a literal date from whatever tick count arrives, so the
// Int64 maximum - the clause's own "at or after 9999" sentinel - decodes to
// a date in the year 30828 rather than to the latest time the platform can
// represent.
//
// This is the gap PR #906 leaves open and does not claim to close. It is not
// a regression: the behaviour is the same before and after that change. The
// test is here so the gap is measured on every run rather than remembered in
// a document, and so that whoever closes it finds out immediately.

package encoding

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
)

func TestDateTimeOutOfRangeSaturates(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. README.md says when to delete it.
	const knownBug = "a tick count outside the range §5.2.2.5 defines decodes to a literal date instead of saturating to the platform's earliest or latest time, so the Int64 maximum - the clause's own sentinel for \"at or after 9999-12-31\" - comes back as a date in the year 30828"

	// §5.2.2.5 rule 10/11: "A date/time is decoded as the latest time that
	// can be represented on the platform if either: the encoded value is the
	// maximum value for an Int64 [or] the encoded value represents a time
	// later than the latest time that can be represented".
	//
	// Go's latest representable time is not a named constant, so the check
	// is the weaker one the clause still forces: all three tick counts below
	// are "the maximum value for an Int64" or "later than the latest time
	// that can be represented", so the clause maps all three to one instant.
	// gopcua maps them to three different dates, which no implementation of
	// the rule can do. The check needs no name for Go's upper bound.
	probes := []struct {
		name  string
		ticks *big.Int
	}{
		{"the Int64 maximum", new(big.Int).SetUint64(1<<63 - 1)},
		{"one tick below the Int64 maximum", new(big.Int).SetUint64(1<<63 - 2)},
		// ff ff ff ff ff ff ff ff. Read as the signed integer §5.2.2.5
		// specifies this is -1, a time before 1601, which rule 7/9 sends to
		// the platform's earliest. gopcua reads it unsigned and lands in the
		// year 60056. Either reading makes a literal date wrong.
		{"MaxUint64", new(big.Int).SetUint64(^uint64(0))},
	}

	got := make([]time.Time, len(probes))
	for i, p := range probes {
		var into wireDateTime
		if _, err := ua.Decode(wireBytes(p.ticks), &into); err != nil {
			t.Fatalf("could not set the test up: decoding %s failed: %v", p.name, err)
		}
		got[i] = into.V.UTC()
	}

	saturated := true
	for _, g := range got[1:] {
		if !g.Equal(got[0]) {
			saturated = false
		}
	}

	var detail string
	for i, p := range probes {
		detail += fmt.Sprintf("\n    %-34s decoded as %s", p.name, got[i].Format(time.RFC3339Nano))
	}

	switch {
	case saturated && knownBug != "":
		t.Errorf("this now passes (every out-of-range tick count decodes to %s) - "+
			"delete the knownBug constant and the two arms that read it, in the commit that fixed it",
			got[0].Format(time.RFC3339Nano))
	case saturated:
		t.Logf("every out-of-range tick count decodes to %s", got[0].Format(time.RFC3339Nano))
	case knownBug != "":
		t.Logf("known bug: %s; §5.2.2.5 maps all of these to one instant, the latest time the "+
			"platform can represent:%s", knownBug, detail)
	default:
		t.Errorf("§5.2.2.5 decodes the Int64 maximum and any later-than-representable tick count "+
			"as the latest time the platform can represent, and they differ:%s", detail)
	}
}
