package server

import (
	"context"
	"slices"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/ualog"
	"github.com/gopcua/opcua/uasc"
)

var (
	hasSubtype = ua.NewNumericNodeID(0, id.HasSubtype)
)

// ViewService implements the View Service Set.
//
// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.8
type ViewService struct {
	srv *Server
}

func NewViewService(s *Server) *ViewService {
	return &ViewService{
		srv: s,
	}
}

var newViewServiceLogAttributes = newServiceLogAttributeCreatorForSet("view")

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.8.2
func (s *ViewService) Browse(ctx context.Context, sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	ctx = ualog.WithAttrs(ctx, newViewServiceLogAttributes("browse"))
	logServiceRequest(ctx, r)

	req, err := safeReq[*ua.BrowseRequest](r)
	if err != nil {
		return nil, err
	}

	resp := &ua.BrowseResponse{
		ResponseHeader: &ua.ResponseHeader{
			Timestamp:          time.Now(),
			RequestHandle:      req.RequestHeader.RequestHandle,
			ServiceResult:      ua.StatusOK,
			ServiceDiagnostics: &ua.DiagnosticInfo{},
			StringTable:        []string{},
			AdditionalHeader:   ua.NewExtensionObject(nil),
		},
		Results: make([]*ua.BrowseResult, len(req.NodesToBrowse)),

		DiagnosticInfos: []*ua.DiagnosticInfo{{}},
	}

	for i := range req.NodesToBrowse {
		br := req.NodesToBrowse[i]
		ualog.Debug(ctx, "browsing node", ualog.Any(ualog.NodeIdKey, br.NodeID))

		ns, err := s.srv.Namespace(int(br.NodeID.Namespace()))
		if err != nil {
			resp.Results[i] = &ua.BrowseResult{StatusCode: ua.StatusBad}
			continue
		}
		resp.Results[i] = ns.Browse(ctx, br)
	}

	return resp, nil
}

func suitableRef(ctx context.Context, srv *Server, desc *ua.BrowseDescription, ref *ua.ReferenceDescription) bool {
	if !suitableDirection(desc.BrowseDirection, ref.IsForward) {
		ualog.Debug(ctx, "reference not suitable because of direction", ualog.Any("ref", ref))
		return false
	}
	if !suitableRefType(srv, desc.ReferenceTypeID, ref.ReferenceTypeID, desc.IncludeSubtypes) {
		ualog.Debug(ctx, "reference not suitable because of type", ualog.Any("ref", ref))
		return false
	}
	if desc.NodeClassMask > 0 && desc.NodeClassMask&uint32(ref.NodeClass) == 0 {
		ualog.Debug(ctx, "reference not suitable because of node class", ualog.Any("ref", ref))
		return false
	}
	return true
}

func suitableDirection(bd ua.BrowseDirection, isForward bool) bool {
	switch {
	case bd == ua.BrowseDirectionBoth:
		return true
	case bd == ua.BrowseDirectionForward && isForward:
		return true
	case bd == ua.BrowseDirectionInverse && !isForward:
		return true
	default:
		return false
	}
}

func suitableRefType(srv *Server, ref1, ref2 *ua.NodeID, subtypes bool) bool {
	if ref1.Equal(ua.NewNumericNodeID(0, 0)) {
		// refType is not specified in browse description. Return all types
		return true
	}
	if ref1.Equal(ref2) {
		return true
	}
	hasRef2Fn := func(nid *ua.NodeID) bool { return nid.Equal(ref2) }
	hasSubtypeFn := func(nid *ua.NodeID) bool { return nid.Equal(hasSubtype) }
	oktypes := getSubRefs(srv, ref1)
	if !subtypes && slices.ContainsFunc(oktypes, hasSubtypeFn) {
		for n := slices.IndexFunc(oktypes, hasSubtypeFn); n > 0; {
			oktypes = slices.Delete(oktypes, n, n+1)
		}
	}
	return slices.ContainsFunc(oktypes, hasRef2Fn)
}

func getSubRefs(srv *Server, nid *ua.NodeID) []*ua.NodeID {
	var refs []*ua.NodeID
	ns, err := srv.Namespace(int(nid.Namespace()))
	if err != nil {
		// TODO: return error
		return nil
	}
	node := ns.Node(nid)
	if node == nil {
		return nil
	}
	for _, ref := range node.refs {
		if ref.ReferenceTypeID.Equal(hasSubtype) && ref.IsForward && ref.NodeID != nil {
			refs = append(refs, ref.NodeID.NodeID)
			refs = append(refs, getSubRefs(srv, ref.NodeID.NodeID)...)
		}
	}
	return refs
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.8.3
func (s *ViewService) BrowseNext(ctx context.Context, sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	ctx = ualog.WithAttrs(ctx, newViewServiceLogAttributes("browse next"))
	logServiceRequest(ctx, r)

	req, err := safeReq[*ua.BrowseNextRequest](r)
	if err != nil {
		return nil, err
	}

	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.8.4
func (s *ViewService) TranslateBrowsePathsToNodeIDs(ctx context.Context, sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	ctx = ualog.WithAttrs(ctx, newViewServiceLogAttributes("translate browse paths to node ids"))
	logServiceRequest(ctx, r)

	req, err := safeReq[*ua.TranslateBrowsePathsToNodeIDsRequest](r)
	if err != nil {
		return nil, err
	}

	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.8.5
func (s *ViewService) RegisterNodes(ctx context.Context, sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	ctx = ualog.WithAttrs(ctx, newViewServiceLogAttributes("register nodes"))
	logServiceRequest(ctx, r)

	req, err := safeReq[*ua.RegisterNodesRequest](r)
	if err != nil {
		return nil, err
	}

	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.8.6
func (s *ViewService) UnregisterNodes(ctx context.Context, sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	ctx = ualog.WithAttrs(ctx, newViewServiceLogAttributes("unregister nodes"))
	logServiceRequest(ctx, r)

	req, err := safeReq[*ua.UnregisterNodesRequest](r)
	if err != nil {
		return nil, err
	}

	return serviceUnsupported(req.RequestHeader), nil
}
