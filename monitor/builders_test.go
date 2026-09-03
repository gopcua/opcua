package monitor

import (
	"testing"

	"github.com/gopcua/opcua/ua"
)

func TestBuildCreateRequestKeepsCallerParameters(t *testing.T) {
	filter := &ua.ExtensionObject{}
	// All values differ from constructor defaults so a request that ignored
	// the caller cannot pass by coincidence.
	caller := &ua.MonitoringParameters{
		SamplingInterval: 250,
		QueueSize:        42,
		DiscardOldest:    false,
		Filter:           filter,
	}
	node := Request{
		NodeID:               ua.NewNumericNodeID(0, 2258),
		MonitoringParameters: caller,
	}

	got := buildCreateRequest(node, 7).RequestedParameters

	if got.SamplingInterval != 250 {
		t.Errorf("SamplingInterval = %v, want 250", got.SamplingInterval)
	}
	if got.QueueSize != 42 {
		t.Errorf("QueueSize = %d, want 42", got.QueueSize)
	}
	if got.DiscardOldest {
		t.Error("DiscardOldest = true, want false")
	}
	if got.Filter != filter {
		t.Errorf("Filter = %v, want the caller's filter", got.Filter)
	}
	if got.ClientHandle != 7 {
		t.Errorf("ClientHandle = %d, want 7", got.ClientHandle)
	}
}

func TestBuildCreateRequestWithNilParametersUsesDefaults(t *testing.T) {
	node := Request{NodeID: ua.NewNumericNodeID(0, 2258)}

	got := buildCreateRequest(node, 9).RequestedParameters

	if got.ClientHandle != 9 {
		t.Errorf("ClientHandle = %d, want 9", got.ClientHandle)
	}
	if got.QueueSize != 10 {
		t.Errorf("QueueSize = %d, want the default 10", got.QueueSize)
	}
	if !got.DiscardOldest {
		t.Error("DiscardOldest = false, want the default true")
	}
	if got.SamplingInterval != 0 {
		t.Errorf("SamplingInterval = %v, want the default 0", got.SamplingInterval)
	}
	if got.Filter != nil {
		t.Errorf("Filter = %v, want the default nil", got.Filter)
	}
}

func TestBuildCreateRequestKeepsCallerMonitoringMode(t *testing.T) {
	node := Request{
		NodeID:         ua.NewNumericNodeID(0, 2258),
		MonitoringMode: ua.MonitoringModeSampling,
	}

	got := buildCreateRequest(node, 3).MonitoringMode

	if got != ua.MonitoringModeSampling {
		t.Errorf("MonitoringMode = %v, want %v", got, ua.MonitoringModeSampling)
	}
}

// TestBuildCreateRequestDoesNotWriteCallerParameters verifies that reusing one
// Request produces a fresh copy each time, not a shared struct.
func TestBuildCreateRequestDoesNotWriteCallerParameters(t *testing.T) {
	shared := &ua.MonitoringParameters{SamplingInterval: 100, QueueSize: 5}
	before := *shared
	node := Request{
		NodeID:               ua.NewNumericNodeID(0, 2258),
		MonitoringParameters: shared,
	}

	first := buildCreateRequest(node, 11)
	second := buildCreateRequest(node, 12)

	if *shared != before {
		t.Errorf("caller's parameters = %+v, want them untouched at %+v", *shared, before)
	}
	if first.RequestedParameters == shared || second.RequestedParameters == shared {
		t.Error("request aliases the caller's parameters, want a copy")
	}
	if first.RequestedParameters == second.RequestedParameters {
		t.Error("both requests share one parameters struct, want one per call")
	}
	if first.RequestedParameters.ClientHandle != 11 {
		t.Errorf("first ClientHandle = %d, want 11", first.RequestedParameters.ClientHandle)
	}
	if second.RequestedParameters.ClientHandle != 12 {
		t.Errorf("second ClientHandle = %d, want 12", second.RequestedParameters.ClientHandle)
	}
}

func TestBuildModifyRequestsStampsExistingHandleOnACopy(t *testing.T) {
	nodeID := ua.NewNumericNodeID(0, 2258)
	caller := &ua.MonitoringParameters{SamplingInterval: 500, QueueSize: 3}
	before := *caller
	items := []Item{{id: 77, nodeID: nodeID, handle: 55}}

	got := buildModifyRequests([]Request{{NodeID: nodeID, MonitoringParameters: caller}}, items)

	if len(got) != 1 {
		t.Fatalf("built %d requests, want 1", len(got))
	}
	if got[0].MonitoredItemID != 77 {
		t.Errorf("MonitoredItemID = %d, want 77", got[0].MonitoredItemID)
	}
	if got[0].RequestedParameters == caller {
		t.Error("request aliases the caller's parameters, want a copy")
	}
	if *caller != before {
		t.Errorf("caller's parameters = %+v, want them untouched at %+v", *caller, before)
	}
	if got[0].RequestedParameters.ClientHandle != 55 {
		t.Errorf("ClientHandle = %d, want the item's existing handle 55", got[0].RequestedParameters.ClientHandle)
	}
	if got[0].RequestedParameters.SamplingInterval != 500 {
		t.Errorf("SamplingInterval = %v, want 500", got[0].RequestedParameters.SamplingInterval)
	}
	if got[0].RequestedParameters.QueueSize != 3 {
		t.Errorf("QueueSize = %d, want 3", got[0].RequestedParameters.QueueSize)
	}
}

