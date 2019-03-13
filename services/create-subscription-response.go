package services

type CreateSubscriptionResponse struct {
	ResponseHeader            *ResponseHeader
	SubscriptionID            uint32
	RevisedPublishingInterval float64
	RevisedLifetimeCount      uint32
	RevisedMaxKeepAliveCount  uint32
}
