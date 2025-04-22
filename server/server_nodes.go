package server

import (
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/server/attrs"
	"github.com/gopcua/opcua/ua"
)

func CurrentTimeNode() *Node {
	return NewNode(
		ua.NewNumericNodeID(0, id.Server_ServerStatus_CurrentTime),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: DataValueFromValue(attrs.BrowseName("CurrentTime")),
			ua.AttributeIDNodeClass:  DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return DataValueFromValue(time.Now()) },
	)
}

func NamespacesNode(s *Server) *Node {
	return NewNode(
		ua.NewNumericNodeID(0, id.Server_NamespaceArray),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: DataValueFromValue(attrs.BrowseName("Namespaces")),
			ua.AttributeIDNodeClass:  DataValueFromValue(uint32(ua.NodeClassObject)),
		},
		nil,
		func() *ua.DataValue {
			n := s.Namespaces()
			ns := make([]string, len(n))
			for i := range ns {
				ns[i] = n[i].Name()
			}
			return DataValueFromValue(ns)
		},
	)
}

func ServerCapabilitiesNodes(s *Server) []*Node {
	var nodes []*Node
	nodes = append(nodes, NewNode(
		ua.NewNumericNodeID(0, id.Server_ServerCapabilities_OperationLimits_MaxNodesPerRead),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: DataValueFromValue(attrs.BrowseName("MaxNodesPerRead")),
			ua.AttributeIDNodeClass:  DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return DataValueFromValue(s.cfg.cap.OperationalLimits.MaxNodesPerRead) },
	))
	return nodes
}

func RootNode() *Node {
	return NewNode(
		ua.NewNumericNodeID(0, id.RootFolder),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDNodeClass:  DataValueFromValue(attrs.NodeClass(ua.NodeClassObject)),
			ua.AttributeIDBrowseName: DataValueFromValue(attrs.BrowseName("Root")),
			ua.AttributeIDDataType:   DataValueFromValue(ua.NewNumericExpandedNodeID(0, id.DataTypesFolder)),
		},
		nil,
		nil,
	)
}

