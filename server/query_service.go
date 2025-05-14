package server

import (
	"github.com/gopcua/opcua/internal/ualog"
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

// QueryService implements the Query Service Set.
//
// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.9
type QueryService struct {
	srv *Server
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.9.3
func (s *QueryService) QueryFirst(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	dlog := s.srv.logger.With("func", "QueryService.QueryFirst")
	dlog.Debug("Handling", "type", ualog.TypeOf(r))

	req, err := safeReq[*ua.QueryFirstRequest](r)
	if err != nil {
		return nil, err
	}
	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.9.4
func (s *QueryService) QueryNext(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	dlog := s.srv.logger.With("func", "QueryService.QueryNext")
	dlog.Debug("Handling", "type", ualog.TypeOf(r))

	req, err := safeReq[*ua.QueryNextRequest](r)
	if err != nil {
		return nil, err
	}
	return serviceUnsupported(req.RequestHeader), nil
}
