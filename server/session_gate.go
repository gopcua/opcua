// Copyright 2018-2026 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package server

import (
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
)

// sessionlessServices can be called without an activated session.
//
// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.4 (Discovery)
// https://reference.opcfoundation.org/Core/Part4/v105/docs/5.6 (Session)
var sessionlessServices = map[uint16]bool{
	id.FindServersRequest_Encoding_DefaultBinary:          true,
	id.FindServersOnNetworkRequest_Encoding_DefaultBinary: true,
	id.GetEndpointsRequest_Encoding_DefaultBinary:         true,
	id.RegisterServerRequest_Encoding_DefaultBinary:       true,
	id.RegisterServer2Request_Encoding_DefaultBinary:      true,
	id.CreateSessionRequest_Encoding_DefaultBinary:        true,
	id.ActivateSessionRequest_Encoding_DefaultBinary:      true,
	id.CloseSessionRequest_Encoding_DefaultBinary:         true,
}

// checkSession makes sure session-bound services are only called on a valid
// and activated session.
func (s *Server) checkSession(typeID uint16, req ua.Request) ua.StatusCode {
	if sessionlessServices[typeID] {
		return ua.StatusOK
	}
	hdr := req.Header()
	if hdr == nil || hdr.AuthenticationToken == nil {
		return ua.StatusBadSessionIDInvalid
	}
	sess := s.sb.Session(hdr.AuthenticationToken)
	if sess == nil {
		return ua.StatusBadSessionIDInvalid
	}
	if !sess.Activated() {
		return ua.StatusBadSessionNotActivated
	}
	return ua.StatusOK
}
