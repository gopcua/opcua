package ua

type BrowseDirection uint32

const (
	BrowseDirectionForward BrowseDirection = 0
	BrowseDirectionInverse BrowseDirection = 1
	BrowseDirectionBoth    BrowseDirection = 2
	BrowseDirectionInvalid BrowseDirection = 3
)

type BrowseResultMask uint32

const (
	BrowseResultMaskNone           BrowseResultMask = 0x00
	BrowseResultMaskReferenceType  BrowseResultMask = 0x01
	BrowseResultMaskIsForward      BrowseResultMask = 0x02
	BrowseResultMaskNodeClass      BrowseResultMask = 0x04
	BrowseResultMaskBrowseName     BrowseResultMask = 0x08
	BrowseResultMaskDisplayName    BrowseResultMask = 0x10
	BrowseResultMaskTypeDefinition BrowseResultMask = 0x20
	BrowseResultMaskAll            BrowseResultMask = 0x3f
)

type NodeClass uint32

const (
	NodeClassNone          NodeClass = 0x00
	NodeClassObject        NodeClass = 0x01
	NodeClassVariable      NodeClass = 0x02
	NodeClassMethod        NodeClass = 0x04
	NodeClassObjectType    NodeClass = 0x08
	NodeClassVariableType  NodeClass = 0x10
	NodeClassReferenceType NodeClass = 0x20
	NodeClassDataType      NodeClass = 0x40
	NodeClassView          NodeClass = 0x80
	NodeClassAll           NodeClass = 0xff
)

type MonitoringMode uint32

const (
	MonitoringModeDisabled  MonitoringMode = 0
	MonitoringModeSampling  MonitoringMode = 1
	MonitoringModeReporting MonitoringMode = 2
)
