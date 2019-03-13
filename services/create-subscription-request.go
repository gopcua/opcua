// Copyright 2018-2019 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

// CreateSubscriptionRequest is used to create a Subscription. Subscriptions monitor a set of MonitoredItems for
// Notifications and return them to the Client in response to Publish requests.
// Illegal request values for parameters that can be revised do not generate errors. Instead the
// Server will choose default values and indicate them in the corresponding revised parameter.
//
// Specification: Part 4, 5.13.2
type CreateSubscriptionRequest struct {
	RequestHeader               *RequestHeader
	RequestedPublishingInterval float64
	RequestedLifetimeCount      uint32
	RequestedMaxKeepAliveCount  uint32
	MaxNotificationsPerPublish  uint32
	PublishingEnabled           bool
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
		RequestHeader:               reqHeader,
		RequestedPublishingInterval: pubInterval,
		RequestedLifetimeCount:      lifetime,
		RequestedMaxKeepAliveCount:  keepAlive,
		MaxNotificationsPerPublish:  notifications,
		PublishingEnabled:           enabled,
		Priority:                    priority,
	}
}
