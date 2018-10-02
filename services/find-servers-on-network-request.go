// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/binary"

	"github.com/wmnsk/gopcua/datatypes"
)

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
type FindServersOnNetworkRequest struct {
	TypeID *datatypes.ExpandedNodeID
	*RequestHeader
	StartingRecordID       uint32
	MaxRecordsToReturn     uint32
	ServerCapabilityFilter *datatypes.StringArray
}

// NewFindServersOnNetworkRequest creates a new FindServersOnNetworkRequest.
func NewFindServersOnNetworkRequest(reqHeader *RequestHeader, startRecord, maxRecords uint32, servers []string) *FindServersOnNetworkRequest {
	return &FindServersOnNetworkRequest{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(
				0, ServiceTypeFindServersOnNetworkRequest,
			),
			"", 0,
		),
		RequestHeader:          reqHeader,
		StartingRecordID:       startRecord,
		MaxRecordsToReturn:     maxRecords,
		ServerCapabilityFilter: datatypes.NewStringArray(servers),
	}
}

// DecodeFindServersOnNetworkRequest decodes given bytes into FindServersOnNetworkRequest.
func DecodeFindServersOnNetworkRequest(b []byte) (*FindServersOnNetworkRequest, error) {
	f := &FindServersOnNetworkRequest{}
	if err := f.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return f, nil
}

// DecodeFromBytes decodes given bytes into FindServersOnNetworkRequest.
func (f *FindServersOnNetworkRequest) DecodeFromBytes(b []byte) error {
	offset := 0

	// type id
	f.TypeID = &datatypes.ExpandedNodeID{}
	if err := f.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += f.TypeID.Len()

	f.RequestHeader = &RequestHeader{}
	if err := f.RequestHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += f.RequestHeader.Len() - len(f.RequestHeader.Payload)

	f.StartingRecordID = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	f.MaxRecordsToReturn = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	f.ServerCapabilityFilter = &datatypes.StringArray{}
	return f.ServerCapabilityFilter.DecodeFromBytes(b[offset:])
}

// Serialize serializes FindServersOnNetworkRequest into bytes.
func (f *FindServersOnNetworkRequest) Serialize() ([]byte, error) {
	b := make([]byte, f.Len())
	if err := f.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes FindServersOnNetworkRequest into bytes.
func (f *FindServersOnNetworkRequest) SerializeTo(b []byte) error {
	var offset = 0
	if f.TypeID != nil {
		if err := f.TypeID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += f.TypeID.Len()
	}

	if f.RequestHeader != nil {
		if err := f.RequestHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += f.RequestHeader.Len()
	}

	binary.LittleEndian.PutUint32(b[offset:offset+4], f.StartingRecordID)
	offset += 4

	binary.LittleEndian.PutUint32(b[offset:offset+4], f.MaxRecordsToReturn)
	offset += 4

	if f.ServerCapabilityFilter != nil {
		if err := f.ServerCapabilityFilter.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += f.ServerCapabilityFilter.Len()
	}

	return nil
}

// Len returns the actual length of FindServersOnNetworkRequest.
func (f *FindServersOnNetworkRequest) Len() int {
	l := 8

	if f.TypeID != nil {
		l += f.TypeID.Len()
	}

	if f.RequestHeader != nil {
		l += f.RequestHeader.Len()
	}

	if f.ServerCapabilityFilter != nil {
		l += f.ServerCapabilityFilter.Len()
	}

	return l
}

// ServiceType returns type of Service.
func (f *FindServersOnNetworkRequest) ServiceType() uint16 {
	return ServiceTypeFindServersOnNetworkRequest
}
