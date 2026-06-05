package ualog

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

func TestSanitizedRequestNil(t *testing.T) {
	require.Nil(t, DefaultRequestSanitizer(nil))
}

func TestSanitizedRequestReadRequest(t *testing.T) {
	req := &ua.ReadRequest{
		RequestHeader:      testRequestHeader(),
		MaxAge:             123,
		TimestampsToReturn: ua.TimestampsToReturnBoth,
		NodesToRead: []*ua.ReadValueID{
			nil,
			{
				NodeID:      ua.NewFourByteNodeID(0, 2256),
				AttributeID: ua.AttributeIDValue,
				IndexRange:  "1:2",
			},
		},
	}

	got := sanitizedRequestMap(t, req)

	require.Equal(t, "*ua.ReadRequest", got["type"])
	require.Equal(t, uint32(ua.ServiceTypeID(req)), got["service_type_id"])
	require.Equal(t, uint32(42), got["request_handle"])
	require.Equal(t, uint32(1000), got["timeout_hint"])
	require.Equal(t, float64(123), got["max_age"])
	require.Equal(t, ua.TimestampsToReturnBoth, got["timestamps_to_return"])
	require.Equal(t, 2, got["nodes_to_read_count"])

	nodes := requireMapSlice(t, got["nodes_to_read"])
	require.Len(t, nodes, 2)

	require.Equal(t, true, nodes[0]["nil"])
	require.Equal(t, "i=2256", nodes[1]["node_id"])
	require.Equal(t, ua.AttributeIDValue, nodes[1]["attribute_id"])
	require.Equal(t, "1:2", nodes[1]["index_range"])

	requireJSONDoesNotContain(t, got, "AuthenticationToken")
	requireJSONDoesNotContain(t, got, "secret")
}

func TestSanitizedRequestWriteRequestOmitsValue(t *testing.T) {
	req := &ua.WriteRequest{
		RequestHeader: testRequestHeader(),
		NodesToWrite: []*ua.WriteValue{
			nil,
			{
				NodeID:      ua.NewStringNodeID(2, "TemperatureSetpoint"),
				AttributeID: ua.AttributeIDValue,
				IndexRange:  "0",
				Value: &ua.DataValue{
					Value: ua.MustVariant("secret-write-value"),
				},
			},
			{
				NodeID:      ua.NewStringNodeID(2, "NilValue"),
				AttributeID: ua.AttributeIDValue,
				Value:       nil,
			},
		},
	}

	got := sanitizedRequestMap(t, req)

	require.Equal(t, 3, got["nodes_to_write_count"])

	nodes := requireMapSlice(t, got["nodes_to_write"])
	require.Len(t, nodes, 3)

	require.Equal(t, true, nodes[0]["nil"])
	require.Equal(t, "ns=2;s=TemperatureSetpoint", nodes[1]["node_id"])
	require.Equal(t, ua.AttributeIDValue, nodes[1]["attribute_id"])
	require.Equal(t, "0", nodes[1]["index_range"])
	require.Equal(t, "string", nodes[1]["value_type"])
	require.Equal(t, "", nodes[2]["value_type"])

	requireJSONDoesNotContain(t, got, "secret-write-value")
}

func TestSanitizedRequestBrowseRequest(t *testing.T) {
	req := &ua.BrowseRequest{
		RequestHeader:                 testRequestHeader(),
		RequestedMaxReferencesPerNode: 25,
		NodesToBrowse: []*ua.BrowseDescription{
			nil,
			{
				NodeID:          ua.NewTwoByteNodeID(84),
				BrowseDirection: ua.BrowseDirectionForward,
				ReferenceTypeID: ua.NewTwoByteNodeID(35),
				IncludeSubtypes: true,
				NodeClassMask:   1,
				ResultMask:      63,
			},
		},
	}

	got := sanitizedRequestMap(t, req)

	require.Equal(t, uint32(25), got["requested_max_references_per_node"])
	require.Equal(t, 2, got["nodes_to_browse_count"])

	nodes := requireMapSlice(t, got["nodes_to_browse"])
	require.Len(t, nodes, 2)

	require.Equal(t, true, nodes[0]["nil"])
	require.Equal(t, "i=84", nodes[1]["node_id"])
	require.Equal(t, ua.BrowseDirectionForward, nodes[1]["browse_direction"])
	require.Equal(t, "i=35", nodes[1]["reference_type_id"])
	require.Equal(t, true, nodes[1]["include_subtypes"])
	require.Equal(t, uint32(1), nodes[1]["node_class_mask"])
	require.Equal(t, uint32(63), nodes[1]["result_mask"])
}

