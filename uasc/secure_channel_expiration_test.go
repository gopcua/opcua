// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"testing"

	"github.com/gopcua/opcua/uacp"
	"github.com/stretchr/testify/require"
)

// TestScheduleExpirationKeyedByChannelID verifies that a security token
// whose lifetime and grace window are over is dropped from the bucket its
// channel id maps to, since that is the key every arriving chunk is looked
// up under: a token is valid for its channel until it expires, and after
// that messages secured with it must no longer verify (OPC 10000-4 §5.5.2,
// OPC 10000-6 §6.7.2.4). A server that assigns the channel id and the
// token id different numbers must not leave the expired instance in place.
func TestScheduleExpirationKeyedByChannelID(t *testing.T) {
	sc := &SecureChannel{
		c:         &uacp.Conn{},
		closing:   make(chan struct{}),
		instances: make(map[uint32][]*channelInstance),
	}

	expired := &channelInstance{
		secureChannelID: 20,
		securityTokenID: 15,
	}
	renewed := &channelInstance{
		secureChannelID: 20,
		securityTokenID: 16,
	}
	other := &channelInstance{
		secureChannelID: 7,
		securityTokenID: 9,
	}
	sc.instances[20] = []*channelInstance{expired, renewed}
	sc.instances[7] = []*channelInstance{other}

	// revisedLifetime is zero, so the timer fires at once and
	// scheduleExpiration returns only after rewriting the bucket.
	sc.scheduleExpiration(expired)

	instances := sc.instances[20]
	require.Len(t, instances, 1, "channel 20 after the expired token's expiration")
	require.Equal(t, uint32(16), instances[0].securityTokenID, "channel 20 token id after expiration")
	require.Equal(t, uint32(20), instances[0].secureChannelID, "channel 20 instance channel id after expiration")
	require.Len(t, sc.instances[7], 1, "channel 7 after a token on channel 20 expired")
	require.Equal(t, uint32(9), sc.instances[7][0].securityTokenID, "channel 7 token id")
}
