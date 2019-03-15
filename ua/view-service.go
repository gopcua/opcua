package ua

import (
	"time"
)

type BrowseRequest struct {
	RequestHeader                 *RequestHeader
	View                          *ViewDescription
	RequestedMaxReferencesPerNode uint32
	NodesToBrowse                 []*BrowseDescription
}

type BrowseResponse struct {
	ResponseHeader  *ResponseHeader
	Results         []*BrowseResult
	DiagnosticInfos []*DiagnosticInfo
}

type BrowseDescription struct {
	NodeID          *NodeID
	Direction       BrowseDirection
	ReferenceTypeID *NodeID
	IncludeSubtypes bool
	NodeClassMask   NodeClass
	ResultMask      BrowseResultMask
}

type BrowseResult struct {
	StatusCode        StatusCode
	ContinuationPoint []byte
	References        []*ReferenceDescription
}

type ReferenceDescription struct {
	ReferenceTypeID *NodeID
	IsForward       bool
	NodeID          *ExpandedNodeID
	BrowseName      *QualifiedName
	DisplayName     *LocalizedText
	NodeClass       NodeClass
	TypeDefinition  *ExpandedNodeID
}

type ViewDescription struct {
	ViewID      *NodeID
	Timestamp   time.Time // UtcTime
	ViewVersion uint32
}
