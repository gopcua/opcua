// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// FindServersOnNetworkRequest returns the Servers known to a Discovery Server. Unlike FindServers, this Service is
// only implemented by Discovery Servers.
//
// The Client may reduce the number of results returned by specifying filter criteria. An empty list is
// returned if no Server matches the criteria specified by the Client.
//
// This Service shall not require message security but it may require transport layer security.
//
// Each time the Discovery Server creates or updates a record in its cache it shall assign a
// monotonically increasing identifier to the record. This allows Clients to request records in batches
// by specifying the identifier for the last record received in the last call to FindServersOnNetwork.
// To support this the Discovery Server shall return records in numerical order starting from the
// lowest record identifier. The Discovery Server shall also return the last time the counter was reset
// for example due to a restart of the Discovery Server. If a Client detects that this time is more
// recent than the last time the Client called the Service it shall call the Service again with a
// startingRecordId of 0.
//
// This Service can be used without security and it is therefore vulnerable to denial of service (DOS)
// attacks. A Server should minimize the amount of processing required to send the response for this
// Service. This can be achieved by preparing the result in advance.
//
// Specification: Part 4, 5.4.3
// type FindServersOnNetworkRequest struct {
// 	RequestHeader          *RequestHeader
// 	StartingRecordID       uint32
// 	MaxRecordsToReturn     uint32
// 	ServerCapabilityFilter []string
// }

// NewFindServersOnNetworkRequest creates a new FindServersOnNetworkRequest.
func NewFindServersOnNetworkRequest(reqHeader *RequestHeader, startRecord, maxRecords uint32, filters []string) *FindServersOnNetworkRequest {
	return &FindServersOnNetworkRequest{
		RequestHeader:          reqHeader,
		StartingRecordID:       startRecord,
		MaxRecordsToReturn:     maxRecords,
		ServerCapabilityFilter: filters,
	}
}