func TestSanitizedRequestCallRequestOmitsInputArgumentValues(t *testing.T) {
	req := &ua.CallRequest{
		RequestHeader: testRequestHeader(),
		MethodsToCall: []*ua.CallMethodRequest{
			nil,
			{
				ObjectID: ua.NewStringNodeID(2, "Object"),
				MethodID: ua.NewStringNodeID(2, "Method"),
				InputArguments: []*ua.Variant{
					ua.MustVariant("secret-method-argument"),
					nil,
					ua.MustVariant(uint32(123)),
				},
			},
		},
	}

	got := sanitizedRequestMap(t, req)

	require.Equal(t, 2, got["methods_to_call_count"])

	methods := requireMapSlice(t, got["methods_to_call"])
	require.Len(t, methods, 2)

	require.Equal(t, true, methods[0]["nil"])
	require.Equal(t, "ns=2;s=Object", methods[1]["object_id"])
	require.Equal(t, "ns=2;s=Method", methods[1]["method_id"])
	require.Equal(t, 3, methods[1]["input_arguments_count"])

	argTypes := requireStringSlice(t, methods[1]["input_argument_types"])
	require.Equal(t, []string{"string", "", "uint32"}, argTypes)

	requireJSONDoesNotContain(t, got, "secret-method-argument")
}

func TestSanitizedRequestCreateMonitoredItemsRequest(t *testing.T) {
	req := &ua.CreateMonitoredItemsRequest{
		RequestHeader:      testRequestHeader(),
		SubscriptionID:     1234,
		TimestampsToReturn: ua.TimestampsToReturnBoth,
		ItemsToCreate: []*ua.MonitoredItemCreateRequest{
			nil,
			{
				ItemToMonitor:       nil,
				RequestedParameters: nil,
			},
			{
				ItemToMonitor: &ua.ReadValueID{
					NodeID:      ua.NewStringNodeID(2, "Temperature"),
					AttributeID: ua.AttributeIDValue,
					IndexRange:  "3:4",
				},
				RequestedParameters: &ua.MonitoringParameters{
					ClientHandle:     777,
					SamplingInterval: 250,
					QueueSize:        10,
					DiscardOldest:    true,
					Filter:           nil,
				},
			},
		},
	}

	got := sanitizedRequestMap(t, req)

	require.Equal(t, uint32(1234), got["subscription_id"])
	require.Equal(t, ua.TimestampsToReturnBoth, got["timestamps_to_return"])
	require.Equal(t, 3, got["items_to_create_count"])

	items := requireMapSlice(t, got["items_to_create"])
	require.Len(t, items, 3)

	require.Equal(t, true, items[0]["nil"])
	require.Equal(t, true, items[1]["item_to_monitor_nil"])
	require.Nil(t, items[1]["requested_parameters"])

	require.Equal(t, "ns=2;s=Temperature", items[2]["node_id"])
	require.Equal(t, ua.AttributeIDValue, items[2]["attribute_id"])
	require.Equal(t, "3:4", items[2]["index_range"])

	params := requireMap(t, items[2]["requested_parameters"])
	require.Equal(t, uint32(777), params["client_handle"])
	require.Equal(t, float64(250), params["sampling_interval"])
	require.Equal(t, uint32(10), params["queue_size"])
	require.Equal(t, true, params["discard_oldest"])
	require.Equal(t, false, params["filter_present"])
}

func TestSanitizedRequestSubscriptionRequests(t *testing.T) {
	t.Run("create subscription", func(t *testing.T) {
		req := &ua.CreateSubscriptionRequest{
			RequestHeader:               testRequestHeader(),
			RequestedPublishingInterval: 500,
			RequestedLifetimeCount:      2400,
			RequestedMaxKeepAliveCount:  10,
			MaxNotificationsPerPublish:  65536,
			PublishingEnabled:           true,
			Priority:                    7,
		}

		got := sanitizedRequestMap(t, req)

		require.Equal(t, float64(500), got["requested_publishing_interval"])
		require.Equal(t, uint32(2400), got["requested_lifetime_count"])
		require.Equal(t, uint32(10), got["requested_max_keep_alive_count"])
		require.Equal(t, uint32(65536), got["max_notifications_per_publish"])
		require.Equal(t, true, got["publishing_enabled"])
		require.Equal(t, byte(7), got["priority"])
	})

	t.Run("delete subscriptions caps ids", func(t *testing.T) {
		req := &ua.DeleteSubscriptionsRequest{
			RequestHeader:   testRequestHeader(),
			SubscriptionIDs: makeUint32s(55),
		}

		got := sanitizedRequestMap(t, req)

		require.Equal(t, 55, got["subscription_ids_count"])

		ids := requireUint32Slice(t, got["subscription_ids"])
		require.Len(t, ids, 50)
		require.Equal(t, uint32(1), ids[0])
		require.Equal(t, uint32(50), ids[49])
	})

	t.Run("delete monitored items caps ids", func(t *testing.T) {
		req := &ua.DeleteMonitoredItemsRequest{
			RequestHeader:    testRequestHeader(),
			SubscriptionID:   987,
			MonitoredItemIDs: makeUint32s(55),
		}

		got := sanitizedRequestMap(t, req)

		require.Equal(t, uint32(987), got["subscription_id"])
		require.Equal(t, 55, got["monitored_item_ids_count"])

		ids := requireUint32Slice(t, got["monitored_item_ids"])
		require.Len(t, ids, 50)
		require.Equal(t, uint32(1), ids[0])
		require.Equal(t, uint32(50), ids[49])
	})
}

