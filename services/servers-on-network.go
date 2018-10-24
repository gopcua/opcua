// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

// ServersOnNetwork is a DNS service record that meet criteria specified in the request.
// This list is empty if no Servers meet the criteria.
//
// Specification: Part4, 5.4.3.2
type ServersOnNetwork struct {
	RecordID           uint32
	ServerName         string
	DiscoveryURI       string
	ServerCapabilities []string
}

// NewServersOnNetwork creates a new NewServersOnNetwork.
func NewServersOnNetwork(record uint32, serverName, discoveryURI string, serverCap []string) *ServersOnNetwork {
	return &ServersOnNetwork{
		RecordID:           record,
		ServerName:         serverName,
		DiscoveryURI:       discoveryURI,
		ServerCapabilities: serverCap,
	}
}
