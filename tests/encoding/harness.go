// Package encoding holds the gopcua encoding tests: one behaviour per file,
// with the clause it checks in the file's opening comment. This file holds
// the helpers they share.
//
// The shared file is helpers rather than a scripted server, because these
// rows drive ua.Encode and ua.Decode directly and need no server.
package encoding

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	"github.com/gopcua/opcua/ua"
)

const (
	// ticksPerSecond is the clause's unit: 100 nanosecond intervals.
	ticksPerSecond = 10000000

	// epochShiftSeconds is 1601-01-01T00:00:00Z expressed as a Unix time,
	// negated. The clause counts from 1601; Go counts from 1970.
	epochShiftSeconds = 11644473600

	// firstValidYear and lastValidYear bound the range §5.2.2.5 gives
	// literal dates for: it names 1601-01-01 12:00AM as the low end and
	// 9999-12-31 11:59:59PM as the high end.
	firstValidYear = 1601
	lastValidYear  = 9999
)

// wireDateTime is a struct with a single DateTime field. Service responses
// carry timestamps this way, and ua.Encode and ua.Decode reach the
// conversion through their reflection path, as they do in production.
type wireDateTime struct {
	V time.Time
}

// validRangeInstants returns the instants the sweep runs over: one per year
// across the range §5.2.2.5 gives dates for, plus the tick counts where an
// int64 nanosecond count overflows, which is where the conversion this file
// was written for went wrong before #906.
//
// The per-year instant carries a sub-second component that is a whole number
// of ticks and not a whole number of milliseconds, so a conversion that
// silently works in a coarser unit shows up as a mismatch rather than
// rounding away.
func validRangeInstants() []time.Time {
	var out []time.Time
	for y := firstValidYear; y <= lastValidYear; y++ {
		out = append(out, time.Date(y, time.June, 15, 13, 45, 30, 123456700, time.UTC))
	}
	// The first representable tick, where the subtraction of the 1601 epoch
	// underflows an unsigned conversion.
	out = append(out, time.Date(1601, time.January, 1, 0, 0, 0, 100, time.UTC))
	// 1601-01-01T00:00:00Z itself is tick 0, which the sweep leaves out;
	// the rows that sweep this range declare that exclusion, and its
	// reason, in their headers.
	//
	// Either side of each end of the window an int64 nanosecond count can
	// hold. Both edges are one tick wide, and a sweep that brackets only
	// one of them leaves the other to samples a year apart.
	out = append(out,
		time.Date(1677, time.September, 21, 0, 12, 43, 145224200, time.UTC),
		time.Date(1677, time.September, 21, 0, 12, 43, 145224100, time.UTC),
		time.Date(2262, time.April, 11, 23, 47, 16, 854775800, time.UTC),
		time.Date(2262, time.April, 11, 23, 47, 16, 854775900, time.UTC),
	)
	// The high end the clause names.
	out = append(out, time.Date(lastValidYear, time.December, 31, 23, 59, 59, 0, time.UTC))
	return out
}

// ticksFor returns the tick count §5.2.2.5 defines for an instant: the number
// of 100 nanosecond intervals since 1601-01-01T00:00:00Z. The arithmetic is
// in math/big, which is the clause's own algebra in a type that cannot
// overflow - the conversions under test run it forwards in one row and
// backwards in the other - so the oracle cannot fail the way they do.
func ticksFor(t time.Time) *big.Int {
	u := t.UTC()
	secs := new(big.Int).SetInt64(u.Unix())
	secs.Add(secs, big.NewInt(epochShiftSeconds))
	secs.Mul(secs, big.NewInt(ticksPerSecond))
	return secs.Add(secs, big.NewInt(int64(u.Nanosecond()/100)))
}

// wireBytes renders a tick count as the eight little-endian bytes §5.2.2.2
// specifies for a 64-bit integer.
func wireBytes(ticks *big.Int) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, ticks.Uint64())
	return b
}

// decodeVariantRecovering calls ua.Decode on a Variant packet and turns a
// panic into a value, the way encodeRecovering does for encoding.
// Variant.Decode reaches reflect.MakeSlice with a length taken straight off
// the wire, and a panic there stops every case after it in the test binary.
// The Variant it returns is the one the packet was decoded into, so the
// caller can read its state; only a test function turns any of this into
// a failure.
func decodeVariantRecovering(b []byte) (v *ua.Variant, err error, panicked string) {
	defer func() {
		if r := recover(); r != nil {
			panicked = fmt.Sprint(r)
		}
	}()
	v = new(ua.Variant)
	_, err = ua.Decode(b, v)
	return v, err, panicked
}

// encodeRecovering calls ua.Encode and turns a nil dereference into a value.
// A missing nil guard panics, and a panic in a test binary stops every case
// after it; the point of this suite is to say which types are affected, not
// just that one is.
func encodeRecovering(v interface{}) (b []byte, err error, panicked string) {
	defer func() {
		if r := recover(); r != nil {
			panicked = fmt.Sprint(r)
		}
	}()
	b, err = ua.Encode(v)
	return b, err, ""
}
