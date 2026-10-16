package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/stretchr/testify/require"
)

// TestCloseWhileClientsConnect closes the server while clients keep connecting. Run with
// -race: RegisterConn must not call wg.Add concurrently with Close's wg.Wait, and no secure
// channel may be registered after Close.
func TestCloseWhileClientsConnect(t *testing.T) {
	for i := 0; i < 10; i++ {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		port := l.Addr().(*net.TCPAddr).Port
		require.NoError(t, l.Close())

		ctx, cancel := context.WithCancel(context.Background())
		s := New(
			EndPoint("127.0.0.1", port),
			EnableSecurity("None", ua.MessageSecurityModeNone),
			EnableAuthMode(ua.UserTokenTypeAnonymous),
		)
		require.NoError(t, s.Start(ctx))

		url := fmt.Sprintf("opc.tcp://127.0.0.1:%d", port)
		stop := make(chan struct{})
		var wg sync.WaitGroup
		for j := 0; j < 8; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-stop:
						return
					default:
					}
					dctx, dcancel := context.WithTimeout(ctx, time.Second)
					c, err := uacp.Dial(dctx, url)
					dcancel()
					if err == nil {
						c.Close()
					}
				}
			}()
		}
		time.Sleep(20 * time.Millisecond)
		require.NoError(t, s.Close())
		close(stop)
		wg.Wait()
		cancel()

		s.cb.mu.RLock()
		n := len(s.cb.s)
		s.cb.mu.RUnlock()
		require.Zero(t, n, "secure channels registered after Close")
	}
}
