package services

import (
	"time"

	uad "github.com/wmnsk/gopcua/datatypes"
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
	DiagnosticInfos []*uad.DiagnosticInfo
}

type BrowseDescription struct {
	NodeID          *uad.NodeID
	Direction       BrowseDirection
	ReferenceTypeID *uad.NodeID
	IncludeSubtypes bool
	NodeClassMask   NodeClass
	ResultMask      BrowseResultMask
}

type BrowseResult struct {
	StatusCode        uad.StatusCode
	ContinuationPoint []byte
	References        []*ReferenceDescription
}

type ReferenceDescription struct {
	ReferenceTypeID *uad.NodeID
	IsForward       bool
	NodeID          *uad.ExpandedNodeID
	BrowseName      *uad.QualifiedName
	DisplayName     *uad.LocalizedText
	NodeClass       NodeClass
	TypeDefinition  *uad.ExpandedNodeID
}

type ViewDescription struct {
	ViewID      *uad.NodeID
	Timestamp   time.Time // UtcTime
	ViewVersion uint32
}
