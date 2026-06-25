package server

import (
	"fmt"
	"testing"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

// browseTestServer builds a server whose Objects node has childCount children,
// and returns a ViewService plus the parent node id to browse.
func browseTestServer(t *testing.T, childCount int) (*ViewService, *ua.NodeID) {
	t.Helper()

	s := New()
	ns := NewNodeNameSpace(s, "test")
	parent := ns.Objects()
	for i := 0; i < childCount; i++ {
		child := ns.AddNewVariableNode(fmt.Sprintf("var%04d", i), int32(i))
		parent.AddRef(child, id.HasComponent, true)
	}
	return &ViewService{srv: s}, parent.ID()
}

func browseDesc(nodeID *ua.NodeID) *ua.BrowseDescription {
	return &ua.BrowseDescription{
		NodeID:          nodeID,
		BrowseDirection: ua.BrowseDirectionForward,
		ReferenceTypeID: ua.NewTwoByteNodeID(0),
		IncludeSubtypes: true,
		ResultMask:      uint32(ua.BrowseResultMaskAll),
	}
}

func callBrowse(t *testing.T, vs *ViewService, nodeID *ua.NodeID, maxRefs uint32) *ua.BrowseResult {
	t.Helper()
	req := &ua.BrowseRequest{
		RequestHeader:                 &ua.RequestHeader{},
		RequestedMaxReferencesPerNode: maxRefs,
		NodesToBrowse:                 []*ua.BrowseDescription{browseDesc(nodeID)},
	}
	resp, err := vs.Browse(nil, req, 0)
	require.NoError(t, err)
	br, ok := resp.(*ua.BrowseResponse)
	require.True(t, ok, "expected *ua.BrowseResponse, got %T", resp)
	require.Len(t, br.Results, 1)
	return br.Results[0]
}

func callBrowseNext(t *testing.T, vs *ViewService, cp []byte, release bool) *ua.BrowseResult {
	t.Helper()
	req := &ua.BrowseNextRequest{
		RequestHeader:             &ua.RequestHeader{},
		ReleaseContinuationPoints: release,
		ContinuationPoints:        [][]byte{cp},
	}
	resp, err := vs.BrowseNext(nil, req, 0)
	require.NoError(t, err)
	bn, ok := resp.(*ua.BrowseNextResponse)
	require.True(t, ok, "expected *ua.BrowseNextResponse, got %T", resp)
	require.Len(t, bn.Results, 1)
	return bn.Results[0]
}

// refKey identifies a reference well enough to detect duplicates across pages.
func refKey(r *ua.ReferenceDescription) string {
	return r.ReferenceTypeID.String() + "|" + r.NodeID.String()
}

// TestContinuationPointRoundTrip checks the stateless continuation point
// encode/decode is lossless for the fields BrowseNext relies on, and that a
// malformed point is rejected with BadContinuationPointInvalid.
func TestContinuationPointRoundTrip(t *testing.T) {
	vs := &ViewService{}

	bd := &ua.BrowseDescription{
		NodeID:          ua.NewStringNodeID(2, "the-parent"),
		BrowseDirection: ua.BrowseDirectionInverse,
		ReferenceTypeID: ua.NewNumericNodeID(0, id.HasComponent),
		IncludeSubtypes: true,
		NodeClassMask:   uint32(ua.NodeClassVariable),
		ResultMask:      uint32(ua.BrowseResultMaskAll),
	}

	cp, err := vs.encodeContinuationPoint(bd, 42, 100)
	require.NoError(t, err)

	gotBD, offset, maxRefs, err := vs.decodeContinuationPoint(cp)
	require.NoError(t, err)
	require.Equal(t, uint32(42), offset)
	require.Equal(t, uint32(100), maxRefs)
	require.True(t, gotBD.NodeID.Equal(bd.NodeID), "node id round-trip")
	require.Equal(t, bd.BrowseDirection, gotBD.BrowseDirection)
	require.True(t, gotBD.ReferenceTypeID.Equal(bd.ReferenceTypeID))
	require.Equal(t, bd.IncludeSubtypes, gotBD.IncludeSubtypes)
	require.Equal(t, bd.NodeClassMask, gotBD.NodeClassMask)
	require.Equal(t, bd.ResultMask, gotBD.ResultMask)

	for _, bad := range [][]byte{nil, {}, {1, 2, 3}, {1, 2, 3, 4, 5, 6, 7}} {
		_, _, _, err := vs.decodeContinuationPoint(bad)
		require.Equal(t, ua.StatusBadContinuationPointInvalid, err, "short cp %v", bad)
	}
}

// TestBrowseNoLimitReturnsNoContinuationPoint checks that when the client does
// not set RequestedMaxReferencesPerNode the server returns everything in one
// response with no continuation point (the zero-cost path).
func TestBrowseNoLimitReturnsNoContinuationPoint(t *testing.T) {
	vs, parentID := browseTestServer(t, 100)

	res := callBrowse(t, vs, parentID, 0)
	require.Equal(t, ua.StatusOK, res.StatusCode)
	require.Empty(t, res.ContinuationPoint, "no continuation point when unlimited")
	require.GreaterOrEqual(t, len(res.References), 100)
}

// TestBrowseWithContinuationCollectsAllRefsOnce drives a full Browse +
// BrowseNext paging loop and asserts it yields exactly the same set of
// references as a single unpaged Browse, each one exactly once.
func TestBrowseWithContinuationCollectsAllRefsOnce(t *testing.T) {
	const childCount = 250
	const pageSize = 10

	vs, parentID := browseTestServer(t, childCount)

	// ground truth: one unpaged browse
	want := map[string]int{}
	for _, r := range callBrowse(t, vs, parentID, 0).References {
		want[refKey(r)]++
	}
	require.GreaterOrEqual(t, len(want), childCount, "ground-truth browse should see all children")
	require.Greater(t, len(want), pageSize, "need more than one page for this test to be meaningful")

	// paged walk
	got := map[string]int{}
	res := callBrowse(t, vs, parentID, pageSize)
	require.Equal(t, ua.StatusOK, res.StatusCode)
	require.LessOrEqual(t, len(res.References), pageSize)
	for _, r := range res.References {
		got[refKey(r)]++
	}

	cp := res.ContinuationPoint
	rounds := 0
	for len(cp) > 0 {
		rounds++
		require.Less(t, rounds, childCount+10, "BrowseNext did not terminate")

		next := callBrowseNext(t, vs, cp, false)
		require.Equal(t, ua.StatusOK, next.StatusCode)
		require.LessOrEqual(t, len(next.References), pageSize)
		for _, r := range next.References {
			got[refKey(r)]++
		}
		cp = next.ContinuationPoint
	}

	require.Equal(t, want, got, "paged browse must yield every reference exactly once")
	for k, n := range got {
		require.Equal(t, 1, n, "reference %s returned %d times", k, n)
	}
}

// TestBrowseNextReleaseContinuationPoints checks that releasing a continuation
// point is acknowledged with no references and no further point (stateless, so
// it is a no-op acknowledgement).
func TestBrowseNextReleaseContinuationPoints(t *testing.T) {
	vs, parentID := browseTestServer(t, 50)

	first := callBrowse(t, vs, parentID, 10)
	require.NotEmpty(t, first.ContinuationPoint)

	res := callBrowseNext(t, vs, first.ContinuationPoint, true)
	require.Equal(t, ua.StatusOK, res.StatusCode)
	require.Empty(t, res.References)
	require.Empty(t, res.ContinuationPoint)
}

// TestBrowseNextInvalidContinuationPoint checks a malformed continuation point
// is reported per-result without failing the whole service call.
func TestBrowseNextInvalidContinuationPoint(t *testing.T) {
	vs, _ := browseTestServer(t, 10)

	res := callBrowseNext(t, vs, []byte{0xde, 0xad}, false)
	require.Equal(t, ua.StatusBadContinuationPointInvalid, res.StatusCode)
}

// TestBrowseNextRejectsZeroMaxRefs checks that a continuation point with maxRefs
// 0 is rejected rather than driving an infinite paging loop. The server never
// emits one, but a replayed or forged point would otherwise loop on empty pages.
func TestBrowseNextRejectsZeroMaxRefs(t *testing.T) {
	vs, parentID := browseTestServer(t, 50)

	// decode must reject it outright
	cp, err := vs.encodeContinuationPoint(browseDesc(parentID), 0, 0)
	require.NoError(t, err)
	_, _, _, err = vs.decodeContinuationPoint(cp)
	require.Equal(t, ua.StatusBadContinuationPointInvalid, err)

	// BrowseNext must surface it as a terminal error, not a loopable empty page + CP.
	res := callBrowseNext(t, vs, cp, false)
	require.Equal(t, ua.StatusBadContinuationPointInvalid, res.StatusCode)
	require.Empty(t, res.ContinuationPoint, "must not return a continuation point that would re-loop")
}
