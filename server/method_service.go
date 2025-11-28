package server

import (
	"context"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/ualog"
	"github.com/gopcua/opcua/uasc"
)

// MethodService implements the Method Service Set.
//
// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.11
type MethodService struct {
	srv *Server
}

func NewMethodService(s *Server) *MethodService {
	return &MethodService{
		srv: s,
	}
}

var newMethodServiceLogAttributes = newServiceLogAttributeCreatorForSet("method")

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.11.2
func (s *MethodService) Call(ctx context.Context, sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	ctx = ualog.WithAttrs(ctx, newMethodServiceLogAttributes("call"))
	logServiceRequest(ctx, r)

	req, err := safeReq[*ua.CallRequest](r)
	if err != nil {
		return nil, err
	}

	return serviceUnsupported(req.RequestHeader), nil
}