func TestBuildModifyRequestsSkipsNilParameters(t *testing.T) {
	nodeID := ua.NewNumericNodeID(0, 2258)
	items := []Item{{id: 77, nodeID: nodeID, handle: 55}}

	got := buildModifyRequests([]Request{{NodeID: nodeID}}, items)

	if len(got) != 0 {
		t.Errorf("built %d requests, want none: a node with nil parameters has nothing to modify", len(got))
	}
}

// TestBuildModifyRequestsSkipsNodesMatchingNoItem verifies that unmonitored
// nodes produce no request.
func TestBuildModifyRequestsSkipsNodesMatchingNoItem(t *testing.T) {
	monitored := ua.NewNumericNodeID(0, 2258)
	items := []Item{{id: 77, nodeID: monitored, handle: 55}}
	unmonitored := Request{
		NodeID:               ua.NewNumericNodeID(0, 2259),
		MonitoringParameters: &ua.MonitoringParameters{SamplingInterval: 500},
	}

	got := buildModifyRequests([]Request{unmonitored}, items)

	if len(got) != 0 {
		t.Errorf("built %d requests, want none for an unmonitored node", len(got))
	}
}

// TestBuildModifyRequestsBuildsOneRequestPerNode verifies one request per
// Request even when multiple items share a NodeID.
func TestBuildModifyRequestsBuildsOneRequestPerNode(t *testing.T) {
	nodeID := ua.NewNumericNodeID(0, 2258)
	items := []Item{
		{id: 77, nodeID: nodeID, handle: 55},
		{id: 88, nodeID: nodeID, handle: 66},
	}
	node := Request{NodeID: nodeID, MonitoringParameters: &ua.MonitoringParameters{SamplingInterval: 500}}

	got := buildModifyRequests([]Request{node}, items)

	if len(got) != 1 {
		t.Fatalf("built %d requests for one node matching two items, want 1", len(got))
	}
	if got[0].MonitoredItemID != 77 {
		t.Errorf("MonitoredItemID = %d, want the first matching item 77", got[0].MonitoredItemID)
	}
	if got[0].RequestedParameters.ClientHandle != 55 {
		t.Errorf("ClientHandle = %d, want the first matching item's handle 55", got[0].RequestedParameters.ClientHandle)
	}
}

// TestBuildModifyRequestsStampsEachNodesOwnHandle verifies that two nodes
// produce two independent requests, each with its own handle and parameters
// copy, and that both are modified.
func TestBuildModifyRequestsStampsEachNodesOwnHandle(t *testing.T) {
	first := ua.NewNumericNodeID(0, 2258)
	second := ua.NewNumericNodeID(0, 2259)
	items := []Item{
		{id: 77, nodeID: first, handle: 55},
		{id: 88, nodeID: second, handle: 66},
	}
	nodes := []Request{
		{NodeID: first, MonitoringParameters: &ua.MonitoringParameters{SamplingInterval: 250}},
		{NodeID: second, MonitoringParameters: &ua.MonitoringParameters{SamplingInterval: 500}},
	}

	got := buildModifyRequests(nodes, items)

	if len(got) != 2 {
		t.Fatalf("built %d requests for two monitored nodes, want 2", len(got))
	}
	if got[0].RequestedParameters == got[1].RequestedParameters {
		t.Error("both requests share one parameters struct, want one per node")
	}
	if got[0].MonitoredItemID != 77 || got[0].RequestedParameters.ClientHandle != 55 {
		t.Errorf("first request modifies item %d with handle %d, want item 77 with handle 55",
			got[0].MonitoredItemID, got[0].RequestedParameters.ClientHandle)
	}
	if got[1].MonitoredItemID != 88 || got[1].RequestedParameters.ClientHandle != 66 {
		t.Errorf("second request modifies item %d with handle %d, want item 88 with handle 66",
			got[1].MonitoredItemID, got[1].RequestedParameters.ClientHandle)
	}
	if got[0].RequestedParameters.SamplingInterval != 250 {
		t.Errorf("first SamplingInterval = %v, want 250", got[0].RequestedParameters.SamplingInterval)
	}
	if got[1].RequestedParameters.SamplingInterval != 500 {
		t.Errorf("second SamplingInterval = %v, want 500", got[1].RequestedParameters.SamplingInterval)
	}
}
