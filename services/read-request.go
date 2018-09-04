package services

import (
	"encoding/binary"

	"github.com/wmnsk/gopcua/datatypes"
)

type TimestampsToReturn uint32

// The TimestampsToReturn is an enumeration that specifies the Timestamp Attributes to be
// transmitted for MonitoredItems or Nodes in Read and HistoryRead.
//
// Specification: Part 4, 7.35
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
	TypeID *datatypes.ExpandedNodeID

	// Common request parameters.
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
	NodesToRead *datatypes.ReadValueIDArray
}

// DecodeReadRequest decodes given bytes into ReadRequest.
func DecodeReadRequest(b []byte) (*ReadRequest, error) {
	r := &ReadRequest{}
	if err := r.DecodeFromBytes(b); err != nil {
		return nil, err
	}
	return r, nil
}

// DecodeFromBytes decodes given bytes into ReadRequest.
func (r *ReadRequest) DecodeFromBytes(b []byte) error {
	offset := 0

	r.TypeID = &datatypes.ExpandedNodeID{}
	if err := r.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += r.TypeID.Len()

	// request header
	r.RequestHeader = &RequestHeader{}
	if err := r.RequestHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += r.RequestHeader.Len() - len(r.RequestHeader.Payload)

	// max age
	r.MaxAge = binary.LittleEndian.Uint64(b[offset : offset+8])
	offset += 8

	// timestamps to return
	r.TimestampsToReturn = TimestampsToReturn(binary.LittleEndian.Uint32(b[offset : offset+4]))
	offset += 4

	// nodes to read
	r.NodesToRead = &datatypes.ReadValueIDArray{}
	if err := r.NodesToRead.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}

	return nil
}

// Serialize serializes ReadRequest into bytes.
func (r *ReadRequest) Serialize() ([]byte, error) {
	b := make([]byte, r.Len())
	if err := r.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes ReadRequest into bytes.
func (r *ReadRequest) SerializeTo(b []byte) error {
	offset := 0

	// type id
	if err := r.TypeID.SerializeTo(b[offset:]); err != nil {
		return err
	}
	offset += r.TypeID.Len()

	// request header
	if err := r.RequestHeader.SerializeTo(b[offset:]); err != nil {
		return err
	}
	offset += r.RequestHeader.Len()

	// max age
	binary.LittleEndian.PutUint64(b[offset:offset+8], r.MaxAge)
	offset += 8

	// timestamps to return
	binary.LittleEndian.PutUint32(b[offset:offset+4], uint32(r.TimestampsToReturn))
	offset += 4

	// nodes to read
	return r.NodesToRead.SerializeTo(b[offset:])
}

// Len returns the actual length of ReadRequest.
func (r *ReadRequest) Len() int {
	// max age + timestamps to return
	length := 12

	// type id
	if r.TypeID != nil {
		length += r.TypeID.Len()
	}

	// request header
	if r.RequestHeader != nil {
		length += r.RequestHeader.Len()
	}

	// nodes to read
	if r.NodesToRead != nil {
		length += r.NodesToRead.Len()
	}

	return length
}

// ServiceType returns type of Service in uint16.
func (r *ReadRequest) ServiceType() uint16 {
	return ServiceTypeReadRequest
}
