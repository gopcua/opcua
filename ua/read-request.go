// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package ua

// TimestampsToReturn is an enumeration that specifies the Timestamp Attributes to be
// transmitted for MonitoredItems or Nodes in Read and HistoryRead.
//
// Specification: Part 4, 7.35
type TimestampsToReturn uint32

// TimestampsToReturn definitions.
const (
	// Return the source timestamp.
	TimestampsToReturnSource TimestampsToReturn = iota

	// Return the Server timestamp.
	TimestampsToReturnServer

	// Return both the source and Server timestamps.
	TimestampsToReturnBoth

	// Return neither timestamp.
	// This is the default value for MonitoredItems if a Variable value is not being accessed.
	// For HistoryRead this is not a valid setting.
	TimestampsToReturnNeither
)

// ReadRequest is used to read one or more Attributes of one or more Nodes.
// For constructed Attribute values whose elements are indexed, such as an array,
// this Service allows Clients to read the entire set of indexed values as a composite,
// to read individual elements or to read ranges of elements of the composite.
//
// Specification: Part 4, 5.10.2.2
type ReadRequest struct {
	*RequestHeader

	// Maximum age of the value to be read in milliseconds.
	// The age of the value is based on the difference between the ServerTimestamp
	// and the time when the Server starts processing the request.
	// For example if the Client specifies a maxAge of 500 milliseconds
	// and it takes 100 milliseconds until the Server starts processing the request,
	// the age of the returned value could be 600 milliseconds prior to the time it was requested.
	//
	// If the Server has one or more values of an Attribute that are within the maximum age,
	// it can return any one of the values or it can read a new value from the data source.
	// The number of values of an Attribute that a Server has depends on the
	// number of MonitoredItems that are defined for the Attribute.
	// In any case, the Client can make no assumption about which copy of the data will be returned.
	// If the Server does not have a value that is within the maximum age,
	// it shall attempt to read a new value from the data source.
	//
	// If the Server cannot meet the requested maxAge, it returns its “best effort” value
	// rather than rejecting the request. This may occur when the time it takes the
	// Server to process and return the new data value after it has been accessed is
	// greater than the specified maximum age.
	//
	// If maxAge is set to 0, the Server shall attempt to read a new value from the data source.
	//
	// If maxAge is set to the max Int32 value or greater, the Server shall attempt to get
	// a cached value.
	//
	// Negative values are invalid for maxAge.
	MaxAge uint64

	// An enumeration that specifies the Timestamps to be returned for each requested
	// Variable Value Attribute.
	TimestampsToReturn TimestampsToReturn

	// List of Nodes and their Attributes to read. For each entry in this list,
	// a StatusCode is returned, and if it indicates success, the Attribute Value is also returned.
	NodesToRead []*ReadValueID
}

// NewReadRequest creates a new ReadRequest.
func NewReadRequest(reqHeader *RequestHeader, maxAge uint64, tsRet TimestampsToReturn, nodes ...*ReadValueID) *ReadRequest {
	return &ReadRequest{
		RequestHeader:      reqHeader,
		MaxAge:             maxAge,
		TimestampsToReturn: tsRet,
		NodesToRead:        nodes,
	}
}
