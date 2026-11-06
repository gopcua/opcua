// A tick count outside the range §5.2.2.5's definition covers decodes to
// the latest or earliest time that can be represented, not to whatever
// date the conversion arithmetic yields.
//
// OPC 10000-6 §5.2.2.5: "A date/time is decoded as the latest time that
// can be represented on the platform if either: the encoded value is the
// maximum value for an Int64 [or] the encoded value represents a time
// later than the latest time that can be represented", and decodes as the
// earliest time that can be represented when the encoded value represents
// a time earlier than that.
//
// The neighbouring rows are
// datetime_decodes_every_tick_in_the_valid_range_test.go and
// datetime_encodes_every_instant_in_the_valid_range_test.go, the clause's
// definition read both ways for tick counts inside the range it covers;
// this row is the clause's rules for tick counts outside that range.
//
// The row runs against the shared helpers, with no scripted server in
// this package: each probe's tick count is rendered to its eight wire
// bytes by wireBytes and decoded into a wireDateTime, so ua.Decode takes
// the same reflection path a service response does. See harness.go.
//
// "The platform" is read as the clause's own range, 1601-01-01 to
// 9999-12-31, and not as Go's time.Time range, which runs years wider in
// both directions. Under Go's reading neither rule could ever fire - every
// tick count an eight-byte field can carry names a time Go can represent -
// and the clause's sentence would be dead letters in a Go stack. The
// clause names the ends of its own range, 1601-01-01 12:00AM and 9999-12-31
// 11:59:59PM, the bounds harness.go holds as firstValidYear and
// lastValidYear. Read that way both rules fire, and the row names the
// instants they decode to exactly:
//
//	earliest  1601-01-01T00:00:00Z - tick 0, the clause's own low end
//	latest    9999-12-31T23:59:59Z - the high end the clause names
//
// The naming is evidence, not invention: the tick counts those instants
// correspond to are constants in the stacks that coerce. ticksFor(latest)
// is 2650467743990000000, the count UA-.NETStandard clamps to
// (DateTimeUtc.cs's kMaxValue, line 685) and the count open62541 encodes
// up to (UA_DATETIME_ENCODE_MAX = 265046774399 * UA_DATETIME_SEC,
// ua_types_encoding_binary.c line 288, whose comment names it "9999-12-31
// 23:59:59 UTC"); ticksFor(earliest) is 0, the count UA-.NETStandard
// clamps negative values to. The test checks both equalities as part of
// its setup, so the named instants cannot drift from the constants they
// cite. The sub-second part of the latest instant is not left open: the
// clause names its high end as an instant to the second, so the 100ns
// ticks between 23:59:59.0000000 and 23:59:59.9999999 on 9999-12-31 are
// themselves "later than the latest time that can be represented" and
// decode as that named instant.
//
// The probes are the two groups the clause's rules define. The latest
// group is the Int64 maximum - the clause's own sentinel, "the maximum
// value for an Int64" - one tick below it (one tick under the Int64
// maximum, which under the bounded reading is "later than the latest time
// that can be represented"), and 2650467743990000001, one tick past the
// latest instant the clause names. The field's top and the clause's
// boundary are different ticks: a fix keyed on the top two values alone
// would satisfy those probes and still leave 2650467743990000001 decoding
// as a garbage date, so the group carries it too. It also makes the
// sub-second ruling above literal: 9999-12-31T23:59:59.0000001Z is later
// than the named latest, and decodes as that named instant. The earliest
// group is MaxUint64 (ff ff ff ff ff ff ff ff, which read as the signed
// integer §5.2.2.2 and §5.2.2.5 specify is -1, a time before 1601), the
// Int64 minimum, and -2^62 (00 00 00 00 00 00 00 c0 little-endian), a
// middle negative - a time before 1601 that is not at the negative
// domain's edge, so a fix keyed on the extremes alone misses it. Each
// group must collapse to one instant, and that instant must be the one
// named above; the two named instants differ, so a fix that clamps both
// groups to a single date fails.
//
// What the row deliberately does not assert: nothing about the encode
// direction, and nothing about tick counts inside the range - those are
// the two neighbouring rows' behaviours - and nothing about the clause's
// "the encoded value is 0" condition, which is the zero time gopcua
// reserves tick 0 for. A fix for this row must leave tick 0 decoding as
// it does - to time.Time{}, gopcua's own null convention - because no row
// in this package probes tick 0, and nothing here would catch a fix that
// changed it. No row in this package probes the two encode-side
// saturation rules at all. The earliest group's probes do not carry tick
// 0, so gopcua's zero-time convention does not reach them: the clause
// decodes them as the earliest time the range admits, not as a null
// marker.
//
// All three reference stacks were read for this row. UA-.NETStandard
// implements the rule and passes it: BinaryDecoder.cs's ReadDateTime (lines 680-683)
// reads the field as the signed Int64 the clause specifies and constructs
// DateTimeUtc(SafeReadInt64()), whose constructor (DateTimeUtc.cs lines
// 111-114) runs EnsureBounded (lines 630-641): a negative count becomes 0
// and one above kMaxValue (line 685) becomes kMaxValue. That is to say
// UA-.NETStandard decodes both groups to exactly the instants this row
// names. open62541
// and Eclipse Milo do not coerce on decode and would fail the row:
// open62541 dispatches DateTime to UInt64_decodeBinary
// (ua_types_encoding_binary.c line 2032), whose body (lines 303-314) copies
// the eight bytes and bounds-checks only - UA_DateTime is a signed int64
// (types.h line 281), so MaxUint64 arrives as -1, but neither group
// collapses; its clamp sits on the encode side (lines 294-301). Milo's
// decodeDateTime (OpcUaBinaryDecoder.java lines 189-191) constructs
// DateTime(buffer.readLongLE()), a record holding the raw long
// (DateTime.java line 19), and neither group collapses either. Under
// ../README.md's sixth question the row stands: UA-.NETStandard shows a
// known-good stack earns the outcome the clause states, so the row asserts
// the outcome rather than how an implementation works, and it is not the
// case that every library leaves the rule to the application. The two
// failing stacks fail it by not implementing the rule - the same failure
// this row exists to measure in gopcua. The row still fails an
// implementation that fakes the outcome instead of earning it: a fix that
// reads the field unsigned and clamps to the range's top, and a fix that
// clamps both groups to one instant, each leave a group collapsed to an
// instant that is not the one named, which is the hard-failing arm below.
//
// gopcua implements none of the rules it probes: it feeds whatever tick
// count arrives to its conversion arithmetic and returns the date that
// falls out, so the latest group comes back today as three wrapped-overflow
// dates (two in 2185 and one in 1816), and the earliest group as two
// wrapped-overflow dates in 2185 - none of them an end of the clause's
// range. Under PR #906's in-range fix the probes decode exactly but still
// do not collapse: the latest group spreads over 9999-12-31T23:59:59.0000001Z
// and two dates in 30828, the earliest group over three dates in 60056,
// 30828 and 45442. PR #906
// fixes the in-range conversion and adds no saturation, so the gap stays
// open through it, though the probes' decoded values move. The row is here
// so the gap is measured on every run rather than remembered in a
// document, and so that whoever closes it finds out immediately.
//
// Arm keying: the green arm keys on a presence, not an absence - a group
// whose members decode to different instants. That presence is what
// today's library shows and what a fixed library does not: the fix
// collapses both groups onto the named instants, so the deletion arm
// fires. A library that cannot be measured at all - one that returns the
// same instant for every tick count, the zero time say - does not show the
// presence either: it collapses both groups and lands in the wrong-fix
// arm, red, not green. A decode error is not a decline: it is that
// probe's outcome, it goes in the detail line, it counts as its group not
// landing on the named instant, and it routes to a red arm of its own -
// the decoder rejected what the clause saturates. Today's library does
// not error, so an error is a change, and a change that is not the fix is
// a wrong fix. The ticksFor anchors above still decline with a Fatalf:
// they check the oracle, which ua.Decode cannot reach.

