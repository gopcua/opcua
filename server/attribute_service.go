package server

import (
	"context"
	"time"

	"github.com/gopcua/opcua/internal/ualog"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

// AttributeService implements the Attribute Service Set.
//
// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.10
type AttributeService struct {
	srv *Server
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.10.2
func (s *AttributeService) Read(ctx context.Context, sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	dlog := ualog.FromContext(ctx)

	req, err := safeReq[*ua.ReadRequest](r)
	if err != nil {
		return nil, err
	}

	results := make([]*ua.DataValue, len(req.NodesToRead))
	for i, n := range req.NodesToRead {
		dlog.Debug("server: read", "node", n.NodeID, "attr", n.AttributeID)

		ns, err := s.srv.Namespace(int(n.NodeID.Namespace()))
		if err != nil {
			results[i] = &ua.DataValue{
				EncodingMask:    ua.DataValueServerTimestamp | ua.DataValueStatusCode,
				ServerTimestamp: time.Now(),
				Status:          ua.StatusBad,
			}
			continue
		}
		results[i] = ns.Attribute(n.NodeID, n.AttributeID)

	}

	response := &ua.ReadResponse{
		ResponseHeader: responseHeader(req.RequestHeader.RequestHandle, ua.StatusOK),
		Results:        results,
	}

	return response, nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.10.3
func (s *AttributeService) HistoryRead(ctx context.Context, sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {

	req, err := safeReq[*ua.HistoryReadRequest](r)
	if err != nil {
		return nil, err
	}
	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.10.4
func (s *AttributeService) Write(ctx context.Context, sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	dlog := ualog.FromContext(ctx)

	req, err := safeReq[*ua.WriteRequest](r)
	if err != nil {
		return nil, err
	}

	status := make([]ua.StatusCode, len(req.NodesToWrite))

	for i := range req.NodesToWrite {
		n := req.NodesToWrite[i]
		dlog.Debug("server: write: ", "node", n.NodeID, "attr", n.AttributeID)

		ns, err := s.srv.Namespace(int(n.NodeID.Namespace()))
		if err != nil {
			status[i] = ua.StatusBadNodeNotInView
		}

		status[i] = ns.SetAttribute(n.NodeID, n.AttributeID, n.Value)
	}
	response := &ua.WriteResponse{
		ResponseHeader: &ua.ResponseHeader{
			Timestamp:          time.Now(),
			RequestHandle:      req.RequestHeader.RequestHandle,
			ServiceResult:      ua.StatusOK,
			ServiceDiagnostics: &ua.DiagnosticInfo{},
			StringTable:        []string{},
			AdditionalHeader:   ua.NewExtensionObject(nil),
		},
		Results:         status,
		DiagnosticInfos: []*ua.DiagnosticInfo{},
	}

	return response, nil

}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.10.5
func (s *AttributeService) HistoryUpdate(ctx context.Context, sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {

	req, err := safeReq[*ua.HistoryUpdateRequest](r)
	if err != nil {
		return nil, err
	}
	return serviceUnsupported(req.RequestHeader), nil
}
