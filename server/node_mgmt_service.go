package server

import (
	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uasc"
)

// NodeManagementService implements the Node Management Service Set.
//
// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.7
type NodeManagementService struct {
	srv *Server
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.7.2
func (s *NodeManagementService) AddNodes(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.AddNodesRequest](r)
	if err != nil {
		return nil, err
	}
	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.7.3
func (s *NodeManagementService) AddReferences(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.AddReferencesRequest](r)
	if err != nil {
		return nil, err
	}
	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.7.4
func (s *NodeManagementService) DeleteNodes(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.DeleteNodesRequest](r)
	if err != nil {
		return nil, err
	}
	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.7.5
func (s *NodeManagementService) DeleteReferences(sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	if s.srv.cfg.logger != nil {
		s.srv.cfg.logger.Debug("Handling %T", r)
	}

	req, err := safeReq[*ua.DeleteReferencesRequest](r)
	if err != nil {
		return nil, err
	}
	return serviceUnsupported(req.RequestHeader), nil
}
