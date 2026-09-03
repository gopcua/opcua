// Copyright 2018-2020 opcua authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package uasc

import (
	"errors"
	"io"
	"testing"

	"github.com/gopcua/opcua/ua"
	"github.com/stretchr/testify/require"
)

// A response whose header carries a Bad ServiceResult is the answer to the
// request that drew it. It reaches the request's handler through the
// dispatcher, and it reaches the client's monitor — which reconnects — only
// when the status says the channel or the session is gone.
func TestMessageBodyIsChannelError(t *testing.T) {
	cases := []struct {
		name string
		msg  MessageBody
		want bool
	}{
		{"transport EOF", MessageBody{Err: io.EOF}, true},
		{"decoding error", MessageBody{Err: errors.New("too many chunks")}, true},
		{"UACP ERR status is not a service fault", MessageBody{Err: ua.StatusBadTCPSecureChannelUnknown}, true},
		{"service fault: too many operations", MessageBody{Err: ua.StatusBadTooManyOperations, serviceFault: true}, false},
		{"service fault: service unsupported", MessageBody{Err: ua.StatusBadServiceUnsupported, serviceFault: true}, false},
		{"service fault: user access denied", MessageBody{Err: ua.StatusBadUserAccessDenied, serviceFault: true}, false},
		{"service fault: secure channel id invalid", MessageBody{Err: ua.StatusBadSecureChannelIDInvalid, serviceFault: true}, true},
		{"service fault: session id invalid", MessageBody{Err: ua.StatusBadSessionIDInvalid, serviceFault: true}, true},
		{"service fault: session not activated", MessageBody{Err: ua.StatusBadSessionNotActivated, serviceFault: true}, true},
		{"service fault: subscription id invalid", MessageBody{Err: ua.StatusBadSubscriptionIDInvalid, serviceFault: true}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, c.msg.isChannelError())
		})
	}
}

func TestChannelLost(t *testing.T) {
	for _, code := range []ua.StatusCode{
		ua.StatusBadSecureChannelIDInvalid,
		ua.StatusBadSecureChannelClosed,
		ua.StatusBadSecureChannelTokenUnknown,
		ua.StatusBadSessionIDInvalid,
		ua.StatusBadSessionClosed,
		ua.StatusBadSessionNotActivated,
		ua.StatusBadSubscriptionIDInvalid,
		ua.StatusBadCertificateInvalid,
		ua.StatusBadConnectionClosed,
	} {
		require.True(t, channelLost(code), "%s", code)
	}
	for _, code := range []ua.StatusCode{
		ua.StatusBadTooManyOperations,
		ua.StatusBadServiceUnsupported,
		ua.StatusBadUserAccessDenied,
		ua.StatusBadNodeIDUnknown,
		ua.StatusBadTimeout,
		ua.StatusBadInternalError,
	} {
		require.False(t, channelLost(code), "%s", code)
	}
}