func ServerStatusNodes(s *Server, ServerNode *Node) []*Node {

	/*
		Server_ServerArray                                                                                                                                                    = 2254
		Server_NamespaceArray                                                                                                                                                 = 2255
		Server_ServerStatus_BuildInfo                                                                                                                                         = 2260
		Server_ServerStatus_BuildInfo_ProductName                                                                                                                             = 2261
		Server_ServerStatus_BuildInfo_ProductURI                                                                                                                              = 2262
		Server_ServerStatus_BuildInfo_ManufacturerName                                                                                                                        = 2263
		Server_ServerStatus_BuildInfo_SoftwareVersion                                                                                                                         = 2264
		Server_ServerStatus_BuildInfo_BuildNumber                                                                                                                             = 2265
		Server_ServerStatus_BuildInfo_BuildDate                                                                                                                               = 2266
		Server_ServiceLevel                                                                                                                                                   = 2267
		Server_ServerCapabilities                                                                                                                                             = 2268
		Server_ServerCapabilities_ServerProfileArray                                                                                                                          = 2269
		Server_ServerCapabilities_LocaleIDArray                                                                                                                               = 2271
		Server_ServerCapabilities_MinSupportedSampleRate                                                                                                                      = 2272
		Server_ServerDiagnostics                                                                                                                                              = 2274
		Server_ServerDiagnostics_ServerDiagnosticsSummary                                                                                                                     = 2275
		Server_ServerDiagnostics_ServerDiagnosticsSummary_ServerViewCount                                                                                                     = 2276
		Server_ServerDiagnostics_ServerDiagnosticsSummary_CurrentSessionCount                                                                                                 = 2277
		Server_ServerDiagnostics_ServerDiagnosticsSummary_CumulatedSessionCount                                                                                               = 2278
		Server_ServerDiagnostics_ServerDiagnosticsSummary_SecurityRejectedSessionCount                                                                                        = 2279
		Server_ServerDiagnostics_ServerDiagnosticsSummary_SessionTimeoutCount                                                                                                 = 2281
		Server_ServerDiagnostics_ServerDiagnosticsSummary_SessionAbortCount                                                                                                   = 2282
		Server_ServerDiagnostics_ServerDiagnosticsSummary_PublishingIntervalCount                                                                                             = 2284
		Server_ServerDiagnostics_ServerDiagnosticsSummary_CurrentSubscriptionCount                                                                                            = 2285
		Server_ServerDiagnostics_ServerDiagnosticsSummary_CumulatedSubscriptionCount                                                                                          = 2286
		Server_ServerDiagnostics_ServerDiagnosticsSummary_SecurityRejectedRequestsCount                                                                                       = 2287
		Server_ServerDiagnostics_ServerDiagnosticsSummary_RejectedRequestsCount                                                                                               = 2288
		Server_ServerDiagnostics_SamplingIntervalDiagnosticsArray                                                                                                             = 2289
		Server_ServerDiagnostics_SubscriptionDiagnosticsArray                                                                                                                 = 2290
		Server_ServerDiagnostics_EnabledFlag                                                                                                                                  = 2294
		Server_VendorServerInfo                                                                                                                                               = 2295
		Server_ServerRedundancy                                                                                                                                               = 2296
	*/

	sStatus := NewNode(
		ua.NewNumericNodeID(0, id.Server_ServerStatus),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: DataValueFromValue(attrs.BrowseName("Status")),
			ua.AttributeIDNodeClass:  DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return DataValueFromValue(ua.NewExtensionObject(s.Status())) },
	)

	sState := NewNode(
		ua.NewNumericNodeID(0, id.Server_ServerStatus_State),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: DataValueFromValue(attrs.BrowseName("ServerStatus")),
			ua.AttributeIDNodeClass:  DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return DataValueFromValue(int32(s.Status().State)) },
	)
	mName := NewNode(
		ua.NewNumericNodeID(0, id.Server_ServerStatus_BuildInfo_ManufacturerName),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: DataValueFromValue(attrs.BrowseName("ProductName")),
			ua.AttributeIDNodeClass:  DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return DataValueFromValue(s.cfg.manufacturerName) },
	)
	pName := NewNode(
		ua.NewNumericNodeID(0, id.Server_ServerStatus_BuildInfo_ProductName),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: DataValueFromValue(attrs.BrowseName("ProductName")),
			ua.AttributeIDNodeClass:  DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return DataValueFromValue(s.cfg.productName) },
	)

	pURI := NewNode(
		ua.NewNumericNodeID(0, id.Server_ServerStatus_BuildInfo_ProductURI),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: DataValueFromValue(attrs.BrowseName("ProductURI")),
			ua.AttributeIDNodeClass:  DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return DataValueFromValue(s.cfg.applicationURI) },
	)

	bInfo := NewNode(
		ua.NewNumericNodeID(0, id.Server_ServerStatus_BuildInfo),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: DataValueFromValue(attrs.BrowseName("BuildInfo")),
			ua.AttributeIDNodeClass:  DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return DataValueFromValue("") },
	)
	sVersion := NewNode(
		ua.NewNumericNodeID(0, id.Server_ServerStatus_BuildInfo_SoftwareVersion),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: DataValueFromValue(attrs.BrowseName("SoftwareVersion")),
			ua.AttributeIDNodeClass:  DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return DataValueFromValue(s.cfg.softwareVersion) },
	)

	bNumber := NewNode(
		ua.NewNumericNodeID(0, id.Server_ServerStatus_BuildInfo_BuildNumber),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: DataValueFromValue(attrs.BrowseName("BuildNumber")),
			ua.AttributeIDNodeClass:  DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return DataValueFromValue(s.cfg.softwareVersion) },
	)

	ts := time.Now()
	bDate := NewNode(
		ua.NewNumericNodeID(0, id.Server_ServerStatus_BuildInfo_BuildDate),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: DataValueFromValue(attrs.BrowseName("BuildDate")),
			ua.AttributeIDNodeClass:  DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return DataValueFromValue(ts) },
	)
	timeStart := NewNode(
		ua.NewNumericNodeID(0, id.Server_ServerStatus_StartTime),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: DataValueFromValue(attrs.BrowseName("StartTime")),
			ua.AttributeIDNodeClass:  DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return DataValueFromValue(ts) },
	)
	timeCurrent := NewNode(
		ua.NewNumericNodeID(0, id.Server_ServerStatus_CurrentTime),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: DataValueFromValue(attrs.BrowseName("CurrentTime")),
			ua.AttributeIDNodeClass:  DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return DataValueFromValue(time.Now()) },
	)

	//Server_ServerStatus_SecondsTillShutdown                                                                                                                               = 2992
	//Server_ServerStatus_ShutdownReason                                                                                                                                    = 2993
	sTillShutdown := NewNode(
		ua.NewNumericNodeID(0, id.Server_ServerStatus_SecondsTillShutdown),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: DataValueFromValue(attrs.BrowseName("SecondsTillShutdown")),
			ua.AttributeIDNodeClass:  DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return DataValueFromValue(int32(0)) },
	)
	sReason := NewNode(
		ua.NewNumericNodeID(0, id.Server_ServerStatus_ShutdownReason),
		map[ua.AttributeID]*ua.DataValue{
			ua.AttributeIDBrowseName: DataValueFromValue(attrs.BrowseName("ShutdownReason")),
			ua.AttributeIDNodeClass:  DataValueFromValue(uint32(ua.NodeClassVariable)),
		},
		nil,
		func() *ua.DataValue { return DataValueFromValue(int32(0)) },
	)

	nodes := []*Node{sState, mName, pName, pURI, sVersion, bNumber, bDate, timeStart, timeCurrent, bInfo, sTillShutdown, sReason}
	for i := range nodes {
		sStatus.AddRef(nodes[i], id.HasComponent, true)
	}
	ServerNode.AddRef(sStatus, id.HasComponent, true)

	nodes = append(nodes, sStatus)

	return nodes
}
