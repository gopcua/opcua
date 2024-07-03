package server

import (
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

// MethodService implements the Method Service Set.
//
// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.11
type MethodService struct {
	srv *Server
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.11.2
func (s *MethodService) Call(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.CallRequest](r)
	if err != nil {
		return nil, err
	}
	return serviceUnsupported(req.RequestHeader), nil
}