package encoding

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
)

// collapsed reports whether every probe in a group decoded to one instant.
// A group whose probes all errored has no instants at all: it has not
// collapsed to anything, and must not be read as vacuously collapsed, so
// an empty group reports false instead of indexing an empty slice.
func collapsed(instants []time.Time) bool {
	if len(instants) == 0 {
		return false
	}
	for _, g := range instants[1:] {
		if !g.Equal(instants[0]) {
			return false
		}
	}
	return true
}

func TestDateTimeOutOfRangeSaturates(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. ../README.md says when to delete it.
	const knownBug = "a tick count outside the range §5.2.2.5 defines decodes to whatever date " +
		"the conversion arithmetic yields instead of saturating, so neither probe group " +
		"collapses to the instant the clause names for it"

	// The instants the clause's rules name, under the bounded reading of
	// "the platform" stated in the header.
	earliest := time.Date(1601, time.January, 1, 0, 0, 0, 0, time.UTC)
	latest := time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC)

	// The named instants are anchored outside gopcua, in the constants the
	// header cites: UA-.NETStandard's kMaxValue and open62541's
	// UA_DATETIME_ENCODE_MAX for the latest, tick 0 for the earliest.
	uanetKMaxValue := new(big.Int).SetUint64(2650467743990000000)
	if got := ticksFor(latest); got.Cmp(uanetKMaxValue) != 0 {
		t.Fatalf("could not set the test up: 9999-12-31T23:59:59Z is %s ticks, "+
			"not the 2650467743990000000 UA-.NETStandard clamps to and open62541 encodes up to",
			got)
	}
	if got := ticksFor(earliest); got.Sign() != 0 {
		t.Fatalf("could not set the test up: 1601-01-01T00:00:00Z is %s ticks, not 0", got)
	}

	probes := []struct {
		name  string
		group string
		ticks *big.Int
	}{
		{"the Int64 maximum", "latest", new(big.Int).SetUint64(1<<63 - 1)},
		{"one tick below the Int64 maximum", "latest", new(big.Int).SetUint64(1<<63 - 2)},
		// One tick past uanetKMaxValue, the latest instant the clause
		// names: the field's top is not the clause's boundary, and a fix
		// keyed on the top two values alone would leave this tick decoding
		// as a garbage date.
		{"one tick past the clause's latest", "latest", new(big.Int).Add(uanetKMaxValue, big.NewInt(1))},
		{"MaxUint64", "earliest", new(big.Int).SetUint64(^uint64(0))},
		{"the Int64 minimum", "earliest", new(big.Int).SetUint64(1 << 63)},
		// 0xC000000000000000 read as the signed integer §5.2.2.2 specifies
		// is -2^62: a time before 1601 that is not at the negative domain's
		// edge. It is written as the literal pattern because big.Int's
		// Uint64 does not render a negative value as its two's complement.
		{"-2^62, a middle negative", "earliest", new(big.Int).SetUint64(0xC000000000000000)},
	}
	if len(probes) == 0 {
		t.Fatal("the case table is empty - this row would report a pass " +
			"having checked nothing")
	}

	var latestGot, earliestGot []time.Time
	var latestErrs, earliestErrs []string
	var detail string
	for _, p := range probes {
		var into wireDateTime
		_, err := ua.Decode(wireBytes(p.ticks), &into)
		if err != nil {
			// A decode error is this probe's outcome, not a setup
			// failure: the clause saturates, it does not reject, so a
			// library that errors has changed behaviour, and the change
			// is scored below, not declined.
			if p.group == "latest" {
				latestErrs = append(latestErrs, p.name)
			} else {
				earliestErrs = append(earliestErrs, p.name)
			}
			detail += fmt.Sprintf("\n    %-34s decode error: %v", p.name, err)
			continue
		}
		got := into.V.UTC()
		if p.group == "latest" {
			latestGot = append(latestGot, got)
		} else {
			earliestGot = append(earliestGot, got)
		}
		detail += fmt.Sprintf("\n    %-34s decoded as %s", p.name, got.Format(time.RFC3339Nano))
	}

	latestCollapsed := collapsed(latestGot)
	earliestCollapsed := collapsed(earliestGot)
	// The clause's rules name an instant, not just a collapse, and the
	// header names both, so the collapse is checked against the named
	// instants. A group with a decode error has not landed on the instant
	// the clause names for it, whatever its other probes did - the error
	// is part of the group's outcome. The two groups' instants must
	// differ: they name opposite ends of the range, and the named
	// instants differ, so a fix that clamps everything to one date fails
	// here.
	atLatest := len(latestErrs) == 0 && latestCollapsed && latestGot[0].Equal(latest)
	atEarliest := len(earliestErrs) == 0 && earliestCollapsed && earliestGot[0].Equal(earliest)
	// The known bug's signature: a group whose members decode to different
	// instants. The header's "Arm keying" paragraph says why the green arm
	// keys on it.
	unresolved := !latestCollapsed || !earliestCollapsed

	switch {
	case atLatest && atEarliest && knownBug != "":
		t.Errorf("this now passes (the earliest group decodes as %s and the latest group as %s) - "+
			"delete the knownBug constant, every arm that reads it, and "+
			"anything only those arms used, in the commit that fixed it",
			earliest.Format(time.RFC3339Nano), latest.Format(time.RFC3339Nano))
	case atLatest && atEarliest:
		t.Logf("every out-of-range tick count decodes to the instant the clause names: "+
			"the earliest group as %s, the latest group as %s",
			earliest.Format(time.RFC3339Nano), latest.Format(time.RFC3339Nano))
	case len(latestErrs) > 0 || len(earliestErrs) > 0:
		rejected := append(append([]string{}, latestErrs...), earliestErrs...)
		t.Errorf("§5.2.2.5 saturates out-of-range tick counts: it names no error for them, and "+
			"today's library does not return one. The decoder returned an error for %s - a decode "+
			"error is a change, and a change that is not the fix: these tick counts decode to the "+
			"named instants, they are not declined:%s",
			strings.Join(rejected, ", "), detail)
	case latestCollapsed && earliestCollapsed:
		t.Errorf("both probe groups collapse, but to instants other than the ones §5.2.2.5 names - "+
			"the earliest time its range admits, 1601-01-01T00:00:00Z, and the latest, "+
			"9999-12-31T23:59:59Z. The collapse the fix should bring is present, but the instants "+
			"are wrong, whichever state this is: a fix that clamps to the wrong ends of the range, "+
			"or a library that returns one instant for every tick count and that instant is one of "+
			"these. Either way it is not the saturation the clause states:%s", detail)
	case unresolved && knownBug != "":
		t.Logf("known bug: %s; §5.2.2.5 decodes the latest group as 9999-12-31T23:59:59Z and "+
			"the earliest group as 1601-01-01T00:00:00Z:%s", knownBug, detail)
	case unresolved:
		t.Errorf("§5.2.2.5 decodes the latest group as 9999-12-31T23:59:59Z and the earliest "+
			"group as 1601-01-01T00:00:00Z, and at least one group did not even collapse to "+
			"one instant:%s", detail)
	}
}
