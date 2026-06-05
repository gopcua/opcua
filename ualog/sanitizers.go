package ualog

import (
	"fmt"

	"github.com/gopcua/opcua/ua"
)

func sanitizedRequest(req ua.Request) any {
	if req == nil {
		return nil
	}

	out := map[string]any{
		"type":            fmt.Sprintf("%T", req),
		"service_type_id": int(ua.ServiceTypeID(req)),
	}

	if hdr := req.Header(); hdr != nil {
		out["request_handle"] = hdr.RequestHandle

		// I would generally avoid logging AuthenticationToken by default.
		// If needed, log a stable hash/fingerprint instead of the raw NodeID.
		out["timestamp"] = hdr.Timestamp
		out["timeout_hint"] = hdr.TimeoutHint
	}

	switch r := req.(type) {
	case *ua.ReadRequest:
		out["max_age"] = r.MaxAge
		out["timestamps_to_return"] = r.TimestampsToReturn
		out["nodes_to_read_count"] = len(r.NodesToRead)
		out["nodes_to_read"] = summarizeReadValueIDs(r.NodesToRead, 20)

	case *ua.WriteRequest:
		out["nodes_to_write_count"] = len(r.NodesToWrite)
		out["nodes_to_write"] = summarizeWriteValues(r.NodesToWrite, 20)

	case *ua.BrowseRequest:
		out["requested_max_references_per_node"] = r.RequestedMaxReferencesPerNode
		out["nodes_to_browse_count"] = len(r.NodesToBrowse)
		out["nodes_to_browse"] = summarizeBrowseDescriptions(r.NodesToBrowse, 20)

	case *ua.CallRequest:
		out["methods_to_call_count"] = len(r.MethodsToCall)
		out["methods_to_call"] = summarizeCallMethods(r.MethodsToCall, 20)

	case *ua.CreateMonitoredItemsRequest:
		out["subscription_id"] = r.SubscriptionID
		out["timestamps_to_return"] = r.TimestampsToReturn
		out["items_to_create_count"] = len(r.ItemsToCreate)
		out["items_to_create"] = summarizeMonitoredItems(r.ItemsToCreate, 20)

	case *ua.ModifyMonitoredItemsRequest:
		out["subscription_id"] = r.SubscriptionID
		out["timestamps_to_return"] = r.TimestampsToReturn
		out["items_to_modify_count"] = len(r.ItemsToModify)

	case *ua.DeleteMonitoredItemsRequest:
		out["subscription_id"] = r.SubscriptionID
		out["monitored_item_ids_count"] = len(r.MonitoredItemIDs)
		out["monitored_item_ids"] = limitUint32s(r.MonitoredItemIDs, 50)

	case *ua.PublishRequest:
		out["subscription_acknowledgements_count"] = len(r.SubscriptionAcknowledgements)

	case *ua.RepublishRequest:
		out["subscription_id"] = r.SubscriptionID
		out["retransmit_sequence_number"] = r.RetransmitSequenceNumber

	case *ua.CreateSubscriptionRequest:
		out["requested_publishing_interval"] = r.RequestedPublishingInterval
		out["requested_lifetime_count"] = r.RequestedLifetimeCount
		out["requested_max_keep_alive_count"] = r.RequestedMaxKeepAliveCount
		out["max_notifications_per_publish"] = r.MaxNotificationsPerPublish
		out["publishing_enabled"] = r.PublishingEnabled
		out["priority"] = r.Priority

	case *ua.ModifySubscriptionRequest:
		out["subscription_id"] = r.SubscriptionID
		out["requested_publishing_interval"] = r.RequestedPublishingInterval
		out["requested_lifetime_count"] = r.RequestedLifetimeCount
		out["requested_max_keep_alive_count"] = r.RequestedMaxKeepAliveCount
		out["max_notifications_per_publish"] = r.MaxNotificationsPerPublish
		out["priority"] = r.Priority

	case *ua.DeleteSubscriptionsRequest:
		out["subscription_ids_count"] = len(r.SubscriptionIDs)
		out["subscription_ids"] = limitUint32s(r.SubscriptionIDs, 50)

	case *ua.SetPublishingModeRequest:
		out["publishing_enabled"] = r.PublishingEnabled
		out["subscription_ids_count"] = len(r.SubscriptionIDs)
		out["subscription_ids"] = limitUint32s(r.SubscriptionIDs, 50)

	case *ua.ActivateSessionRequest:
		// Deliberately do not log user identity token, signatures, certs,
		// locale IDs, or software certs by default.
		out["client_software_certificates_count"] = len(r.ClientSoftwareCertificates)
		out["locale_ids_count"] = len(r.LocaleIDs)

	case *ua.CreateSessionRequest:
		// Deliberately avoid dumping client certificate, nonce, endpoint URL,
		// server URI, etc. Some of those may be acceptable separately, but not
		// as a raw request dump.
		out["session_name_present"] = r.SessionName != ""
		out["requested_session_timeout"] = r.RequestedSessionTimeout
		out["max_response_message_size"] = r.MaxResponseMessageSize

	default:
		// Safe fallback: no payload dump.
		out["payload"] = "omitted"
	}

	return out
}

