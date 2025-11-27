package server

import (
	"context"
	"crypto/rsa"
	"io"
	mrand "math/rand"
	"sync"
	"time"

	"github.com/gopcua/opcua/ua"
	"github.com/gopcua/opcua/uacp"
	"github.com/gopcua/opcua/ualog"
	"github.com/gopcua/opcua/uasc"
)

type channelBroker struct {
	endpoints map[string]*ua.EndpointDescription

	wg sync.WaitGroup

	// mu protects concurrent modification of s, secureChannelID, and secureTokenID
	mu sync.RWMutex
	// s is a slice of all SecureChannels watched by the channelBroker
	s map[uint32]*uasc.SecureChannel

	// Next Secure Channel ID to issue to a client
	secureChannelID uint32

	// Next Token ID to issue to a client
	secureTokenID uint32

	// msgChan is the common channel that all messages from all channels
	// get funneled into for handling
	msgChan chan *uasc.MessageBody
}

func newChannelBroker() *channelBroker {
	rng := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	return &channelBroker{
		endpoints:       make(map[string]*ua.EndpointDescription),
		s:               make(map[uint32]*uasc.SecureChannel),
		msgChan:         make(chan *uasc.MessageBody),
		secureChannelID: uint32(rng.Int31()),
		secureTokenID:   uint32(rng.Int31()),
	}
}

// RegisterConn connects a new UACP connection to the channel broker's list
// of connections and starts waiting for data on it.  Data is pushed onto the broker's
// Response channel
// Blocks until the context is done, the connection closes, or a critical error
func (c *channelBroker) RegisterConn(ctx context.Context, conn *uacp.Conn, localCert []byte, localKey *rsa.PrivateKey) error {
	cfg := defaultChannelConfig()
	cfg.Certificate = localCert
	cfg.LocalKey = localKey

	c.mu.Lock()
	c.secureChannelID++
	c.secureTokenID++
	secureChannelID := c.secureChannelID
	secureTokenID := c.secureTokenID
	sequenceNumber := uint32(mrand.Int31n(1023) + 1)
	c.mu.Unlock()

	errch := make(chan error, 1)
	sc, err := uasc.NewServerSecureChannel(
		"", // todo(fs): this is most likely wrong
		conn,
		cfg,
		errch,
		secureChannelID,
		sequenceNumber,
		secureTokenID,
	)
	if err != nil {
		ualog.Error(ctx, "could not create secure channel for new connection", ualog.Err(err))
		return err
	}

	c.mu.Lock()
	c.s[secureChannelID] = sc
	c.mu.Unlock()
	c.wg.Add(1)

	ctx = ualog.With(ctx, ualog.Uint32("channel", secureChannelID))
	ualog.Info(ctx, "registered new channel", ualog.Int("count", len(c.s)))

outer:
	for {
		select {
		case <-ctx.Done():
			// todo(fs): return error?
			ualog.Warn(ctx, "context done, closing secure channel")
			break outer

		default:
			msg := sc.Receive(ctx)
			if msg.Err == io.EOF {
				ualog.Warn(ctx, "secure channel closed")
				break outer
			} else if msg.Err != nil {
				ualog.Error(ctx, "secure channel error", ualog.Err(msg.Err))
				break outer
			}
			// todo(fs): honor ctx
			c.msgChan <- msg
		}
	}

	c.mu.Lock()
	delete(c.s, secureChannelID)
	c.mu.Unlock()
	c.wg.Done()

	return nil
}

// Close gracefully closes all secure channels
// todo(fs): use ctx
func (c *channelBroker) Close(ctx context.Context) error {
	var err error
	c.mu.Lock()
	for _, s := range c.s {
		s.Close()
	}
	c.mu.Unlock()

	// Wait for all goroutines to finish or timeout
	done := make(chan struct{})
	go func() {
		defer close(done)
		c.wg.Wait()
	}()

	channelExitTimeout := time.Duration(10 * time.Second) // todo(fs): magic number

	select {
	case <-done:
	case <-time.After(channelExitTimeout):
		ualog.Error(ctx, "timed out waiting for channels to exit",
			ualog.Duration("timeout", channelExitTimeout),
		)
	}

	return err
}

func (c *channelBroker) ReadMessage(ctx context.Context) *uasc.MessageBody {
	select {
	case <-ctx.Done():
		return nil
	case msg := <-c.msgChan:
		return msg
	}
}
