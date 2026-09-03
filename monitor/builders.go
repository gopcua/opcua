package monitor

import (
	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

// buildCreateRequest builds a create request with the given handle. The
// MonitoringParameters copy is shallow: Filter and node.NodeID stay aliased
// with the caller's values.
func buildCreateRequest(node Request, handle uint32) *ua.MonitoredItemCreateRequest {
	request := opcua.NewMonitoredItemCreateRequestWithDefaults(node.NodeID, ua.AttributeIDValue, handle)
	request.MonitoringMode = node.MonitoringMode

	if node.MonitoringParameters != nil {
		params := *node.MonitoringParameters
		params.ClientHandle = handle
		request.RequestedParameters = &params
	}

	return request
}

// buildModifyRequests skips nodes with nil parameters or no matching item.
func buildModifyRequests(nodes []Request, items []Item) []*ua.MonitoredItemModifyRequest {
	requests := make([]*ua.MonitoredItemModifyRequest, 0)

	for _, node := range nodes {
		for _, item := range items {
			if item.nodeID.String() != node.NodeID.String() {
				continue
			}

			if node.MonitoringParameters == nil {
				break
			}

			params := *node.MonitoringParameters
			params.ClientHandle = item.handle
			requests = append(requests, &ua.MonitoredItemModifyRequest{
				MonitoredItemID:     item.id,
				RequestedParameters: &params,
			})
			break
		}
	}

	return requests
}
