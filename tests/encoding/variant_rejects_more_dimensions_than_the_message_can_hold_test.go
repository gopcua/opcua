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
// message plainly cannot hold - which 10000000 is, asking for 40000000
// bytes of dimensions from a packet that holds none - not the largest
// value the field can carry. The Int32 maximum would make every run
// perform make([]int32, 2147483647), roughly 8 GiB, and
// .github/workflows/go.yml runs this suite on windows-latest, where
// Windows charges commit for an allocation instead of overcommitting
// lazily: the 8 GiB is charged for real, a charge failure is a
// "runtime: out of memory" fatal error that recover cannot catch, and it
// kills the test binary and fails the whole package, not just this row.
// That would turn the suite itself into the memory hazard it exists to
// catch, on a public repository, intermittently. A count of 10000000
// keeps the allocation near 38 MiB: still far outside anything a
// conforming decoder does with a nine-byte packet, and safe on any
// runner of the matrix.
//
// The neighbouring rows are
// variant_rejects_an_array_length_that_is_not_a_count_test.go, which is the
// array-length field rather than the dimension-count field, and
// variant_rejects_dimensions_inconsistent_with_the_array_length_test.go,
// which is the relation between the dimensions and the length. The three
// are neighbours of each other and of nothing else in this directory: the
// datetime rows read §5.2.2.5 and the three nil-value rows,
// reference_description_with_nil_fields_encodes_test.go among them, read
// §5.1.2 Table 1 - clauses about a different field of a different type.
// None of the six touches a Variant array field; where a nil row names a
// Variant at all it is a null scalar, not an array.
//
// The row runs against the shared helpers, with no scripted server in
// this package: the packet is decoded by
// decodeVariantRecovering, the same helper the array-length row decodes
// through; the ReadMemStats measurements around it are this row's own, not
// shared. See harness.go.
//
// This row asserts a resource property, not just an error. gopcua
// already returns StatusBadEncodingLimitsExceeded for the packet this row decodes -
// after allocating for the dimensions it is about to find missing - so a
// row that only asserted "decoding returns an error" would pass today
// and establish nothing: the error is real, the evidence is real, and
// the property it is offered as evidence of is not the one held. The
// allocation is the only thing here that distinguishes a conforming
// decoder from this one, so the allocation is what the row measures,
// with the runtime.ReadMemStats TotalAlloc delta around the call.
//
// The row asserts that decoding the packet is rejected and that the
// rejection cost at most dimsAllocationBound of allocation. It deliberately
// does not require a particular error, and it does not check the dimension
// values themselves: with zero bytes behind the count there are none to
// check, and the "greater than zero" clause is what the unpatched decoder
// fails on, one missing dimension too late.
//
// All three reference stacks were read, and all three would pass this row:
// each bounds the dimension count before allocating for it, though not all
// three bound it the same way - open62541 and UA-.NETStandard against the
// bytes actually present, Milo only against its maximum message size, which
// a smaller count with no bytes behind it would still pass. open62541 reads the dimension array through
// Array_decodeBinary, which checks "length / 128 <= remaining /
// type->memSize" before its ctxCalloc (ua_types_encoding_binary.c:551).
// UA-.NETStandard reads it through ReadInt32Array -> ReadArrayLength, which
// throws BadDecodingError when the length times the element size exceeds
// the remaining bytes (BinaryDecoder.cs:2376), before ReadFixedWidthArray
// makes its T[]. Milo's decodeDimensions calls checkArrayLength on the
// count before its "new int[length]" (OpcUaBinaryDecoder.java:502, :504),
// and checkArrayLength throws Bad_EncodingLimitsExceeded when the count
// exceeds the maximum message size (:1406), which 10000000 does against
// Milo's default of two megabytes (EncodingLimits.java:23). Milo's
// dimension-consistency check (:430, dimensionsLength != length) is never
// reached on this packet: the count fails the checkArrayLength bound before
// any dimension is read, so the packet exits through Milo's dimension-count
// path, not its consistency path.

package encoding

import (
	"fmt"
	"runtime"
	"testing"
)

// dimsAllocationBound is the allocation a conforming decoder may make while
// rejecting this packet. The two measured outcomes sit well on either side
// of it: unpatched, decoding the packet below allocates about 40020000
// bytes (~38 MiB) before it fails - the exact figure moves a little from
// run to run, because TotalAlloc counts the runtime's own allocations
// too; with pull request 936's dimension guard
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
	// gopcua did not do this until https://github.com/gopcua/opcua/pull/936
	// merged; it passes since.

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

	// mustRejectBeforeAllocation is the reason the default arm below gives
	// for its failure.
	const mustRejectBeforeAllocation = "§5.2.2.16 encodes each dimension as an Int32, so a count " +
		"of 10000000 with no bytes behind it names dimensions the message cannot " +
		"hold, and they must be rejected before the allocation"

	switch {
	case rejected && withinBound:
		t.Logf("a dimension count larger than the remaining bytes can hold is rejected "+
			"before the allocation (%d bytes against a bound of %d)",
			alloc, dimsAllocationBound)
	default:
		t.Errorf("%s:%s", mustRejectBeforeAllocation, detail)
	}
}
