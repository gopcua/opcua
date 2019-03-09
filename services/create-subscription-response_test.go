// Copyright 2018 gopcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package services

import (
	"testing"
	"time"

	"github.com/wmnsk/gopcua/datatypes"
	"github.com/wmnsk/gopcua/utils/codectest"
)

func TestCreateSubscriptionResponse(t *testing.T) {
	cases := []codectest.Case{
		{
			Name: "normal",
			Struct: &CreateSubscriptionResponse{
				TypeID: datatypes.NewFourByteExpandedNodeID(0, ServiceTypeCreateSubscriptionResponse),
				ResponseHeader: NewResponseHeader(
					time.Date(2018, time.August, 10, 23, 0, 0, 0, time.UTC),
					1, 0, datatypes.NewNullDiagnosticInfo(), []string{}, NewNullAdditionalHeader(),
				),

				SubscriptionID:            1,
				RevisedPublishingInterval: 1000,
				RevisedLifetimeCount:      60,
				RevisedMaxKeepAliveCount:  20,
			},
			Bytes: []byte{
				// TypeID
				0x01, 0x00, 0x16, 0x03,
				// Timestamp
				0x00, 0x98, 0x67, 0xdd, 0xfd, 0x30, 0xd4, 0x01,
				// RequestHandle
				0x01, 0x00, 0x00, 0x00,
				// ServiceResult
				0x00, 0x00, 0x00, 0x00,
				// ServiceDiagnostics
				0x00,
				// StringTable
				0x00, 0x00, 0x00, 0x00,
				// AdditionalHeader
				0x00, 0x00, 0x00,
				// SubscriptionID
				0x01, 0x00, 0x00, 0x00,
				// RevisedPublishingInterval
				0x00, 0x00, 0x00, 0x00, 0x00, 0x40, 0x8f, 0x40,
				// RevisedLifetimeCount
				0x3c, 0x00, 0x00, 0x00,
				// RevisedMaxKeepAliveCount
				0x14, 0x00, 0x00, 0x00,
			},
		},
	}
	codectest.Run(t, cases)

	t.Run("service-id", func(t *testing.T) {
		id := new(CreateSubscriptionResponse).ServiceType()
		if got, want := id, uint16(ServiceTypeCreateSubscriptionResponse); got != want {
			t.Fatalf("got %d want %d", got, want)
		}
	})
}
