// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// ServersOnNetwork is a DNS service record that meet criteria specified in the request.
// This list is empty if no Servers meet the criteria.
//
// Specification: Part4, 5.4.3.2
// type ServersOnNetwork struct {
// 	RecordID           uint32
// 	ServerName         string
// 	DiscoveryURI       string
// 	ServerCapabilities []string
// }

// NewServersOnNetwork creates a new NewServersOnNetwork.
func NewServerOnNetwork(record uint32, serverName, discoveryURL string, serverCap []string) *ServerOnNetwork {
	return &ServerOnNetwork{
		RecordID:           record,
		ServerName:         serverName,
		DiscoveryURL:       discoveryURL,
		ServerCapabilities: serverCap,
	}
}
