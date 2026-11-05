// A dimension count larger than the remaining bytes can hold is rejected
// before the allocation, not after it.
//
// OPC 10000-6 §5.2.2.16 Variant: "The length of each dimension encoded as a
// sequence of Int32 values" and "All dimensions shall be specified and
// shall be greater than zero." The count names how many Int32 dimension
// values follow, each at least four bytes on the wire; a count of
// 10000000 with no bytes behind it names a message the nine-byte packet
// cannot hold - it asks for 40000000 bytes of dimensions and has none -
// and not one of the dimensions it asks for can be read.
//
// The count is 10000000 and not the Int32 maximum of 2147483647, and the
// reason is the suite, not the clause. The property this row scores is
// that the decoder allocates in proportion to a wire value it has not
// checked against the bytes present, and that property needs a count the
// message plainly cannot hold - which 10000000 is, by more than four
// orders of magnitude - not the largest value the field can carry. The
// Int32 maximum would make every run perform make([]int32, 2147483647),
// roughly 8 GiB, and .github/workflows/go.yml runs this suite on a matrix
// whose Windows half enforces commit charge instead of overcommitting
// lazily: the 8 GiB is charged for real, a charge failure is a
// "runtime: out of memory" fatal error that recover cannot catch, and it
// kills the test binary and fails the whole package, not just this row.
// That would turn the suite itself into the memory hazard it exists to
// catch, on a public repository, intermittently. A count of 10000000
// keeps the allocation near 38 MiB: still far outside anything a
// conforming decoder does with a nine-byte packet, and safe on any
// runner of the matrix.
//
// This row asserts a resource property, not just an error. gopcua
// already returns StatusBadEncodingLimitsExceeded for the packet below -
// after allocating for the dimensions it is about to find missing - so a
// row that only asserted "decoding returns an error" would pass today
// and establish nothing: the error is real, the evidence is real, and
// the property it is offered as evidence of is not the one held. The
// allocation is the only thing here that distinguishes a conforming
// decoder from this one, so the allocation is what the row measures,
// with the runtime.ReadMemStats TotalAlloc delta around the call.
//
// The neighbouring rows are
// variant_rejects_an_array_length_that_is_not_a_count_test.go, which is the
// array-length field rather than the dimension-count field, and
// variant_rejects_dimensions_inconsistent_with_the_array_length_test.go,
// which is the relation between the dimensions and the length. The three
// are neighbours of each other and of nothing else in this directory: the
// datetime and nil-builtin rows read §5.2.2.5 and §5.1.2 Table 1, clauses
// about a different field of a different type, and none of them touches a
// Variant array field.
//
// The row asserts that decoding the packet is rejected and that the
// rejection cost at most dimsAllocationBound of allocation. It deliberately
// does not require a particular error, and it does not check the dimension
// values themselves: with zero bytes behind the count there are none to
// check, and the "greater than zero" clause is what the unpatched decoder
// fails on, one missing dimension too late. While the knownBug constant
// stands, every run of this row performs the very allocation it protests -
// the make succeeds, one element is written, and the decode stops - and
// the knownBug line carries the measured size.
//
// No reference stack was read for this row; the two clauses and the
// measured behaviour are the whole of the evidence.

package encoding

import (
	"fmt"
	"runtime"
	"testing"
)

// dimsAllocationBound is the allocation a conforming decoder may make while
// rejecting this packet. The two measured outcomes sit well on either side
// of it: unpatched, decoding the packet below allocates 40019568 bytes
// (~38 MiB) before it fails; with pull request 936's dimension guard
// applied it allocates 384 bytes. A bound of 1 MiB therefore holds ~38x of
// slack against the bug and ~2700x against the fixed decoder, and
// TotalAlloc - cumulative, monotonic under garbage collection, counted
// across the whole runtime - cannot move that far between two ReadMemStats
// calls a decode apart: the only allocations in the window are the
// decode's own plus a few kilobytes of background runtime noise, on a
// loaded machine as on an idle one. Tightening the bound below a megabyte
// buys the row no discrimination it does not already have and invites
// flakiness; leave it where it is.
const dimsAllocationBound = 1 << 20 // bytes

