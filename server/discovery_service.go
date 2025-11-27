package server

import (
	"context"
	"strings"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/ualog"
	"github.com/gopcua/opcua/uasc"
)

// DiscoveryService implements the Discovery Service Set
//
// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.4
type DiscoveryService struct {
	srv *Server
}

func NewDiscoveryService(s *Server) *DiscoveryService {
	return &DiscoveryService{
		srv: s,
	}
}

var newDiscoveryServiceLogAttributes = newServiceLogAttributeCreatorForSet("discovery")

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.4.2
func (s *DiscoveryService) FindServers(ctx context.Context, sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	ctx = ualog.With(ctx, newDiscoveryServiceLogAttributes("find servers"))
	logServiceRequest(ctx, r)

	req, err := safeReq[*ua.FindServersRequest](r)
	if err != nil {
		return nil, err
	}

	response := &ua.FindServersResponse{
		ResponseHeader: responseHeader(req.RequestHeader.RequestHandle, ua.StatusOK),
		Servers: []*ua.ApplicationDescription{
			s.srv.Endpoints()[0].Server,
		},
	}

	return response, nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.4.3
func (s *DiscoveryService) FindServersOnNetwork(ctx context.Context, sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	ctx = ualog.With(ctx, newDiscoveryServiceLogAttributes("find servers on network"))
	logServiceRequest(ctx, r)

	req, err := safeReq[*ua.FindServersOnNetworkRequest](r)
	if err != nil {
		return nil, err
	}

	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.4.4
func (s *DiscoveryService) GetEndpoints(ctx context.Context, sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	ctx = ualog.With(ctx, newDiscoveryServiceLogAttributes("get endpoints"))
	logServiceRequest(ctx, r)

	req, err := safeReq[*ua.GetEndpointsRequest](r)
	if err != nil {
		return nil, err
	}

	requrl := strings.ToLower(req.EndpointURL)
	matching_endpoints := make([]*ua.EndpointDescription, 0)
	for i := range s.srv.endpoints {
		ep := s.srv.endpoints[i]
		if strings.ToLower(ep.EndpointURL) == requrl {
			matching_endpoints = append(matching_endpoints, ep)
		}
	}

	response := &ua.GetEndpointsResponse{
		ResponseHeader: responseHeader(req.RequestHeader.RequestHandle, ua.StatusOK),
		Endpoints:      matching_endpoints,
	}

	return response, nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.4.5
func (s *DiscoveryService) RegisterServer(ctx context.Context, sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	ctx = ualog.With(ctx, newDiscoveryServiceLogAttributes("register server"))
	logServiceRequest(ctx, r)

	req, err := safeReq[*ua.RegisterServerRequest](r)
	if err != nil {
		return nil, err
	}

	return serviceUnsupported(req.RequestHeader), nil
}

// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.4.6
func (s *DiscoveryService) RegisterServer2(ctx context.Context, sc *uasc.SecureChannel, r ua.Request, reqID uint32) (ua.Response, error) {
	ctx = ualog.With(ctx, newDiscoveryServiceLogAttributes("register server 2"))
	logServiceRequest(ctx, r)

	req, err := safeReq[*ua.RegisterServer2Request](r)
	if err != nil {
		return nil, err
	}

	return serviceUnsupported(req.RequestHeader), nil
}