func TestSanitizedRequestSessionRequestsOmitSensitiveFields(t *testing.T) {
	t.Run("create session", func(t *testing.T) {
		req := &ua.CreateSessionRequest{
			RequestHeader:           testRequestHeader(),
			SessionName:             "test-session",
			RequestedSessionTimeout: 60000,
			MaxResponseMessageSize:  65536,

			ClientCertificate: []byte("secret-client-certificate"),
			ClientNonce:       []byte("secret-client-nonce"),
			EndpointURL:       "opc.tcp://secret-endpoint",
			ServerURI:         "secret-server-uri",
		}

		got := sanitizedRequestMap(t, req)

		require.Equal(t, true, got["session_name_present"])
		require.Equal(t, float64(60000), got["requested_session_timeout"])
		require.Equal(t, uint32(65536), got["max_response_message_size"])

		requireJSONDoesNotContain(t, got, "secret-client-certificate")
		requireJSONDoesNotContain(t, got, "secret-client-nonce")
		requireJSONDoesNotContain(t, got, "secret-endpoint")
		requireJSONDoesNotContain(t, got, "secret-server-uri")
	})

	t.Run("activate session", func(t *testing.T) {
		req := &ua.ActivateSessionRequest{
			RequestHeader: testRequestHeader(),
			ClientSignature: &ua.SignatureData{
				Algorithm: "secret-signature-algorithm",
				Signature: []byte("secret-signature"),
			},
			ClientSoftwareCertificates: []*ua.SignedSoftwareCertificate{
				{
					CertificateData: []byte("secret-software-certificate"),
					Signature:       []byte("secret-software-certificate-signature"),
				},
			},
			LocaleIDs: []string{"en-US", "sv-SE"},
		}

		got := sanitizedRequestMap(t, req)

		require.Equal(t, 1, got["client_software_certificates_count"])
		require.Equal(t, 2, got["locale_ids_count"])

		requireJSONDoesNotContain(t, got, "secret-signature")
		requireJSONDoesNotContain(t, got, "secret-signature-algorithm")
		requireJSONDoesNotContain(t, got, "secret-software-certificate")
	})
}

func TestSanitizedRequestFallbackOmitsPayload(t *testing.T) {
	req := &ua.CloseSessionRequest{
		RequestHeader:       testRequestHeader(),
		DeleteSubscriptions: true,
	}

	got := sanitizedRequestMap(t, req)

	require.Equal(t, "*ua.CloseSessionRequest", got["type"])
	require.Equal(t, uint32(ua.ServiceTypeID(req)), got["service_type_id"])
	require.Equal(t, "omitted", got["payload"])
	require.NotContains(t, got, "delete_subscriptions")
}

func TestRequestAttrUsesSanitizedRequest(t *testing.T) {
	req := &ua.ReadRequest{
		RequestHeader: testRequestHeader(),
		NodesToRead:   []*ua.ReadValueID{{NodeID: ua.NewTwoByteNodeID(84)}},
	}

	attr := Request(context.Background(), req)

	require.Equal(t, "request", attr.Key)

	got := attr.Value.Any()
	gotMap := requireMap(t, got)

	require.Equal(t, "*ua.ReadRequest", gotMap["type"])
	require.Equal(t, 1, gotMap["nodes_to_read_count"])
}

func testRequestHeader() *ua.RequestHeader {
	return &ua.RequestHeader{
		AuthenticationToken: ua.NewByteStringNodeID(0, []byte("secret-auth-token")),
		Timestamp:           time.Date(2026, time.June, 5, 12, 0, 0, 0, time.UTC),
		RequestHandle:       42,
		TimeoutHint:         1000,
		AdditionalHeader:    ua.NewExtensionObject(nil),
	}
}

func sanitizedRequestMap(t *testing.T, req ua.Request) map[string]any {
	t.Helper()

	return requireMap(t, DefaultRequestSanitizer(req))
}

func requireMap(t *testing.T, v any) map[string]any {
	t.Helper()

	got, ok := v.(map[string]any)
	require.Truef(t, ok, "got %T, want map[string]any", v)

	return got
}

func requireMapSlice(t *testing.T, v any) []map[string]any {
	t.Helper()

	got, ok := v.([]map[string]any)
	require.Truef(t, ok, "got %T, want []map[string]any", v)

	return got
}

func requireStringSlice(t *testing.T, v any) []string {
	t.Helper()

	got, ok := v.([]string)
	require.Truef(t, ok, "got %T, want []string", v)

	return got
}

func requireUint32Slice(t *testing.T, v any) []uint32 {
	t.Helper()

	got, ok := v.([]uint32)
	require.Truef(t, ok, "got %T, want []uint32", v)

	return got
}

func requireJSONDoesNotContain(t *testing.T, v any, forbidden string) {
	t.Helper()

	b, err := json.Marshal(v)
	require.NoError(t, err)

	require.Falsef(t, strings.Contains(string(b), forbidden), "sanitized JSON contains %q: %s", forbidden, b)
}

func makeUint32s(n int) []uint32 {
	out := make([]uint32, n)
	for i := range out {
		out[i] = uint32(i + 1)
	}
	return out
}
