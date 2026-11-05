// The clause's remaining rules cover what happens outside that range and
// are a separate behaviour from the definition, tested in
// datetime_decodes_every_tick_in_the_valid_range_test.go and
// datetime_encodes_every_instant_in_the_valid_range_test.go.
//
// The row runs against the shared helpers, with no scripted server in
// this package: each probe's tick count is rendered to its eight wire
// bytes by wireBytes and decoded into a wireDateTime, so ua.Decode takes
// the same reflection path a service response does. See harness.go.
//
// TestDateTimeOutOfRangeSaturates checks the four rules §5.2.2.5 states for
// values outside the range the definition covers. gopcua implements none of
// them: it feeds whatever tick count arrives to its conversion arithmetic
// and returns the date that falls out, so the Int64 maximum - the clause's
// own "at or after 9999" sentinel - comes back today as a wrapped-overflow
// date in 2185, and once PR #906's in-range fix lands, as the literal date
// in 30828 its tick count names; neither is the latest time the platform
// can represent.
//
// This is the gap PR #906 leaves open and does not claim to close. It is
// not a regression: the missing rule stays missing before and after that
// change, though the probes' decoded values move with the conversion it
// fixes. The test is here so the gap is measured on every run rather than
// remembered in a document, and so that whoever closes it finds out
// immediately.
//
// What the row deliberately does not assert: which instant the probes
// saturate to. Go's latest representable time has no named constant, so
// the row checks only that the clause's saturating tick counts all decode
// to one instant, not that the instant is the platform's latest, and a
// fix that saturates them to one wrong instant passes as though the
// behaviour were fixed.
//
// No reference stack was read for this row; the clause's rules for values
// outside the range and the measured behaviour are the whole of the
// evidence.

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
	// in place of a failure. ../README.md says when to delete it.
	const knownBug = "a tick count outside the range §5.2.2.5 defines decodes to whatever date the conversion arithmetic yields instead of saturating to the platform's earliest or latest time, so the Int64 maximum - the clause's own sentinel for \"at or after 9999-12-31\" - comes back as a wrapped-overflow date in 2185 rather than as the latest time the platform can represent"

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
		// the platform's earliest. gopcua reads it unsigned, and today its
		// arithmetic wraps that to the same date as the Int64 maximum; once
		// PR #906's conversion lands it comes back as the literal date in
		// the year 60056. Either reading makes a literal date wrong.
		{"MaxUint64", new(big.Int).SetUint64(^uint64(0))},
	}
	if len(probes) == 0 {
		t.Fatal("the case table is empty - this row would report a pass " +
			"having checked nothing")
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

	// The green arm below keys on an absence - the probes not all decoding
	// to one instant - and this file says so, because the behaviour allows
	// no presence to key on instead. The two in-range datetime rows have
	// one: most of their sweep decodes rightly today, so "every wrong value
	// is exactly the value the known conversion predicts" separates the
	// known bug from a fix that landed and is wrong. Here all three probes
	// are outside any window where part of the behaviour is right, and the
	// wrong values are all the conversion produces: a wrong fix - reading
	// the tick count signed and clamping negatives to 1601-01-01, say -
	// moves the probes without saturating them, and this row cannot see
	// the move. Keying the arm on today's values would not do either: PR
	// #906's in-range fix moves them too, and the correct fix for this row
	// sends the Int64-maximum probe to the platform's latest representable
	// time, which the comment above records this row cannot name. No set
	// of predicted values holds across every state this row must stay
	// green in, so the arm stays keyed on the absence, and a wrong fix
	// stays green with it.
	switch {
	case saturated && knownBug != "":
		t.Errorf("this now passes (every out-of-range tick count decodes to %s) - "+
			"delete the knownBug constant, every arm that reads it, and "+
			"anything only those arms used, in the commit that fixed it",
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
