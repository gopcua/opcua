package services

import "github.com/wmnsk/gopcua/datatypes"

type CreateSubscriptionResponse struct {
	TypeID                    *datatypes.ExpandedNodeID
	ResponseHeader            *ResponseHeader
	SubscriptionID            uint32
	RevisedPublishingInterval float64
	RevisedLifetimeCount      uint32
	RevisedMaxKeepAliveCount  uint32
}

func (c *CreateSubscriptionResponse) ServiceType() uint16 {
	return ServiceTypeCreateSubscriptionResponse
}
