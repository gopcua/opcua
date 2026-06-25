package server

import (
	"encoding/binary"
	"slices"
	"time"

	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
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

// Continuation points are stateless: encodeContinuationPoint packs the resume
// offset and the original BrowseDescription, so BrowseNext re-browses the node
// and skips to the offset. The node's references must therefore have a stable
// order across calls.
//
// Format: [offset uint32][maxRefs uint32][encoded BrowseDescription].
func (s *ViewService) encodeContinuationPoint(bd *ua.BrowseDescription, offset, maxRefs uint32) ([]byte, error) {
	bdBytes, err := ua.Encode(bd)
	if err != nil {
		return nil, err
	}
	cp := make([]byte, 8+len(bdBytes))
	binary.LittleEndian.PutUint32(cp[0:4], offset)
	binary.LittleEndian.PutUint32(cp[4:8], maxRefs)
	copy(cp[8:], bdBytes)
	return cp, nil
}

// decodeContinuationPoint decodes a stateless continuation point.
func (s *ViewService) decodeContinuationPoint(cp []byte) (bd *ua.BrowseDescription, offset, maxRefs uint32, err error) {
	if len(cp) < 8 {
		return nil, 0, 0, ua.StatusBadContinuationPointInvalid
	}
	offset = binary.LittleEndian.Uint32(cp[0:4])
	maxRefs = binary.LittleEndian.Uint32(cp[4:8])
	// The server never emits maxRefs == 0; honoring one would page in zero-sized
	// steps and never terminate, so reject it as invalid.
	if maxRefs == 0 {
		return nil, 0, 0, ua.StatusBadContinuationPointInvalid
	}
	bd = new(ua.BrowseDescription)
	if _, err := ua.Decode(cp[8:], bd); err != nil {
		return nil, 0, 0, ua.StatusBadContinuationPointInvalid
	}
	return bd, offset, maxRefs, nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.8.2
func (s *ViewService) Browse(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {

	req, err := safeReq[*ua.BrowseRequest](r)
	if err != nil {
		return nil, err
	}
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("=== Browse incoming")
	}

	maxRefs := req.RequestedMaxReferencesPerNode

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

	for i, br := range req.NodesToBrowse {
		if s.srv.cfg.logger != nil {
			s.srv.cfg.logger.Debug("    Browse of %s", br.NodeID.String())
		}
		ns, err := s.srv.Namespace(int(br.NodeID.Namespace()))
		if err != nil {
			resp.Results[i] = &ua.BrowseResult{StatusCode: ua.StatusBad}
			continue
		}

		result := ns.Browse(br)

		// Page the result if the client capped references per node.
		if maxRefs > 0 && uint32(len(result.References)) > maxRefs {
			cp, err := s.encodeContinuationPoint(br, maxRefs, maxRefs)
			if err != nil {
				resp.Results[i] = &ua.BrowseResult{StatusCode: ua.StatusBad}
				continue
			}
			result.References = result.References[:maxRefs]
			result.ContinuationPoint = cp
		}

		resp.Results[i] = result
	}

	return resp, nil

}

func suitableRef(srv *Server, desc *ua.BrowseDescription, ref *ua.ReferenceDescription) bool {
	if !suitableDirection(desc.BrowseDirection, ref.IsForward) {
		if srv.cfg.logger != nil {
			srv.cfg.logger.Debug("%v not suitable because of direction", ref)
		}
		return false
	}
	if !suitableRefType(srv, desc.ReferenceTypeID, ref.ReferenceTypeID, desc.IncludeSubtypes) {
		if srv.cfg.logger != nil {
			srv.cfg.logger.Debug("%v not suitable because of ref type", ref)
		}
		return false
	}
	if desc.NodeClassMask > 0 && desc.NodeClassMask&uint32(ref.NodeClass) == 0 {
		if srv.cfg.logger != nil {
			srv.cfg.logger.Debug("%v not suitable because of node class", ref)
		}
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
func (s *ViewService) BrowseNext(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.BrowseNextRequest](r)
	if err != nil {
		return nil, err
	}

	resp := &ua.BrowseNextResponse{
		ResponseHeader: &ua.ResponseHeader{
			Timestamp:          time.Now(),
			RequestHandle:      req.RequestHeader.RequestHandle,
			ServiceResult:      ua.StatusOK,
			ServiceDiagnostics: &ua.DiagnosticInfo{},
			StringTable:        []string{},
			AdditionalHeader:   ua.NewExtensionObject(nil),
		},
		Results:         make([]*ua.BrowseResult, len(req.ContinuationPoints)),
		DiagnosticInfos: []*ua.DiagnosticInfo{},
	}

	for i, cpBytes := range req.ContinuationPoints {
		if req.ReleaseContinuationPoints {
			// Stateless: nothing to release.
			resp.Results[i] = &ua.BrowseResult{StatusCode: ua.StatusOK}
			continue
		}

		bd, offset, maxRefs, err := s.decodeContinuationPoint(cpBytes)
		if err != nil {
			resp.Results[i] = &ua.BrowseResult{StatusCode: ua.StatusBadContinuationPointInvalid}
			continue
		}

		ns, err := s.srv.Namespace(int(bd.NodeID.Namespace()))
		if err != nil {
			resp.Results[i] = &ua.BrowseResult{StatusCode: ua.StatusBad}
			continue
		}

		// Re-browse and skip to offset
		result := ns.Browse(bd)
		if uint32(len(result.References)) <= offset {
			// No more results
			resp.Results[i] = &ua.BrowseResult{StatusCode: ua.StatusOK}
			continue
		}

		remaining := result.References[offset:]

		if uint32(len(remaining)) > maxRefs {
			cp, err := s.encodeContinuationPoint(bd, offset+maxRefs, maxRefs)
			if err != nil {
				resp.Results[i] = &ua.BrowseResult{StatusCode: ua.StatusBad}
				continue
			}
			resp.Results[i] = &ua.BrowseResult{
				StatusCode:        ua.StatusOK,
				ContinuationPoint: cp,
				References:        remaining[:maxRefs],
			}
		} else {
			resp.Results[i] = &ua.BrowseResult{
				StatusCode: ua.StatusOK,
				References: remaining,
			}
		}
	}

	return resp, nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.8.4
func (s *ViewService) TranslateBrowsePathsToNodeIDs(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.TranslateBrowsePathsToNodeIDsRequest](r)
	if err != nil {
		return nil, err
	}
	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.8.5
func (s *ViewService) RegisterNodes(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.RegisterNodesRequest](r)
	if err != nil {
		return nil, err
	}
	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.8.6
func (s *ViewService) UnregisterNodes(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.UnregisterNodesRequest](r)
	if err != nil {
		return nil, err
	}
	return serviceUnsupported(req.RequestHeader), nil
}