func summarizeReadValueIDs(xs []*ua.ReadValueID, max int) []map[string]any {
	n := min(len(xs), max)
	out := make([]map[string]any, 0, n)

	for _, x := range xs[:n] {
		if x == nil {
			out = append(out, map[string]any{"nil": true})
			continue
		}

		out = append(out, map[string]any{
			"node_id":      nodeIDString(x.NodeID),
			"attribute_id": x.AttributeID,
			"index_range":  x.IndexRange,
		})
	}

	return out
}

func summarizeWriteValues(xs []*ua.WriteValue, max int) []map[string]any {
	n := min(len(xs), max)
	out := make([]map[string]any, 0, n)

	for _, x := range xs[:n] {
		if x == nil {
			out = append(out, map[string]any{"nil": true})
			continue
		}

		out = append(out, map[string]any{
			"node_id":      nodeIDString(x.NodeID),
			"attribute_id": x.AttributeID,
			"index_range":  x.IndexRange,

			// Intentionally do not include x.Value.Value.
			// The value may contain credentials, production data, large blobs,
			// or personally identifiable/operator data.
			"value_status": valueStatus(x.Value),
			"value_type":   dataValueType(x.Value),
		})
	}

	return out
}

func summarizeBrowseDescriptions(xs []*ua.BrowseDescription, max int) []map[string]any {
	n := min(len(xs), max)
	out := make([]map[string]any, 0, n)

	for _, x := range xs[:n] {
		if x == nil {
			out = append(out, map[string]any{"nil": true})
			continue
		}

		out = append(out, map[string]any{
			"node_id":           nodeIDString(x.NodeID),
			"browse_direction":  x.BrowseDirection,
			"reference_type_id": nodeIDString(x.ReferenceTypeID),
			"include_subtypes":  x.IncludeSubtypes,
			"node_class_mask":   x.NodeClassMask,
			"result_mask":       x.ResultMask,
		})
	}

	return out
}

func summarizeCallMethods(xs []*ua.CallMethodRequest, max int) []map[string]any {
	n := min(len(xs), max)
	out := make([]map[string]any, 0, n)

	for _, x := range xs[:n] {
		if x == nil {
			out = append(out, map[string]any{"nil": true})
			continue
		}

		out = append(out, map[string]any{
			"object_id":             nodeIDString(x.ObjectID),
			"method_id":             nodeIDString(x.MethodID),
			"input_arguments_count": len(x.InputArguments),

			// Deliberately do not include input argument values.
			"input_argument_types": variantTypes(x.InputArguments, 20),
		})
	}

	return out
}

func summarizeMonitoredItems(xs []*ua.MonitoredItemCreateRequest, max int) []map[string]any {
	n := min(len(xs), max)
	out := make([]map[string]any, 0, n)

	for _, x := range xs[:n] {
		if x == nil {
			out = append(out, map[string]any{"nil": true})
			continue
		}

		entry := map[string]any{
			"monitoring_mode":      x.MonitoringMode,
			"requested_parameters": summarizeMonitoringParameters(x.RequestedParameters),
		}

		if item := x.ItemToMonitor; item != nil {
			entry["node_id"] = nodeIDString(item.NodeID)
			entry["attribute_id"] = item.AttributeID
			entry["index_range"] = item.IndexRange
		} else {
			entry["item_to_monitor_nil"] = true
		}

		out = append(out, entry)
	}

	return out
}

func summarizeMonitoringParameters(p *ua.MonitoringParameters) map[string]any {
	if p == nil {
		return nil
	}

	return map[string]any{
		"client_handle":     p.ClientHandle,
		"sampling_interval": p.SamplingInterval,
		"queue_size":        p.QueueSize,
		"discard_oldest":    p.DiscardOldest,

		// Filter is intentionally omitted by default. Depending on the filter
		// type it may be noisy or contain application-specific data.
		"filter_present": p.Filter != nil,
	}
}

func nodeIDString(id *ua.NodeID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func valueStatus(v *ua.DataValue) any {
	if v == nil {
		return nil
	}

	return v.Status
}

func dataValueType(v *ua.DataValue) string {
	if v == nil || v.Value == nil {
		return ""
	}
	return fmt.Sprintf("%T", v.Value.Value())
}

func variantTypes(xs []*ua.Variant, max int) []string {
	n := min(len(xs), max)
	out := make([]string, 0, n)

	for _, v := range xs[:n] {
		if v == nil {
			out = append(out, "")
			continue
		}
		out = append(out, fmt.Sprintf("%T", v.Value()))
	}

	return out
}

func limitUint32s(xs []uint32, max int) []uint32 {
	if len(xs) <= max {
		return xs
	}
	return xs[:max]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
