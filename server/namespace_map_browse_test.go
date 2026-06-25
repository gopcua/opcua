package server

import (
	"fmt"
	"testing"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

// mapBrowseTestServer builds a server with a MapNamespace of tagCount tags and
// returns a ViewService plus the Objects-folder node id to browse.
func mapBrowseTestServer(t *testing.T, tagCount int) (*ViewService, *ua.NodeID) {
	t.Helper()

	s := New()
	ns := NewMapNamespace(s, "maptest")
	for i := 0; i < tagCount; i++ {
		ns.Data[fmt.Sprintf("tag%04d", i)] = int32(i)
	}
	objects := ua.NewNumericNodeID(ns.ID(), id.ObjectsFolder)
	return &ViewService{srv: s}, objects
}

// TestMapNamespaceBrowseContinuationCollectsAllRefsOnce drives a full Browse +
// BrowseNext paging loop over a MapNamespace and asserts it yields the same set
// of references as a single unpaged Browse, each exactly once. Regression test
// for MapNamespace.Browse returning a non-deterministic order across pages.
func TestMapNamespaceBrowseContinuationCollectsAllRefsOnce(t *testing.T) {
	const tagCount = 200
	const pageSize = 10

	vs, parentID := mapBrowseTestServer(t, tagCount)

	// ground truth: one unpaged browse
	want := map[string]int{}
	for _, r := range callBrowse(t, vs, parentID, 0).References {
		want[refKey(r)]++
	}
	require.Len(t, want, tagCount, "ground-truth browse should see every tag exactly once")
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
		require.Less(t, rounds, tagCount+10, "BrowseNext did not terminate")

		next := callBrowseNext(t, vs, cp, false)
		require.Equal(t, ua.StatusOK, next.StatusCode)
		require.LessOrEqual(t, len(next.References), pageSize)
		for _, r := range next.References {
			got[refKey(r)]++
		}
		cp = next.ContinuationPoint
	}

	require.Equal(t, want, got, "paged browse of a MapNamespace must yield every reference exactly once")
	for k, n := range got {
		require.Equal(t, 1, n, "reference %s returned %d times", k, n)
	}
}
