// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"encoding/binary"
	"math"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/id"
)

// Specification: Part 4, 5.13.2.1
type CreateSubscriptionRequest struct {
	TypeID *datatypes.ExpandedNodeID
	*RequestHeader
	RequestedPublishingInterval float64
	RequestedLifetimeCount      uint32
	RequestedMaxKeepAliveCount  uint32
	MaxNotificationsPerPublish  uint32
	PublishingEnabled           *datatypes.Boolean
	Priority                    byte
}

// NewCreateSubscriptionRequest creates a new CreateSubscriptionRequest with the given parameters.
func NewCreateSubscriptionRequest(
	reqHeader *RequestHeader,
	pubInterval float64,
	lifetime uint32,
	keepAlive uint32,
	notifications uint32,
	enabled bool,
	priority byte,
) *CreateSubscriptionRequest {
	return &CreateSubscriptionRequest{
		TypeID: datatypes.NewExpandedNodeID(
			false, false,
			datatypes.NewFourByteNodeID(0, id.CreateSubscriptionRequest_Encoding_DefaultBinary),
			"", 0,
		),
		RequestHeader:               reqHeader,
		RequestedPublishingInterval: pubInterval,
		RequestedLifetimeCount:      lifetime,
		RequestedMaxKeepAliveCount:  keepAlive,
		MaxNotificationsPerPublish:  notifications,
		PublishingEnabled:           datatypes.NewBoolean(enabled),
		Priority:                    priority,
	}
}

// DecodeCreateSubscriptionRequest decodes given bytes into CreateSubscriptionRequest.
func DecodeCreateSubscriptionRequest(b []byte) (*CreateSubscriptionRequest, error) {
	c := &CreateSubscriptionRequest{}
	if err := c.DecodeFromBytes(b); err != nil {
		return nil, err
	}

	return c, nil
}

// DecodeFromBytes decodes given bytes into CreateSubscriptionRequest.
func (c *CreateSubscriptionRequest) DecodeFromBytes(b []byte) error {
	offset := 0

	// type id
	c.TypeID = &datatypes.ExpandedNodeID{}
	if err := c.TypeID.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.TypeID.Len()

	// request header
	c.RequestHeader = &RequestHeader{}
	if err := c.RequestHeader.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.RequestHeader.Len() - len(c.RequestHeader.Payload)

	// requested publishing interval
	rpi := binary.LittleEndian.Uint64(b[offset : offset+8])
	c.RequestedPublishingInterval = math.Float64frombits(rpi)
	offset += 8

	// requested lifetime count
	c.RequestedLifetimeCount = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	// requested max keep alive count
	c.RequestedMaxKeepAliveCount = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	// max notifications per publish
	c.MaxNotificationsPerPublish = binary.LittleEndian.Uint32(b[offset : offset+4])
	offset += 4

	// publishing enabled
	c.PublishingEnabled = &datatypes.Boolean{}
	if err := c.PublishingEnabled.DecodeFromBytes(b[offset:]); err != nil {
		return err
	}
	offset += c.PublishingEnabled.Len()

	// priority
	c.Priority = b[offset]

	return nil
}

// Serialize serializes CreateSubscriptionRequest into bytes.
func (c *CreateSubscriptionRequest) Serialize() ([]byte, error) {
	b := make([]byte, c.Len())
	if err := c.SerializeTo(b); err != nil {
		return nil, err
	}

	return b, nil
}

// SerializeTo serializes CreateSubscriptionRequest into bytes.
func (c *CreateSubscriptionRequest) SerializeTo(b []byte) error {
	var offset = 0
	if c.TypeID != nil {
		if err := c.TypeID.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.TypeID.Len()
	}

	if c.RequestHeader != nil {
		if err := c.RequestHeader.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.RequestHeader.Len() - len(c.Payload)
	}

	// requested publishing interval
	bits := math.Float64bits(c.RequestedPublishingInterval)
	binary.LittleEndian.PutUint64(b[offset:offset+8], bits)
	offset += 8

	// requested lifetime count
	binary.LittleEndian.PutUint32(b[offset:offset+4], c.RequestedLifetimeCount)
	offset += 4

	// requested max keep alive count
	binary.LittleEndian.PutUint32(b[offset:offset+4], c.RequestedMaxKeepAliveCount)
	offset += 4

	// max notifications per publish
	binary.LittleEndian.PutUint32(b[offset:offset+4], c.MaxNotificationsPerPublish)
	offset += 4

	// publishing enabled
	if c.PublishingEnabled != nil {
		if err := c.PublishingEnabled.SerializeTo(b[offset:]); err != nil {
			return err
		}
		offset += c.PublishingEnabled.Len()
	}

	// priority
	b[offset] = c.Priority

	return nil
}

// Len returns the actual length of CreateSubscriptionRequest in int.
func (c *CreateSubscriptionRequest) Len() int {
	length := 21

	if c.TypeID != nil {
		length += c.TypeID.Len()
	}

	if c.RequestHeader != nil {
		length += c.RequestHeader.Len()
	}

	if c.PublishingEnabled != nil {
		length += c.PublishingEnabled.Len()
	}

	return length
}

// ServiceType returns type of Service in uint16.
func (c *CreateSubscriptionRequest) ServiceType() uint16 {
	return id.CreateSubscriptionRequest_Encoding_DefaultBinary
}
