package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/gopcua/opcua/ua"
)

// A subscription whose background task has returned has no reader left on NotifyChannel.
// ChangeNotification sends while holding the monitored item service lock, which the task's
// cleanup needs, so a send that blocks there deadlocks the server.
func TestNotifyDropsOnceTheSubscriptionTaskHasReturned(t *testing.T) {
	sub := NewSubscription()
	for range cap(sub.NotifyChannel) {
		sub.NotifyChannel <- &ua.MonitoredItemNotification{}
	}
	close(sub.done)

	returned := make(chan struct{})
	go func() {
		notify(sub, &ua.MonitoredItemNotification{ClientHandle: 1})
		close(returned)
	}()
	select {
	case <-returned:
	case <-time.After(5 * time.Second):
		t.Fatal("notify blocked on a full NotifyChannel after the subscription task returned")
	}
	require.Len(t, sub.NotifyChannel, cap(sub.NotifyChannel), "the dropped value must not displace queued ones")
}

func TestNotifyQueuesWhileTheSubscriptionTaskRuns(t *testing.T) {
	sub := NewSubscription()
	notify(sub, &ua.MonitoredItemNotification{ClientHandle: 7})
	require.Len(t, sub.NotifyChannel, 1)
	require.Equal(t, uint32(7), (<-sub.NotifyChannel).ClientHandle)
}