func TestVariantRejectsMoreDimensionsThanTheMessageCanHold(t *testing.T) {
	// knownBug records that gopcua does not do this yet; the text is logged
	// in place of a failure. README.md says when to delete it. The
	// behaviour is closed by https://github.com/gopcua/opcua/pull/893 and
	// https://github.com/gopcua/opcua/pull/936.
	const knownBug = "a dimension count larger than the remaining bytes can hold is rejected only after the allocation it protests: make([]int32, arrayDimensionsLength) with the count taken straight off the wire, so a nine-byte message allocates roughly 38 MiB and only then fails on the first missing dimension"

	// mask 0xc1 is Boolean | array values | array dimensions; the array
	// length is 0, so no element bytes; the dimension count is 10000000,
	// and no dimension bytes follow it.
	packet := []byte{0xc1, 0x00, 0x00, 0x00, 0x00, 0x80, 0x96, 0x98, 0x00}

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, err, panicked := decodeVariantRecovering(packet)
	runtime.ReadMemStats(&after)
	alloc := after.TotalAlloc - before.TotalAlloc

	rejected := err != nil && panicked == ""
	withinBound := alloc <= dimsAllocationBound
	detail := fmt.Sprintf("\n    decoding % x returned err %v, panic %q, after "+
		"allocating %d bytes where the bound is %d",
		packet, err, panicked, alloc, dimsAllocationBound)

	// The presence the knownBug arm keys on is the over-bound allocation
	// itself, per the rule in tests/reconnect/README.md: allocating beyond
	// dimsAllocationBound is what says the decoder reached this stimulus and
	// did the wrong thing, so the arm below logs green only while the decode
	// still makes that allocation. Keying it on the absence of the rejection
	// instead would log a decoder that stopped allocating but stopped
	// rejecting too - a fix that landed wrong - green beside the unfixed
	// decoder, and the arm that asks for the constant to be deleted could
	// never fire on a wrong fix.
	//
	// The arms, in the order the rule asks for: the presence with the
	// constant standing is the green known bug; the presence gone but the
	// outcome still wrong is a fix that landed wrong, and red; the outcome
	// right with the constant standing asks for the constant to be deleted,
	// and is red; the outcome right with the constant gone is the row's
	// ordinary pass. What fits none of those - the over-bound allocation with
	// the constant deleted - falls to default, so deleting the constant
	// without the fix is red rather than silent.
	switch {
	case !withinBound && knownBug != "":
		t.Logf("known bug: %s; §5.2.2.16 encodes each dimension as an Int32, so a count "+
			"of 10000000 with no bytes behind it names dimensions the message cannot "+
			"hold, and they must be rejected before the allocation:%s", knownBug, detail)
	case withinBound && !rejected:
		t.Errorf("the over-bound allocation is gone - %d bytes against a bound of %d - "+
			"so a change has landed on this decode, but the outcome is still wrong: "+
			"§5.2.2.16 encodes each dimension as an Int32, so a count of 10000000 with "+
			"no bytes behind it names dimensions the message cannot hold, and the "+
			"decoder must reject them before the allocation; the change that bounded "+
			"the allocation did not make the decoder reject:%s",
			alloc, dimsAllocationBound, detail)
	case rejected && withinBound && knownBug != "":
		t.Errorf("this now passes (a dimension count larger than the remaining bytes "+
			"can hold is rejected before the allocation, %d bytes against a bound of %d) - "+
			"delete the knownBug constant and the two arms that read it, in the commit that fixed it",
			alloc, dimsAllocationBound)
	case rejected && withinBound:
		t.Logf("a dimension count larger than the remaining bytes can hold is rejected "+
			"before the allocation (%d bytes against a bound of %d)",
			alloc, dimsAllocationBound)
	default:
		t.Errorf("§5.2.2.16 encodes each dimension as an Int32, so a count of 10000000 "+
			"with no bytes behind it names dimensions the message cannot hold, and the "+
			"decoder must reject them before the allocation:%s", detail)
	}
}
